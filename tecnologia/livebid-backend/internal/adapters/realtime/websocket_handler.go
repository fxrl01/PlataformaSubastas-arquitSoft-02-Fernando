package realtime

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type WebSocketHandler struct {
	issuer   *security.JWTIssuer
	users    domain.UserRepository
	auctions domain.SubastaRepository
	hub      *AuctionRoomHub
	upgrader websocket.Upgrader
}

func NewWebSocketHandler(issuer *security.JWTIssuer, users domain.UserRepository, auctions domain.SubastaRepository, client *redis.Client) *WebSocketHandler {
	var hub *AuctionRoomHub
	if client != nil {
		hub = NewAuctionRoomHub(NewRedisSubscriber(client))
	}
	return &WebSocketHandler{
		issuer: issuer, users: users, auctions: auctions, hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				parsed, err := url.Parse(origin)
				return err == nil && parsed.Host == r.Host
			},
		},
	}
}

func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawToken := extractBearerOrQueryToken(r)
	if rawToken == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	claims, err := h.issuer.Verify(rawToken)
	if err != nil {
		http.Error(w, "invalid authentication token", http.StatusUnauthorized)
		return
	}
	user, err := h.users.FindByID(r.Context(), claims.UserID)
	if err != nil || user.Status != domain.UserActive || user.Role != claims.Role {
		http.Error(w, "invalid authentication token", http.StatusUnauthorized)
		return
	}
	auctionID := r.PathValue("id")
	if _, err := h.auctions.ObtenerPorID(r.Context(), auctionID); err != nil {
		http.Error(w, "auction not found", http.StatusNotFound)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if h.hub == nil {
		return
	}
	messages, leave, err := h.hub.Join(auctionID)
	if err != nil {
		return
	}
	defer leave()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-messages:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		}
	}
}

func extractBearerOrQueryToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}
