package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

func TestHandlersErrorBranchesAndValidation(t *testing.T) {
	mux, _ := buildTestRouter(t)

	// 1. Auth errors: register invalid password, duplicate email, invalid login, bad refresh
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Short", "email": "short@livebid.pe", "password": "123",
	}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Dup", "email": "admin@livebid.pe", "password": "Password123!",
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate email, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "admin@livebid.pe", "password": "wrong-password",
	}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong login, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": "non-existent-refresh",
	}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid refresh token, got %d", rec.Code)
	}

	// Login Admin para probar validaciones de rutas protegidas
	adminLogin := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "admin@livebid.pe", "password": "AdminPass2026!",
	})
	var adminAuth services.AuthResponse
	_ = json.Unmarshal(adminLogin.Body.Bytes(), &adminAuth)

	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/admin/users?rol=INVALID", adminAuth.AccessToken, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid role filter, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/nope/activate", adminAuth.AccessToken, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 activating unknown user, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/nope/role", adminAuth.AccessToken, map[string]string{"rol": "NOPE"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid role, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/nope/role", adminAuth.AccessToken, map[string]string{"rol": "POSTOR"}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown user role update, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/nope/status", adminAuth.AccessToken, map[string]string{"estado": "NOPE"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid status, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/nope/status", adminAuth.AccessToken, map[string]string{"estado": "ACTIVO"}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown user status update, got %d", rec.Code)
	}

	// 2. Auction errors: invalid create, unknown encargado/lote/transition/audit
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas", adminAuth.AccessToken, map[string]any{
		"titulo": "", "precio_base": -10,
	}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid auction create, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/nope/encargado", adminAuth.AccessToken, map[string]string{"nombre": "X"}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown auction encargado, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/nope/detalles-lote", adminAuth.AccessToken, map[string]any{"categoria": "X"}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown auction lot details, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/nope/activate", adminAuth.AccessToken, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown auction activate, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/nope/auditoria-web3", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 unknown auction audit, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/nope/auditoria-web3?indice=-1", "", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 negative merkle leaf index, got %d", rec.Code)
	}

	// Crear subasta válida y probar conflictos de estado, garantía insuficiente, puja inválida, AI errors
	cRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas", adminAuth.AccessToken, map[string]any{
		"titulo": "Lote Errors", "precio_base": 1000, "inicio": time.Now(), "cierre": time.Now().Add(time.Hour),
	})
	var created map[string]any
	_ = json.Unmarshal(cRes.Body.Bytes(), &created)
	aID := created["id"].(string)

	// No se puede iniciar conteo en PENDIENTE
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/"+aID+"/countdown", adminAuth.AccessToken, nil); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 countdown from pending, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/"+aID+"/encargado", adminAuth.AccessToken, map[string]string{}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 empty encargado, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/"+aID+"/detalles-lote", adminAuth.AccessToken, map[string]any{"precio_reserva": -5}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 negative reserve, got %d", rec.Code)
	}

	// Garantía inexistente -> 404; monto insuficiente -> 409; stripe intent subasta no existe -> 404
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+aID+"/garantias/me", adminAuth.AccessToken, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 no guarantee, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/nope/garantias/stripe-intent", adminAuth.AccessToken, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 stripe-intent unknown auction, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+aID+"/garantias", adminAuth.AccessToken, map[string]any{"monto": 10}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 insufficient guarantee, got %d", rec.Code)
	}

	// Webhook sin firma -> 400
	whReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(`{"type":"ping"}`))
	whRec := httptest.NewRecorder()
	mux.ServeHTTP(whRec, whReq)
	if whRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid stripe webhook signature, got %d", whRec.Code)
	}

	// Puja sin garantía -> 409; delegación en subasta cerrada/pendiente -> 409; switch a modelo desconocido -> 400
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+aID+"/agentes-ia/delegar", adminAuth.AccessToken, map[string]any{
		"agente_id": "a1", "monto_maximo_autorizado": 5000,
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 delegate on pending auction, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/ia/modelo", adminAuth.AccessToken, map[string]string{"modelo": "modelo-inexistente"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 unknown ai model, got %d", rec.Code)
	}
	_ = doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/"+aID+"/activate", adminAuth.AccessToken, nil)
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+aID+"/pujas", adminAuth.AccessToken, map[string]any{"monto": 1200}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 bid without guarantee, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+aID+"/agentes-ia/auto-puja", adminAuth.AccessToken, map[string]string{"mandato_id": "nope"}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 invalid mandate, got %d", rec.Code)
	}

	// RequireRoles sin contexto autenticado -> 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	unauthRec := httptest.NewRecorder()
	handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from RequireRoles without user, got %d", unauthRec.Code)
	}

	// Handler nil-dependency / unconfigured branches + Web3 headers en BidHandler + AuditWeb3 nil
	nilAuditH := handlers.NewAuctionHandler(nil)
	recAuditNil := httptest.NewRecorder()
	nilAuditH.AuditWeb3(recAuditNil, httptest.NewRequest(http.MethodGet, "/api/v1/subastas/1/auditoria-web3", nil))
	if recAuditNil.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when web3Audit is nil, got %d", recAuditNil.Code)
	}

	nilAIH := handlers.NewAgentAIHandler(nil, nil, nil, nil)
	recRepNil := httptest.NewRecorder()
	nilAIH.IntelligenceReport(recRepNil, httptest.NewRequest(http.MethodGet, "/api/v1/subastas/1/inteligencia-ia", nil))
	if recRepNil.Code != http.StatusOK {
		t.Fatalf("expected 200 empty intelligence report when deps nil, got %d", recRepNil.Code)
	}
	recSwitchNil := doJSONReq(t, http.HandlerFunc(nilAIH.SwitchModel), http.MethodPatch, "/api/v1/admin/ia/modelo", "", map[string]string{"modelo": "x"})
	if recSwitchNil.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when ai router is nil, got %d", recSwitchNil.Code)
	}

	// BidHandler con headers X-Web3-* inválidos
	web3BidReq := httptest.NewRequest(http.MethodPost, "/api/v1/subastas/"+aID+"/pujas", bytes.NewBufferString(`{"monto":1500}`))
	web3BidReq.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)
	web3BidReq.Header.Set("X-Web3-Wallet", "bad-wallet")
	web3BidReq.Header.Set("X-Web3-Signature", "0x1234")
	web3BidReq.Header.Set("X-Web3-Nonce", "n1")
	web3BidReq.Header.Set("X-Web3-Deadline", "9999999999")
	web3BidReq.Header.Set("X-AI-Agent-Mandate", "mnd_1")
	web3BidRec := httptest.NewRecorder()
	mux.ServeHTTP(web3BidRec, web3BidReq)
	if web3BidRec.Code == http.StatusCreated {
		t.Fatal("expected rejected bid with missing guarantee/mandate")
	}

	// AuthHandler.Me sin usuario en contexto -> 401
	meRec := httptest.NewRecorder()
	handlers.NewAuthHandler(nil).Me(meRec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on Me without user context, got %d", meRec.Code)
	}
}
