package handlers_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type memUserRepo struct {
	users map[string]*domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[string]*domain.User)}
}

func (r *memUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *memUserRepo) Save(_ context.Context, u *domain.User) error {
	r.users[u.ID] = u
	return nil
}

func (r *memUserRepo) UpdateStatus(_ context.Context, id string, s domain.UserStatus) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = s
	return nil
}

func (r *memUserRepo) ListUsers(_ context.Context, status domain.UserStatus, role domain.UserRole) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if (status == "" || u.Status == status) && (role == "" || u.Role == role) {
			out = append(out, u)
		}
	}
	return out, nil
}

func (r *memUserRepo) UpdateRole(_ context.Context, id string, role domain.UserRole) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Role = role
	return nil
}

func (r *memUserRepo) UpdateKYCAndStatus(_ context.Context, id string, status domain.UserStatus, kyc bool) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = status
	u.KYCApproved = kyc
	return nil
}

type memRefreshStore struct {
	tokens map[string]string
}

func (s *memRefreshStore) Save(_ context.Context, hash, userID string, _ time.Time) error {
	s.tokens[hash] = userID
	return nil
}

func (s *memRefreshStore) Consume(_ context.Context, hash string) (string, error) {
	id, ok := s.tokens[hash]
	if !ok {
		return "", domain.ErrRefreshTokenInvalid
	}
	delete(s.tokens, hash)
	return id, nil
}

type memAuctionRepo struct {
	auctions map[string]*domain.Subasta
}

func (r *memAuctionRepo) ObtenerPorID(_ context.Context, id string) (*domain.Subasta, error) {
	s, ok := r.auctions[id]
	if !ok {
		return nil, domain.ErrSubastaNoEncontrada
	}
	cp := *s
	return &cp, nil
}

func (r *memAuctionRepo) Guardar(_ context.Context, s *domain.Subasta) error {
	cp := *s
	r.auctions[s.ID] = &cp
	return nil
}

func (r *memAuctionRepo) Listar(_ context.Context) ([]*domain.Subasta, error) {
	out := make([]*domain.Subasta, 0, len(r.auctions))
	for _, s := range r.auctions {
		cp := *s
		out = append(out, &cp)
	}
	return out, nil
}

type memBidRepo struct {
	auctions *memAuctionRepo
	bids     []domain.Puja
	hashes   []string
	chain    []domain.AuditLogEntry
	receipt  *domain.OnChainSettlementReceipt
}

func (r *memBidRepo) BuscarPorIdempotencyKey(_ context.Context, subastaID, bidderID, key string) (*domain.Puja, error) {
	return nil, nil
}

func (r *memBidRepo) RegistrarPujaAtomica(_ context.Context, subastaID string, puja domain.Puja, _ float64, fechaCierre time.Time, _ bool, _ string) error {
	if s, ok := r.auctions.auctions[subastaID]; ok {
		s.PrecioActual = puja.Monto
		s.PostorLiderID = puja.PostorID
		s.FechaCierre = fechaCierre
	}
	r.bids = append(r.bids, puja)
	prev := "GENESIS"
	if len(r.chain) > 0 {
		prev = r.chain[len(r.chain)-1].CurrentHash
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f", prev, puja.ID, puja.Monto)))
	h := hex.EncodeToString(sum[:])
	r.hashes = append(r.hashes, h)
	r.chain = append(r.chain, domain.AuditLogEntry{Event: "PUJA_REGISTRADA", PreviousHash: prev, CurrentHash: h})
	return nil
}

func (r *memBidRepo) ListarPorSubasta(_ context.Context, subastaID, _ string, _ float64, limit, offset int) ([]domain.Puja, int, error) {
	var filtered []domain.Puja
	for _, b := range r.bids {
		if b.SubastaID == subastaID {
			filtered = append(filtered, b)
		}
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], len(filtered), nil
}

func (r *memBidRepo) ObtenerHashesAuditoria(context.Context, string) ([]string, error) { return r.hashes, nil }
func (r *memBidRepo) ObtenerCadenaAuditoria(context.Context, string) ([]domain.AuditLogEntry, error) {
	return r.chain, nil
}
func (r *memBidRepo) GuardarReciboOnChain(_ context.Context, rec domain.OnChainSettlementReceipt) error {
	r.receipt = &rec
	return nil
}
func (r *memBidRepo) ObtenerReciboOnChain(context.Context, string) (*domain.OnChainSettlementReceipt, error) {
	return r.receipt, nil
}

type memGuaranteeRepo struct {
	items map[string]*domain.Guarantee
}

func (r *memGuaranteeRepo) FindActive(_ context.Context, auctionID, bidderID string) (*domain.Guarantee, error) {
	g, ok := r.items[auctionID+":"+bidderID]
	if !ok || g.Status != domain.GuaranteeHeld {
		return nil, domain.ErrUserNotFound
	}
	return g, nil
}
func (r *memGuaranteeRepo) Save(_ context.Context, g *domain.Guarantee) error {
	r.items[g.AuctionID+":"+g.BidderID] = g
	return nil
}
func (r *memGuaranteeRepo) SettleAuction(_ context.Context, auctionID, winnerID string) error {
	for k, g := range r.items {
		if g.AuctionID == auctionID && g.Status == domain.GuaranteeHeld {
			if winnerID != "" && g.BidderID == winnerID {
				r.items[k].Status = domain.GuaranteeCaptured
			} else {
				r.items[k].Status = domain.GuaranteeReleased
			}
		}
	}
	return nil
}

func buildTestRouter(t *testing.T) (http.Handler, *services.AuthService) {
	t.Helper()
	users := newMemUserRepo()
	issuer, err := security.NewJWTIssuer("handlers-test-jwt-secret-2026")
	if err != nil {
		t.Fatal(err)
	}
	authSvc := services.NewAuthService(users, security.NewBcryptHasher(4), issuer, &memRefreshStore{tokens: map[string]string{}})
	if _, err := authSvc.BootstrapAdmin(context.Background(), "Admin", "admin@livebid.pe", "AdminPass2026!"); err != nil {
		t.Fatal(err)
	}

	auctions := &memAuctionRepo{auctions: map[string]*domain.Subasta{}}
	bids := &memBidRepo{auctions: auctions}
	guarantees := &memGuaranteeRepo{items: map[string]*domain.Guarantee{}}

	gemini := ai.NewGeminiRiskAdapter("")
	breaker := ai.NewCircuitBreakerIA(gemini, 3, time.Second, time.Second)
	router := ai.NewMultiModelRouter("gemini-2.5-flash", breaker, time.Minute)
	router.RegisterModel("gemini-2.5-pro-forensic", breaker)
	detector := ai.NewAsyncFraudDetector(router)
	stripe := payments.NewStripeAdapter("", "pk_test_handler", "whsec_handler_test")
	escrow := web3.NewEVMEscrowAdapter(8453, "0x1111111111111111111111111111111111111111", "")

	catalogSvc := services.NuevoAuctionCatalogServiceConGarantias(auctions, guarantees).WithWeb3Settlement(bids, escrow)
	bidSvc := services.NewAuctionBidServiceWithNotifierAndGuarantee(auctions, bids, nil, nil, guarantees).WithFraudTelemetry(detector)
	guaranteeSvc := services.NewGuaranteeService(auctions, guarantees, router).WithPaymentGateway(stripe)
	historySvc := services.NewBidHistoryService(auctions, bids)
	auditSvc := services.NewWeb3AuditService(auctions, bids)
	agentSvc := services.NewAgentDelegationService(auctions, bidSvc)

	authH := handlers.NewAuthHandler(authSvc)
	auctionH := handlers.NewAuctionHandler(catalogSvc, auditSvc)
	bidH := handlers.NewBidHandler(bidSvc)
	historyH := handlers.NewBidHistoryHandler(historySvc)
	guaranteeH := handlers.NewGuaranteeHandler(guaranteeSvc)
	aiH := handlers.NewAgentAIHandler(agentSvc, router, breaker, detector).WithAssistant(ai.NewGeminiAssistant(""), auctions)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/ia/asistente", aiH.AssistantChat)
	mux.HandleFunc("POST /api/v1/auth/register", authH.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authH.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authH.Refresh)
	mux.Handle("GET /api/v1/auth/me", handlers.RequireAuth(issuer, users, http.HandlerFunc(authH.Me)))
	mux.Handle("GET /api/v1/admin/users", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authH.ListUsers))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/activate", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authH.ActivateUser))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/role", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authH.UpdateUserRole))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/status", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authH.UpdateUserStatus))))
	mux.Handle("PATCH /api/v1/admin/ia/modelo", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(aiH.SwitchModel))))

	mux.HandleFunc("GET /api/v1/subastas", auctionH.List)
	mux.HandleFunc("GET /api/v1/subastas/{id}", auctionH.Get)
	mux.HandleFunc("GET /api/v1/subastas/{id}/pujas", historyH.List)
	mux.HandleFunc("GET /api/v1/subastas/{id}/auditoria-web3", auctionH.AuditWeb3)
	mux.HandleFunc("GET /api/v1/subastas/{id}/inteligencia-ia", aiH.IntelligenceReport)
	mux.Handle("POST /api/v1/subastas", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleSeller, domain.RoleAdmin)(http.HandlerFunc(auctionH.Create))))
	mux.Handle("PUT /api/v1/subastas/{id}/encargado", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(auctionH.UpdateEncargado))))
	mux.Handle("PUT /api/v1/subastas/{id}/detalles-lote", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(auctionH.UpdateDetallesLote))))
	mux.Handle("PATCH /api/v1/subastas/{id}/activate", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(auctionH.Activate))))
	mux.Handle("PATCH /api/v1/subastas/{id}/countdown", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(auctionH.StartCountdown))))
	mux.Handle("PATCH /api/v1/subastas/{id}/close", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(auctionH.Close))))

	mux.Handle("POST /api/v1/subastas/{id}/pujas", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(bidH.Place))))
	mux.Handle("GET /api/v1/subastas/{id}/garantias/me", handlers.RequireAuth(issuer, users, http.HandlerFunc(guaranteeH.GetMyGuarantee)))
	mux.Handle("POST /api/v1/subastas/{id}/garantias/stripe-intent", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(guaranteeH.PrepareStripeIntent))))
	mux.Handle("POST /api/v1/subastas/{id}/garantias", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(guaranteeH.Create))))
	mux.HandleFunc("POST /api/v1/webhooks/stripe", guaranteeH.StripeWebhook)
	mux.Handle("POST /api/v1/subastas/{id}/agentes-ia/delegar", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(aiH.Delegate))))
	mux.Handle("POST /api/v1/subastas/{id}/agentes-ia/auto-puja", handlers.RequireAuth(issuer, users, handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(aiH.AutoBid))))

	return mux, authSvc
}

func doJSONReq(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if raw, ok := body.([]byte); ok {
			buf.Write(raw)
		} else if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandlersFullHTTPFlowAndRBAC(t *testing.T) {
	mux, _ := buildTestRouter(t)

	// 1. Login Admin y Registro/Activación de Postor
	adminLogin := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "admin@livebid.pe", "password": "AdminPass2026!",
	})
	if adminLogin.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", adminLogin.Code, adminLogin.Body.String())
	}
	var adminAuth services.AuthResponse
	_ = json.Unmarshal(adminLogin.Body.Bytes(), &adminAuth)

	regRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Postor HTTP", "email": "postor.http@livebid.pe", "password": "Password123!", "rol_solicitado": "POSTOR",
	})
	if regRes.Code != http.StatusCreated {
		t.Fatalf("register failed: %d %s", regRes.Code, regRes.Body.String())
	}
	var registered services.UserView
	_ = json.Unmarshal(regRes.Body.Bytes(), &registered)

	// Admin lista usuarios, aprueba KYC y actualiza rol/estado
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/admin/users?estado=PENDIENTE", adminAuth.AccessToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("list users failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/"+registered.ID+"/activate", adminAuth.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("activate user failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/"+registered.ID+"/role", adminAuth.AccessToken, map[string]string{"rol": "POSTOR"}); rec.Code != http.StatusOK {
		t.Fatalf("update role failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/users/"+registered.ID+"/status", adminAuth.AccessToken, map[string]any{"estado": "ACTIVO", "kyc_aprobado": true}); rec.Code != http.StatusOK {
		t.Fatalf("update status failed: %d", rec.Code)
	}

	// Postor login + refresh + /me
	bidderLogin := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "postor.http@livebid.pe", "password": "Password123!",
	})
	var bidderAuth services.AuthResponse
	_ = json.Unmarshal(bidderLogin.Body.Bytes(), &bidderAuth)

	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/auth/me", bidderAuth.AccessToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("/me failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{"refresh_token": bidderAuth.RefreshToken}); rec.Code != http.StatusOK {
		t.Fatalf("refresh failed: %d", rec.Code)
	}

	// RBAC: POSTOR no puede crear subasta (403)
	createPayload := map[string]any{
		"titulo": "Lote HTTP", "descripcion": "Desc", "precio_base": 2000,
		"inicio": time.Now(), "cierre": time.Now().Add(time.Hour),
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas", bidderAuth.AccessToken, createPayload); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for bidder creating auction, got %d", rec.Code)
	}

	// 2. Admin crea subasta, actualiza encargado/lote, activa, lista y consulta
	createRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas", adminAuth.AccessToken, createPayload)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create auction failed: %d %s", createRes.Code, createRes.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(createRes.Body.Bytes(), &created)
	auctionID := created["id"].(string)

	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/"+auctionID+"/encargado", adminAuth.AccessToken, map[string]string{"nombre": "Martillero Test", "licencia": "LIC-1"}); rec.Code != http.StatusOK {
		t.Fatalf("update encargado failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPut, "/api/v1/subastas/"+auctionID+"/detalles-lote", adminAuth.AccessToken, map[string]any{"categoria": "VEHICULOS", "precio_reserva": 2500}); rec.Code != http.StatusOK {
		t.Fatalf("update lot details failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/"+auctionID+"/activate", adminAuth.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("activate auction failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("list auctions failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("get auction failed: %d", rec.Code)
	}

	// 3. Garantía Stripe Intent + Create + GetMe + Webhook firmado
	intentRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/garantias/stripe-intent", bidderAuth.AccessToken, map[string]float64{"monto": 250})
	if intentRes.Code != http.StatusOK {
		t.Fatalf("prepare stripe intent failed: %d %s", intentRes.Code, intentRes.Body.String())
	}
	var offer services.StripeIntentOfferView
	_ = json.Unmarshal(intentRes.Body.Bytes(), &offer)

	garRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/garantias", bidderAuth.AccessToken, map[string]any{
		"monto": offer.HoldAmount, "payment_intent_id": offer.PaymentIntentID,
	})
	if garRes.Code != http.StatusCreated {
		t.Fatalf("create guarantee failed: %d %s", garRes.Code, garRes.Body.String())
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID+"/garantias/me", bidderAuth.AccessToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("get my guarantee failed: %d", rec.Code)
	}

	// 4. Puja + Delegación IA + Auto-Puja + Historial + Reporte IA + Cambio de Modelo
	bidRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/pujas", bidderAuth.AccessToken, map[string]any{"monto": 2100})
	if bidRes.Code != http.StatusCreated {
		t.Fatalf("place bid failed: %d %s", bidRes.Code, bidRes.Body.String())
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID+"/pujas?page=1&per_page=10", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("list bids failed: %d", rec.Code)
	}

	// Admin también obtiene garantía para delegar agente IA y competir contra el postor
	admIntent := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/garantias/stripe-intent", adminAuth.AccessToken, nil)
	var admOffer services.StripeIntentOfferView
	_ = json.Unmarshal(admIntent.Body.Bytes(), &admOffer)
	_ = doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/garantias", adminAuth.AccessToken, map[string]any{
		"monto": admOffer.HoldAmount, "payment_intent_id": admOffer.PaymentIntentID,
	})

	delRes := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/agentes-ia/delegar", adminAuth.AccessToken, map[string]any{
		"agente_id": "agent-1", "monto_maximo_autorizado": 5000, "estrategia": domain.StrategyMinimumIncrement,
	})
	if delRes.Code != http.StatusCreated {
		t.Fatalf("delegate agent failed: %d %s", delRes.Code, delRes.Body.String())
	}
	var mandate domain.AgentMandate
	_ = json.Unmarshal(delRes.Body.Bytes(), &mandate)
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/subastas/"+auctionID+"/agentes-ia/auto-puja", adminAuth.AccessToken, map[string]string{"mandato_id": mandate.ID}); rec.Code != http.StatusCreated {
		t.Fatalf("auto-bid failed: %d %s", rec.Code, rec.Body.String())
	}

	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/admin/ia/modelo", adminAuth.AccessToken, map[string]string{"modelo": "gemini-2.5-pro-forensic"}); rec.Code != http.StatusOK {
		t.Fatalf("switch ai model failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID+"/inteligencia-ia", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("intelligence report failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/ia/asistente", "", map[string]any{
		"mensaje":        "¿Cuanto necesito para ofertar en este lote?",
		"subasta_id":     auctionID,
		"rol_usuario":    "POSTOR",
		"nombre_usuario": "Postor HTTP",
	}); rec.Code != http.StatusOK {
		t.Fatalf("assistant chat failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSONReq(t, mux, http.MethodPost, "/api/v1/ia/asistente", "", map[string]any{
		"mensaje": "   ",
	}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty assistant message, got %d", rec.Code)
	}

	// 5. Conteo, Cierre y Auditoría Web3 + Webhook Stripe
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/"+auctionID+"/countdown", adminAuth.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("countdown failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodPatch, "/api/v1/subastas/"+auctionID+"/close", adminAuth.AccessToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("close auction failed: %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID+"/auditoria-web3?indice=0", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("audit web3 failed: %d %s", rec.Code, rec.Body.String())
	}

	whPayload := []byte(fmt.Sprintf(`{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"%s"}}}`, admOffer.PaymentIntentID))
	mac := hmac.New(sha256.New, []byte("whsec_handler_test"))
	mac.Write([]byte("1731888000."))
	mac.Write(whPayload)
	whReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(whPayload))
	whReq.Header.Set("Stripe-Signature", "t=1731888000,v1="+hex.EncodeToString(mac.Sum(nil)))
	whRec := httptest.NewRecorder()
	mux.ServeHTTP(whRec, whReq)
	if whRec.Code != http.StatusOK {
		t.Fatalf("stripe webhook failed: %d %s", whRec.Code, whRec.Body.String())
	}

	// 6. Errores esperados (JSON inválido, not found, query inválido)
	badJSON := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{bad"))
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badJSON)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad json, got %d", badRec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/inexistente", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on unknown auction, got %d", rec.Code)
	}
	if rec := doJSONReq(t, mux, http.MethodGet, "/api/v1/subastas/"+auctionID+"/pujas?page=abc", "", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid query, got %d", rec.Code)
	}
}
