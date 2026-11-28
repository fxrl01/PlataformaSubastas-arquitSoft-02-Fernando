package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type spyRoomNotifier struct {
	events []domain.RoomMessage
}

func (s *spyRoomNotifier) BroadcastEventoSocialSala(_ context.Context, _ string, msg domain.RoomMessage) error {
	s.events = append(s.events, msg)
	return nil
}

func TestSocialHandlerLifecycleStorefrontRedactionAndRoomChat(t *testing.T) {
	now := time.Now()
	seller, err := domain.NewUser("usr_seller_01", "Valeria Colecciones", "valeria.privado@correo.pe", "hash", now)
	if err != nil {
		t.Fatal(err)
	}
	seller.Status = domain.UserActive
	seller.Role = domain.RoleSeller
	seller.KYCApproved = true
	seller.Bio = "Especialista en relojería clásica certificada."
	seller.Phone = "+51 999 888 777"
	seller.DocumentID = "DNI 09876543"

	issuer, err := security.NewJWTIssuer("test-social-secret-key-2026")
	if err != nil {
		t.Fatal(err)
	}
	token, err := issuer.Issue(domain.TokenClaims{UserID: seller.ID, Role: seller.Role}, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	notifier := &spyRoomNotifier{}
	socialHandler := handlers.NewSocialHandler(nil, middlewareUserRepo{user: seller}).WithRoomRealtime(nil, notifier)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/social/posts", socialHandler.ListPosts)
	mux.Handle("POST /api/v1/social/posts", handlers.RequireAuth(issuer, middlewareUserRepo{user: seller}, http.HandlerFunc(socialHandler.CreatePost)))
	mux.Handle("POST /api/v1/social/posts/{id}/like", handlers.RequireAuth(issuer, middlewareUserRepo{user: seller}, http.HandlerFunc(socialHandler.LikePost)))
	mux.Handle("POST /api/v1/social/posts/{id}/comentarios", handlers.RequireAuth(issuer, middlewareUserRepo{user: seller}, http.HandlerFunc(socialHandler.AddComment)))
	mux.HandleFunc("GET /api/v1/vendedores/{id}/vitrina", socialHandler.GetSellerStorefront)
	mux.HandleFunc("GET /api/v1/subastas/{id}/mensajes", socialHandler.ListRoomMessages)
	mux.Handle("POST /api/v1/subastas/{id}/mensajes", handlers.RequireAuth(issuer, middlewareUserRepo{user: seller}, http.HandlerFunc(socialHandler.PostRoomMessage)))

	// 1. Crear publicación social autenticada
	postBody := bytes.NewBufferString(`{"contenido":"Nuevo lote disponible con certificado pericial","etiqueta":"relojeria","subasta_id":"SUB-2026-001"}`)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/social/posts", postBody)
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	recCreate := httptest.NewRecorder()
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating post, got %d: %s", recCreate.Code, recCreate.Body.String())
	}

	var createdPost domain.SocialPost
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createdPost); err != nil {
		t.Fatal(err)
	}
	if createdPost.Tag != "RELOJERIA" || createdPost.AuthorName != "Valeria Colecciones" {
		t.Fatalf("unexpected created post fields: %+v", createdPost)
	}

	// 2. Dar like y comentar publicación
	reqLike := httptest.NewRequest(http.MethodPost, "/api/v1/social/posts/"+createdPost.ID+"/like", nil)
	reqLike.Header.Set("Authorization", "Bearer "+token)
	recLike := httptest.NewRecorder()
	mux.ServeHTTP(recLike, reqLike)
	if recLike.Code != http.StatusOK {
		t.Fatalf("expected 200 liking post, got %d", recLike.Code)
	}

	reqCmt := httptest.NewRequest(http.MethodPost, "/api/v1/social/posts/"+createdPost.ID+"/comentarios", bytes.NewBufferString(`{"comentario":"Excelente pieza de colección"}`))
	reqCmt.Header.Set("Authorization", "Bearer "+token)
	recCmt := httptest.NewRecorder()
	mux.ServeHTTP(recCmt, reqCmt)
	if recCmt.Code != http.StatusCreated {
		t.Fatalf("expected 201 adding comment, got %d", recCmt.Code)
	}

	// 3. Verificar vitrina pública sanitizada (NO expone email, teléfono ni documento)
	reqStore := httptest.NewRequest(http.MethodGet, "/api/v1/vendedores/"+seller.ID+"/vitrina", nil)
	recStore := httptest.NewRecorder()
	mux.ServeHTTP(recStore, reqStore)
	if recStore.Code != http.StatusOK {
		t.Fatalf("expected 200 on storefront, got %d", recStore.Code)
	}
	rawStore := recStore.Body.String()
	if strings.Contains(rawStore, "valeria.privado@correo.pe") || strings.Contains(rawStore, "09876543") || strings.Contains(rawStore, "999 888 777") {
		t.Fatalf("PII leaked in seller storefront: %s", rawStore)
	}
	if !strings.Contains(rawStore, "Valeria Colecciones") {
		t.Fatalf("expected public seller name in storefront: %s", rawStore)
	}

	// 4. Chat en vivo de sala (persiste CHAT con paleta anónima y transmite REACTION efímera)
	reqChat := httptest.NewRequest(http.MethodPost, "/api/v1/subastas/SUB-2026-001/mensajes", bytes.NewBufferString(`{"tipo":"CHAT","mensaje":"¡Atentos al cierre!"}`))
	reqChat.Header.Set("Authorization", "Bearer "+token)
	recChat := httptest.NewRecorder()
	mux.ServeHTTP(recChat, reqChat)
	if recChat.Code != http.StatusCreated {
		t.Fatalf("expected 201 on room chat, got %d: %s", recChat.Code, recChat.Body.String())
	}
	if !strings.Contains(recChat.Body.String(), "Paleta #") {
		t.Fatalf("expected anonymized paddle label in room chat: %s", recChat.Body.String())
	}

	reqReact := httptest.NewRequest(http.MethodPost, "/api/v1/subastas/SUB-2026-001/mensajes", bytes.NewBufferString(`{"tipo":"REACTION","mensaje":"🔥"}`))
	reqReact.Header.Set("Authorization", "Bearer "+token)
	recReact := httptest.NewRecorder()
	mux.ServeHTTP(recReact, reqReact)
	if recReact.Code != http.StatusCreated || len(notifier.events) != 2 {
		t.Fatalf("expected 201 and 2 broadcast events, got code=%d events=%d", recReact.Code, len(notifier.events))
	}

	reqListRoom := httptest.NewRequest(http.MethodGet, "/api/v1/subastas/SUB-2026-001/mensajes", nil)
	recListRoom := httptest.NewRecorder()
	mux.ServeHTTP(recListRoom, reqListRoom)
	if recListRoom.Code != http.StatusOK || !strings.Contains(recListRoom.Body.String(), "¡Atentos al cierre!") {
		t.Fatalf("expected persisted CHAT message in history, got %d: %s", recListRoom.Code, recListRoom.Body.String())
	}
}
