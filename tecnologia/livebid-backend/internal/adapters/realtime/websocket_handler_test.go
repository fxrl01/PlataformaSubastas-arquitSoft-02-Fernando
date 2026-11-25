package realtime_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/realtime"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type wsUserRepo struct{ user *domain.User }

func (r wsUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r wsUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}
func (r wsUserRepo) Save(context.Context, *domain.User) error                       { return nil }
func (r wsUserRepo) UpdateStatus(context.Context, string, domain.UserStatus) error { return nil }

type wsAuctionRepo struct{ auction *domain.Subasta }

func (r wsAuctionRepo) ObtenerPorID(_ context.Context, id string) (*domain.Subasta, error) {
	if r.auction == nil || r.auction.ID != id {
		return nil, domain.ErrSubastaNoEncontrada
	}
	return r.auction, nil
}
func (r wsAuctionRepo) Guardar(context.Context, *domain.Subasta) error    { return nil }
func (r wsAuctionRepo) Listar(context.Context) ([]*domain.Subasta, error) { return nil, nil }

func TestWebSocketHandlerRejectsUnauthorizedAndUpgradesValidConnection(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("ws-usr-1", "WS User", "ws@livebid.pe", "hash", now)
	user.Status = domain.UserActive
	auction := domain.NuevaSubasta("sub-ws-1", "Lote WS", "", 1000, now, now.Add(time.Hour))

	issuer, _ := security.NewJWTIssuer("ws-test-secret")
	token, _ := issuer.Issue(domain.TokenClaims{UserID: user.ID, Role: user.Role}, now.Add(time.Minute))

	wsHandler := realtime.NewWebSocketHandler(issuer, wsUserRepo{user: user}, wsAuctionRepo{auction: auction}, nil)
	mux := http.NewServeMux()
	mux.Handle("GET /ws/subastas/{id}", wsHandler)

	// 1. Sin token -> 401
	req := httptest.NewRequest(http.MethodGet, "/ws/subastas/sub-ws-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// 2. Subasta inexistente -> 404; token inválido -> 401; usuario no encontrado -> 401; Origin cruzado -> 403
	reqBadTok := httptest.NewRequest(http.MethodGet, "/ws/subastas/sub-ws-1?token=bad.jwt.tok", nil)
	recBadTok := httptest.NewRecorder()
	mux.ServeHTTP(recBadTok, reqBadTok)
	if recBadTok.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad token, got %d", recBadTok.Code)
	}

	unknownTok, _ := issuer.Issue(domain.TokenClaims{UserID: "ghost", Role: domain.RoleBidder}, now.Add(time.Minute))
	reqGhost := httptest.NewRequest(http.MethodGet, "/ws/subastas/sub-ws-1", nil)
	reqGhost.Header.Set("Authorization", "Bearer "+unknownTok)
	recGhost := httptest.NewRecorder()
	mux.ServeHTTP(recGhost, reqGhost)
	if recGhost.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown user, got %d", recGhost.Code)
	}

	req404 := httptest.NewRequest(http.MethodGet, "/ws/subastas/unknown?token="+token, nil)
	rec404 := httptest.NewRecorder()
	mux.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown auction, got %d", rec404.Code)
	}

	// 3. Handshake WebSocket real sobre httptest.NewServer
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/subastas/sub-ws-1?token=" + token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v (resp=%v)", err, resp)
	}
	_ = conn.Close()

	// 4. Si Redis local está disponible, verificar entrega de mensajes en vivo vía WebSocketHandler + RedisBidNotifier
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err == nil {
		wsRedisHandler := realtime.NewWebSocketHandler(issuer, wsUserRepo{user: user}, wsAuctionRepo{auction: auction}, rdb)
		muxRedis := http.NewServeMux()
		muxRedis.Handle("GET /ws/subastas/{id}", wsRedisHandler)
		srvRedis := httptest.NewServer(muxRedis)
		defer srvRedis.Close()

		wsRedisURL := "ws" + strings.TrimPrefix(srvRedis.URL, "http") + "/ws/subastas/sub-ws-1?token=" + token
		c, _, err := websocket.DefaultDialer.Dial(wsRedisURL, nil)
		if err != nil {
			t.Fatalf("websocket redis dial failed: %v", err)
		}
		defer c.Close()

		time.Sleep(40 * time.Millisecond)
		notifier := realtime.NewRedisBidNotifier(rdb)
		_ = notifier.BroadcastNuevaPuja(context.Background(), "sub-ws-1", domain.NuevaPuja("b-ws-1", "sub-ws-1", user.ID, 1250, now), false, now.Add(time.Hour))
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := c.ReadMessage()
		if err != nil || !strings.Contains(string(msg), "b-ws-1") {
			t.Fatalf("expected live ws bid payload, got msg=%q err=%v", string(msg), err)
		}

		// BroadcastSubastaCerrada con y sin ganador
		_ = notifier.BroadcastSubastaCerrada(context.Background(), "sub-ws-1", nil)
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, closedMsg, err := c.ReadMessage()
		if err != nil || !strings.Contains(string(closedMsg), "auction.closed") {
			t.Fatalf("expected auction.closed payload, got msg=%q err=%v", string(closedMsg), err)
		}
		winner := domain.NuevaPuja("b-ws-win", "sub-ws-1", user.ID, 1500, now)
		_ = notifier.BroadcastSubastaCerrada(context.Background(), "sub-ws-1", &winner)
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, closedWinMsg, err := c.ReadMessage()
		if err != nil || !strings.Contains(string(closedWinMsg), user.ID) {
			t.Fatalf("expected winner in auction.closed payload, got msg=%q err=%v", string(closedWinMsg), err)
		}
	}
}
