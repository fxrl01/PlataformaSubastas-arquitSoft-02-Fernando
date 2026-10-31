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
	redis    *redis.Client
	upgrader websocket.Upgrader
}

func NewWebSocketHandler(issuer *security.JWTIssuer, users domain.UserRepository, auctions domain.SubastaRepository, client *redis.Client) *WebSocketHandler {
	return &WebSocketHandler{
		issuer: issuer, users: users, auctions: auctions, redis: client,
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
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	claims, err := h.issuer.Verify(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
	if err != nil {
		http.Error(w, "invalid authentication token", http.StatusUnauthorized)
		return
	}
	user, err := h.users.FindByID(r.Context(), claims.UserID)
	if err != nil || user.Status != domain.UserActive || user.Role != claims.Role {
		http.Error(w, "invalid authentication token", http.StatusUnauthorized)
		return
	}
	if _, err := h.auctions.ObtenerPorID(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, "auction not found", http.StatusNotFound)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	channel := RedisChannelPrefix() + r.PathValue("id")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		return
	}

	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		message, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return
		}
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message.Payload)); err != nil {
			return
		}
	}
}
