package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type e2eUserRepo struct {
	users map[string]*domain.User
}

func newE2EUserRepo() *e2eUserRepo {
	return &e2eUserRepo{users: make(map[string]*domain.User)}
}

func (r *e2eUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *e2eUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *e2eUserRepo) Save(_ context.Context, user *domain.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *e2eUserRepo) UpdateStatus(_ context.Context, id string, status domain.UserStatus) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = status
	return nil
}

func (r *e2eUserRepo) ListUsers(_ context.Context, status domain.UserStatus, role domain.UserRole) ([]*domain.User, error) {
	var list []*domain.User
	for _, u := range r.users {
		if (status == "" || u.Status == status) && (role == "" || u.Role == role) {
			list = append(list, u)
		}
	}
	return list, nil
}

func (r *e2eUserRepo) UpdateRole(_ context.Context, id string, role domain.UserRole) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Role = role
	return nil
}

func (r *e2eUserRepo) UpdateKYCAndStatus(_ context.Context, id string, status domain.UserStatus, kycApproved bool) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = status
	u.KYCApproved = kycApproved
	return nil
}

type e2eGuaranteeRepo struct {
	byBidder      map[string]*domain.Guarantee
	settledWinner string
}

func newE2EGuaranteeRepo() *e2eGuaranteeRepo {
	return &e2eGuaranteeRepo{byBidder: make(map[string]*domain.Guarantee)}
}

func (r *e2eGuaranteeRepo) FindActive(_ context.Context, auctionID, bidderID string) (*domain.Guarantee, error) {
	g, ok := r.byBidder[auctionID+":"+bidderID]
	if !ok || g.Status != domain.GuaranteeHeld {
		return nil, domain.ErrUserNotFound
	}
	return g, nil
}

func (r *e2eGuaranteeRepo) Save(_ context.Context, g *domain.Guarantee) error {
	r.byBidder[g.AuctionID+":"+g.BidderID] = g
	return nil
}

func (r *e2eGuaranteeRepo) SettleAuction(_ context.Context, auctionID, winnerID string) error {
	r.settledWinner = winnerID
	for key, g := range r.byBidder {
		if g.AuctionID != auctionID || g.Status != domain.GuaranteeHeld {
			continue
		}
		if winnerID != "" && g.BidderID == winnerID {
			r.byBidder[key].Status = domain.GuaranteeCaptured
		} else {
			r.byBidder[key].Status = domain.GuaranteeReleased
		}
	}
	return nil
}

type e2eBidAuditRepo struct {
	auctionRepo *MockRepo
	bids        []domain.Puja
	hashes      []string
	chain       []domain.AuditLogEntry
	receipt     *domain.OnChainSettlementReceipt
}

func (r *e2eBidAuditRepo) BuscarPorIdempotencyKey(_ context.Context, _, _, _ string) (*domain.Puja, error) {
	return nil, nil
}

func (r *e2eBidAuditRepo) RegistrarPujaAtomica(_ context.Context, subastaID string, puja domain.Puja, _ float64, fechaCierre time.Time, _ bool, _ string) error {
	if s, ok := r.auctionRepo.subastas[subastaID]; ok {
		s.PrecioActual = puja.Monto
		s.PostorLiderID = puja.PostorID
		s.FechaCierre = fechaCierre
	}
	r.bids = append(r.bids, puja)

	prev := "GENESIS"
	if len(r.chain) > 0 {
		prev = r.chain[len(r.chain)-1].CurrentHash
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%.2f", prev, subastaID, puja.ID, puja.Monto)))
	hash := hex.EncodeToString(sum[:])
	r.hashes = append(r.hashes, hash)
	r.chain = append(r.chain, domain.AuditLogEntry{
		Event:        "PUJA_REGISTRADA",
		PreviousHash: prev,
		CurrentHash:  hash,
	})
	return nil
}

func (r *e2eBidAuditRepo) ListarPorSubasta(_ context.Context, _ string, _ string, _ float64, _, _ int) ([]domain.Puja, int, error) {
	return r.bids, len(r.bids), nil
}

func (r *e2eBidAuditRepo) ObtenerHashesAuditoria(context.Context, string) ([]string, error) {
	return r.hashes, nil
}

func (r *e2eBidAuditRepo) ObtenerCadenaAuditoria(context.Context, string) ([]domain.AuditLogEntry, error) {
	return r.chain, nil
}

func (r *e2eBidAuditRepo) GuardarReciboOnChain(_ context.Context, receipt domain.OnChainSettlementReceipt) error {
	r.receipt = &receipt
	prev := "GENESIS"
	if len(r.chain) > 0 {
		prev = r.chain[len(r.chain)-1].CurrentHash
	}
	r.chain = append(r.chain, domain.AuditLogEntry{
		Event:        "CIERRE_ONCHAIN_MERKLE",
		PreviousHash: prev,
		CurrentHash:  receipt.TxHash,
	})
	return nil
}

func (r *e2eBidAuditRepo) ObtenerReciboOnChain(context.Context, string) (*domain.OnChainSettlementReceipt, error) {
	return r.receipt, nil
}

func (r *e2eBidAuditRepo) AppendAuditEvent(_ context.Context, subastaID, event string, _ map[string]any) error {
	prev := "GENESIS"
	if len(r.chain) > 0 {
		prev = r.chain[len(r.chain)-1].CurrentHash
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", prev, subastaID, event)))
	r.chain = append(r.chain, domain.AuditLogEntry{
		Event:        event,
		PreviousHash: prev,
		CurrentHash:  hex.EncodeToString(sum[:]),
	})
	return nil
}

func TestEndToEndLiveBidSixModulesLifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. M1 — Bootstrap Admin, Registro de Vendedor y Postores, y Aprobación KYC/Rol
	userRepo := newE2EUserRepo()
	authService := services.NewAuthService(userRepo, authHasher{}, authTokenIssuer{}, &refreshStore{tokens: map[string]string{}})

	admin, err := authService.BootstrapAdmin(ctx, "Admin LiveBid", "admin@livebid.pe", "AdminLiveBid2026!")
	if err != nil || admin.Role != domain.RoleAdmin {
		t.Fatalf("BootstrapAdmin failed: %v", err)
	}

	seller, err := authService.Register(ctx, services.RegisterUserCmd{
		Name: "Consignaciones Perú", Email: "vendedor@livebid.pe", Password: "Password123!", RequestedRole: domain.RoleSeller,
	})
	if err != nil {
		t.Fatalf("Register seller failed: %v", err)
	}
	if _, err := authService.UpdateUserRole(ctx, seller.ID, domain.RoleSeller); err != nil {
		t.Fatalf("UpdateUserRole seller failed: %v", err)
	}
	kycTrue := true
	if _, err := authService.UpdateUserStatus(ctx, seller.ID, domain.UserActive, &kycTrue); err != nil {
		t.Fatalf("UpdateUserStatus seller failed: %v", err)
	}

	bidder1, _ := authService.Register(ctx, services.RegisterUserCmd{
		Name: "Postor 1", Email: "postor1@livebid.pe", Password: "Password123!",
	})
	_ = authService.ActivateUser(ctx, bidder1.ID)

	bidder2, _ := authService.Register(ctx, services.RegisterUserCmd{
		Name: "Postor 2", Email: "postor2@livebid.pe", Password: "Password123!",
	})
	_ = authService.ActivateUser(ctx, bidder2.ID)

	// 2. M2 — Creación de Subasta, Encargado Oficial, Ficha del Lote con Imágenes y Activación
	auctionRepo := &MockRepo{subastas: map[string]*domain.Subasta{}}
	guaranteeRepo := newE2EGuaranteeRepo()
	bidAuditRepo := &e2eBidAuditRepo{auctionRepo: auctionRepo}
	escrowOnChain := web3.NewEVMEscrowAdapter(8453, "0x1111111111111111111111111111111111111111", "")

	catalogService := services.NuevoAuctionCatalogServiceConGarantias(auctionRepo, guaranteeRepo).
		WithWeb3Settlement(bidAuditRepo, escrowOnChain)

	now := time.Now()
	subasta, err := catalogService.Crear(ctx, services.CrearSubastaCmd{
		ID:         "SUB-E2E-2026",
		VendedorID: seller.ID,
		Titulo:     "Porsche 911 Carrera S",
		PrecioBase: 50000,
		Inicio:     now,
		Cierre:     now.Add(time.Hour),
		Encargado: &domain.EncargadoSubasta{
			MartilleroID: "MART-01",
			Nombre:       "Dr. Javier Prado",
			Cargo:        "Martillero Público",
			Licencia:     "REG-SUNARP-402",
		},
		DetallesLote: &domain.DetallesLote{
			Categoria:     "VEHICULOS_PREMIUM",
			Ubicacion:     "San Isidro, Lima",
			PrecioReserva: 55000,
			ImagenesURLs:  []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="},
		},
	})
	if err != nil {
		t.Fatalf("Crear subasta E2E failed: %v", err)
	}
	if err := catalogService.Activar(ctx, subasta.ID); err != nil {
		t.Fatalf("Activar subasta E2E failed: %v", err)
	}

	// 3. M3 — Constitución de Garantías con Stripe PaymentIntents (Manual Hold) para ambos postores
	stripeGateway := payments.NewStripeAdapter("", "pk_test_e2e", "whsec_e2e")
	guaranteeService := services.NewGuaranteeService(auctionRepo, guaranteeRepo).WithPaymentGateway(stripeGateway)

	for _, bID := range []string{bidder1.ID, bidder2.ID} {
		offer, err := guaranteeService.PrepareStripeIntent(ctx, subasta.ID, bID, 0)
		if err != nil {
			t.Fatalf("PrepareStripeIntent for %s failed: %v", bID, err)
		}
		if _, err := guaranteeService.Create(ctx, services.CreateGuaranteeCmd{
			AuctionID:       subasta.ID,
			BidderID:        bID,
			Amount:          offer.HoldAmount,
			PaymentIntentID: offer.PaymentIntentID,
		}); err != nil {
			t.Fatalf("Create guarantee for %s failed: %v", bID, err)
		}
	}

	// 4. M4 & M5 — Pujas en Vivo + Auto-Puja Delegada de IA
	bidService := services.NewAuctionBidServiceWithNotifierAndGuarantee(
		auctionRepo, bidAuditRepo, &mockBidLock{}, nil, guaranteeRepo,
	)
	agentService := services.NewAgentDelegationService(auctionRepo, bidService)

	// Puja manual del Postor 1 ($51,000)
	resp1, err := bidService.PlaceBid(ctx, services.PlaceBidCmd{
		SubastaID: subasta.ID, PostorID: bidder1.ID, Monto: 51000, IdempotencyKey: "idem-b1-1",
	})
	if err != nil || resp1.PrecioActual != 51000 {
		t.Fatalf("PlaceBid bidder1 failed: resp=%+v err=%v", resp1, err)
	}

	// Postor 2 delega un Agente Autónomo con techo $55,000 y ejecuta Auto-Puja ($52,000)
	mandate, err := agentService.CreateMandate(ctx, services.CreateMandateCmd{
		AuctionID:           subasta.ID,
		PrincipalBidderID:   bidder2.ID,
		AgentID:             "agent-gemini-sniper",
		MaxAuthorizedAmount: 55000,
		Strategy:            domain.StrategyMinimumIncrement,
	})
	if err != nil {
		t.Fatalf("CreateMandate failed: %v", err)
	}
	resp2, err := agentService.ExecuteAutonomousBid(ctx, services.ExecuteAutoBidCmd{
		AuctionID: subasta.ID, BidderID: bidder2.ID, MandateID: mandate.ID,
	})
	if err != nil || resp2.PostorLiderID != bidder2.ID || resp2.PrecioActual != 52000 {
		t.Fatalf("ExecuteAutonomousBid bidder2 failed: resp=%+v err=%v", resp2, err)
	}

	// 5. M6 — Conteo de Martillo, Cierre, Liquidación de Garantías y Auditoría Criptográfica
	if err := catalogService.IniciarConteo(ctx, subasta.ID); err != nil {
		t.Fatalf("IniciarConteo failed: %v", err)
	}
	if err := catalogService.Cerrar(ctx, subasta.ID); err != nil {
		t.Fatalf("Cerrar subasta failed: %v", err)
	}

	// Verificar liquidación: Ganador (Postor 2) CAPTURADA_COBRO, Perdedor (Postor 1) LIBERADA
	if guaranteeRepo.byBidder[subasta.ID+":"+bidder2.ID].Status != domain.GuaranteeCaptured {
		t.Fatalf("expected winning bidder2 guarantee CAPTURADA_COBRO, got %s", guaranteeRepo.byBidder[subasta.ID+":"+bidder2.ID].Status)
	}
	if guaranteeRepo.byBidder[subasta.ID+":"+bidder1.ID].Status != domain.GuaranteeReleased {
		t.Fatalf("expected losing bidder1 guarantee LIBERADA, got %s", guaranteeRepo.byBidder[subasta.ID+":"+bidder1.ID].Status)
	}

	// Verificar Certificado de Auditoría Web3 + Cadena SHA-256 + Merkle Proof
	auditService := services.NewWeb3AuditService(auctionRepo, bidAuditRepo)
	proofIdx := 1
	report, err := auditService.InspectAuction(ctx, subasta.ID, &proofIdx)
	if err != nil {
		t.Fatalf("InspectAuction failed: %v", err)
	}
	if !report.ChainIntact || report.TotalAuditedBids != 2 || report.MerkleProof == nil || !report.MerkleProof.Verified {
		t.Fatalf("expected intact audit chain and verified Merkle proof, got %+v", report)
	}
	if len(report.AuditChain) < 5 {
		t.Fatalf("expected full lifecycle events in AuditChain (SUBASTA_CREADA, SALA_ACTIVADA, PUJA_REGISTRADA x2, CONTEO, CIERRE), got %d", len(report.AuditChain))
	}
}
