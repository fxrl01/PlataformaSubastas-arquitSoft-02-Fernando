package services_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type spyNotifier struct {
	bidEvents    atomic.Int64
	closedEvents atomic.Int64
}

func (n *spyNotifier) BroadcastNuevaPuja(context.Context, string, domain.Puja, bool, time.Time) error {
	n.bidEvents.Add(1)
	return nil
}

func (n *spyNotifier) BroadcastSubastaCerrada(context.Context, string, *domain.Puja) error {
	n.closedEvents.Add(1)
	return nil
}

func TestServicesCoverageMissingPaths(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	// 1. AuthService: GetProfile, ListUsers, UpdateUserRole, UpdateUserStatus
	userRepo := newE2EUserRepo()
	auth := services.NewAuthService(userRepo, authHasher{}, authTokenIssuer{}, &refreshStore{tokens: map[string]string{}})
	reg, err := auth.Register(ctx, services.RegisterUserCmd{Name: "Lucia", Email: "lucia@livebid.pe", Password: "Password123!"})
	if err != nil {
		t.Fatal(err)
	}
	if prof, err := auth.GetProfile(ctx, reg.ID); err != nil || prof.Email != "lucia@livebid.pe" {
		t.Fatalf("GetProfile failed: %+v err=%v", prof, err)
	}
	if _, err := auth.GetProfile(ctx, "unknown"); err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if list, err := auth.ListUsers(ctx, domain.UserPending, domain.RoleBidder); err != nil || len(list) != 1 {
		t.Fatalf("ListUsers failed: len=%d err=%v", len(list), err)
	}
	if _, err := auth.ListUsers(ctx, "BAD", ""); err != domain.ErrInvalidUserStatus {
		t.Fatalf("expected ErrInvalidUserStatus, got %v", err)
	}
	if _, err := auth.ListUsers(ctx, "", "BAD"); err != domain.ErrInvalidRole {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}

	// 2. BidHistoryService.List + AuctionCatalogService con notifier + scheduler tick
	auction := domain.NuevaSubasta("sub-cov-1", "Arte", "", 1000, now.Add(-2*time.Hour), now.Add(-time.Minute))
	_ = auction.Activar()
	auctionRepo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-cov-1": auction}}
	bidRepo := &mockBidRepository{
		registered: []domain.Puja{domain.NuevaPuja("p1", "sub-cov-1", "u1", 1100, now)},
	}
	history := services.NewBidHistoryService(auctionRepo, bidRepo)
	items, err := history.List(ctx, "sub-cov-1")
	if err != nil || len(items) != 1 {
		t.Fatalf("BidHistoryService.List failed: len=%d err=%v", len(items), err)
	}

	notifier := &spyNotifier{}
	catalog := services.NuevoAuctionCatalogServiceConGarantiasYNotificador(auctionRepo, nil, notifier)
	sCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		services.NewAuctionScheduler(catalog, 10*time.Millisecond).Run(sCtx)
	}()
	time.Sleep(45 * time.Millisecond)
	cancel()
	<-done
	if auction.Estado != domain.SubastaCerrada || notifier.closedEvents.Load() == 0 {
		t.Fatalf("expected scheduler tick to close expired auction and notify, state=%s events=%d", auction.Estado, notifier.closedEvents.Load())
	}

	// 3. AuctionBidService WithNotifier & WithFraudTelemetry + GuaranteeService GetActive & WithBlockchainEscrow
	sub2 := domain.NuevaSubasta("sub-cov-2", "Reloj", "", 1000, now, now.Add(time.Hour))
	_ = sub2.Activar()
	auctionRepo.subastas["sub-cov-2"] = sub2

	router := ai.NewMultiModelRouter("gemini", ai.NewCircuitBreakerIA(ai.NewGeminiRiskAdapter(""), 2, time.Second, time.Second), time.Minute)
	detector := ai.NewAsyncFraudDetector(router)

	bidSvc := services.NewAuctionBidServiceWithNotifier(auctionRepo, bidRepo, nil, notifier).WithFraudTelemetry(detector)
	if _, err := bidSvc.PlaceBid(ctx, services.PlaceBidCmd{SubastaID: "sub-cov-2", PostorID: "u-cov", Monto: 1100}); err != nil {
		t.Fatalf("PlaceBid with notifier & telemetry failed: %v", err)
	}
	if notifier.bidEvents.Load() == 0 {
		t.Fatal("expected notifier to emit bid event")
	}

	garRepo := newE2EGuaranteeRepo()
	escrow := web3.NewEVMEscrowAdapter(8453, "0x1111111111111111111111111111111111111111", "")
	garSvc := services.NewGuaranteeService(auctionRepo, garRepo, router).WithBlockchainEscrow(escrow)
	createdGar, err := garSvc.Create(ctx, services.CreateGuaranteeCmd{
		AuctionID: "sub-cov-2", BidderID: "u-cov", Amount: 200, WalletAddress: "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	})
	if err != nil || createdGar.OnChainTxHash == "" {
		t.Fatalf("GuaranteeService with blockchain escrow failed: %+v err=%v", createdGar, err)
	}
	if active, err := garSvc.GetActive(ctx, "sub-cov-2", "u-cov"); err != nil || active.ID != createdGar.ID {
		t.Fatalf("GetActive failed: %+v err=%v", active, err)
	}

	// 4. Web3AuditService verifyHashChain tamper detection
	tamperedAudit := &mockAuditTrailRepo{
		hashes: []string{"aaaa"},
		chain: []domain.AuditLogEntry{
			{Event: "E1", PreviousHash: "GENESIS", CurrentHash: "h1"},
			{Event: "E2", PreviousHash: "BROKEN_LINK", CurrentHash: "h2"},
		},
	}
	auditSvc := services.NewWeb3AuditService(auctionRepo, tamperedAudit)
	rep, err := auditSvc.InspectAuction(ctx, "sub-cov-2", nil)
	if err != nil || rep.ChainIntact {
		t.Fatalf("expected ChainIntact=false on broken hash chain, got %+v err=%v", rep, err)
	}

	// Empty chain & invalid first PreviousHash & missing CurrentHash
	if repEmpty, err := services.NewWeb3AuditService(auctionRepo, &mockAuditTrailRepo{}).InspectAuction(ctx, "sub-cov-2", nil); err != nil || !repEmpty.ChainIntact {
		t.Fatalf("expected empty chain to be intact, got %+v err=%v", repEmpty, err)
	}
	badGenesis := &mockAuditTrailRepo{chain: []domain.AuditLogEntry{{Event: "E1", PreviousHash: "WRONG", CurrentHash: "h1"}}}
	if r2, _ := services.NewWeb3AuditService(auctionRepo, badGenesis).InspectAuction(ctx, "sub-cov-2", nil); r2.ChainIntact {
		t.Fatal("expected ChainIntact=false when first entry PreviousHash != GENESIS")
	}
	emptyHash := &mockAuditTrailRepo{chain: []domain.AuditLogEntry{{Event: "E1", PreviousHash: "GENESIS", CurrentHash: ""}}}
	if r3, _ := services.NewWeb3AuditService(auctionRepo, emptyHash).InspectAuction(ctx, "sub-cov-2", nil); r3.ChainIntact {
		t.Fatal("expected ChainIntact=false when CurrentHash is empty")
	}

	// 5. AuctionCatalogService.Listar, error paths y AgentDelegationService.CountActiveMandates
	if listed, err := catalog.Listar(ctx); err != nil || len(listed) == 0 {
		t.Fatalf("catalog.Listar failed: len=%d err=%v", len(listed), err)
	}
	if _, err := catalog.Obtener(ctx, ""); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on empty id, got %v", err)
	}
	if err := catalog.Activar(ctx, "missing"); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on Activar missing, got %v", err)
	}
	if err := catalog.IniciarConteo(ctx, "missing"); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on IniciarConteo missing, got %v", err)
	}
	if err := catalog.Cerrar(ctx, "missing"); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on Cerrar missing, got %v", err)
	}
	if _, err := catalog.ActualizarEncargado(ctx, "missing", domain.EncargadoSubasta{Nombre: "X"}); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on ActualizarEncargado missing, got %v", err)
	}
	if _, err := catalog.ActualizarDetallesLote(ctx, "missing", domain.DetallesLote{Categoria: "X"}); err != domain.ErrSubastaNoEncontrada {
		t.Fatalf("expected ErrSubastaNoEncontrada on ActualizarDetallesLote missing, got %v", err)
	}

	agentSvc := services.NewAgentDelegationService(auctionRepo, bidSvc)
	if _, err := agentSvc.CreateMandate(ctx, services.CreateMandateCmd{
		AuctionID: "sub-cov-2", PrincipalBidderID: "u-agent", AgentID: "ag-1", MaxAuthorizedAmount: 5000,
	}); err != nil {
		t.Fatalf("CreateMandate failed: %v", err)
	}
	if count := agentSvc.CountActiveMandates("sub-cov-2", now); count != 1 {
		t.Fatalf("expected 1 active mandate, got %d", count)
	}

	// GuaranteeService PrepareStripeIntent & HandleStripeWebhook without gateway
	noGwGar := services.NewGuaranteeService(auctionRepo, garRepo)
	if _, err := noGwGar.PrepareStripeIntent(ctx, "sub-cov-2", "u-new", 0); err != domain.ErrStripeHoldFailed {
		t.Fatalf("expected ErrStripeHoldFailed without gateway, got %v", err)
	}
	if _, err := noGwGar.HandleStripeWebhook(ctx, []byte("{}"), "sig"); err != domain.ErrStripeWebhookInvalid {
		t.Fatalf("expected ErrStripeWebhookInvalid without gateway, got %v", err)
	}

	// AuthService: basic non-admin repository fallbacks + BootstrapAdmin promoting existing user
	basicRepo := &basicOnlyUserRepo{users: map[string]*domain.User{}}
	basicAuth := services.NewAuthService(basicRepo, authHasher{}, authTokenIssuer{}, &refreshStore{tokens: map[string]string{}})
	regBasic, err := basicAuth.Register(ctx, services.RegisterUserCmd{Name: "Basic", Email: "basic@livebid.pe", Password: "Password123!"})
	if err != nil {
		t.Fatal(err)
	}
	if list, err := basicAuth.ListUsers(ctx, "", ""); err != nil || len(list) != 0 {
		t.Fatalf("expected empty list for non-admin repo, got len=%d err=%v", len(list), err)
	}
	if u, err := basicAuth.UpdateUserRole(ctx, regBasic.ID, domain.RoleSeller); err != nil || u.Role != domain.RoleSeller {
		t.Fatalf("UpdateUserRole fallback failed: %+v err=%v", u, err)
	}
	if u, err := basicAuth.UpdateUserStatus(ctx, regBasic.ID, domain.UserActive, nil); err != nil || u.Status != domain.UserActive {
		t.Fatalf("UpdateUserStatus fallback failed: %+v err=%v", u, err)
	}
	if _, err := basicAuth.BootstrapAdmin(ctx, "Root", "basic@livebid.pe", "Password123!"); err != nil {
		t.Fatalf("BootstrapAdmin existing failed: %v", err)
	}
	if _, err := basicAuth.BootstrapAdmin(ctx, "", "", "short"); err != domain.ErrInvalidUser {
		t.Fatalf("expected ErrInvalidUser on invalid bootstrap args, got %v", err)
	}

	// AuctionService.ProcesarPuja lock timeout + escrow failure
	lockBusy := &mockBidLock{locked: true}
	busySvc := services.NuevoAuctionService(auctionRepo, lockBusy, nil, nil, nil)
	if _, err := busySvc.ProcesarPuja(ctx, services.ProcesarPujaCmd{SubastaID: "sub-cov-2", PostorID: "u1", Monto: 1500}); err != domain.ErrLockTimeout {
		t.Fatalf("expected ErrLockTimeout, got %v", err)
	}
	noEscrowSvc := services.NuevoAuctionService(auctionRepo, &mockBidLock{}, &MockEscrow{valido: false}, nil, nil)
	if _, err := noEscrowSvc.ProcesarPuja(ctx, services.ProcesarPujaCmd{SubastaID: "sub-cov-2", PostorID: "u1", Monto: 1500}); err != domain.ErrGarantiaInsuficiente {
		t.Fatalf("expected ErrGarantiaInsuficiente, got %v", err)
	}
}

type basicOnlyUserRepo struct {
	users map[string]*domain.User
}

func (r *basicOnlyUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (r *basicOnlyUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (r *basicOnlyUserRepo) Save(_ context.Context, u *domain.User) error {
	r.users[u.ID] = u
	return nil
}
func (r *basicOnlyUserRepo) UpdateStatus(_ context.Context, id string, s domain.UserStatus) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = s
	return nil
}
