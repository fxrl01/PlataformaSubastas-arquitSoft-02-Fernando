package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

func TestAuctionCatalogCreatesPendingAuction(t *testing.T) {
	repo := &MockRepo{subastas: map[string]*domain.Subasta{}}
	service := services.NuevoAuctionCatalogService(repo)
	start := time.Now().Add(time.Hour)

	auction, err := service.Crear(context.Background(), services.CrearSubastaCmd{
		ID: "sub-1", VendedorID: "usr-seller", Titulo: "Obra",
		PrecioBase: 100, Inicio: start, Cierre: start.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if auction.Estado != domain.SubastaPendiente || auction.VendedorID != "usr-seller" {
		t.Fatalf("unexpected auction: %+v", auction)
	}
}

func TestAuctionCatalogRejectsInvalidDates(t *testing.T) {
	repo := &MockRepo{subastas: map[string]*domain.Subasta{}}
	service := services.NuevoAuctionCatalogService(repo)
	start := time.Now()

	_, err := service.Crear(context.Background(), services.CrearSubastaCmd{
		ID: "sub-1", VendedorID: "usr-seller", Titulo: "Obra",
		PrecioBase: 100, Inicio: start, Cierre: start,
	})
	if err != domain.ErrSubastaInvalida {
		t.Fatalf("expected invalid auction error, got %v", err)
	}
}

func TestAuctionCatalogActivatesAndClosesAuction(t *testing.T) {
	start := time.Now().Add(time.Hour)
	auction := domain.NuevaSubasta("sub-1", "Obra", "", 100, start, start.Add(time.Hour))
	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}
	service := services.NuevoAuctionCatalogService(repo)

	if err := service.Activar(context.Background(), "sub-1"); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if err := service.Cerrar(context.Background(), "sub-1"); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if auction.Estado != domain.SubastaCerrada {
		t.Fatalf("expected closed auction, got %s", auction.Estado)
	}
}

func TestAuctionCatalogSettlesGuaranteesWhenClosing(t *testing.T) {
	start := time.Now().Add(time.Hour)
	auction := domain.NuevaSubasta("sub-1", "Obra", "", 100, start, start.Add(time.Hour))
	auction.Estado = domain.SubastaActiva
	auction.PostorLiderID = "usr-winner"
	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}
	guarantees := &settlementGuaranteeRepository{}
	service := services.NuevoAuctionCatalogServiceConGarantias(repo, guarantees)

	if err := service.Cerrar(context.Background(), "sub-1"); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if guarantees.auctionID != "sub-1" || guarantees.winnerID != "usr-winner" {
		t.Fatalf("unexpected settlement: auction=%q winner=%q", guarantees.auctionID, guarantees.winnerID)
	}
}

type settlementGuaranteeRepository struct {
	auctionID string
	winnerID  string
}

func (r *settlementGuaranteeRepository) FindActive(context.Context, string, string) (*domain.Guarantee, error) {
	return nil, domain.ErrUserNotFound
}

func (r *settlementGuaranteeRepository) Save(context.Context, *domain.Guarantee) error {
	return nil
}

func (r *settlementGuaranteeRepository) SettleAuction(_ context.Context, auctionID, winnerID string) error {
	r.auctionID = auctionID
	r.winnerID = winnerID
	return nil
}

func TestAuctionCatalogClosesExpiredAuctions(t *testing.T) {
	now := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	expired := domain.NuevaSubasta("expired", "Obra", "", 100, now.Add(-2*time.Hour), now.Add(-time.Minute))
	expired.Estado = domain.SubastaActiva
	current := domain.NuevaSubasta("current", "Otra obra", "", 100, now, now.Add(time.Hour))
	current.Estado = domain.SubastaActiva
	repo := &MockRepo{subastas: map[string]*domain.Subasta{
		expired.ID: expired, current.ID: current,
	}}
	service := services.NuevoAuctionCatalogService(repo)

	closed, err := service.CerrarVencidas(context.Background(), now)
	if err != nil {
		t.Fatalf("close expired auctions failed: %v", err)
	}
	if closed != 1 || expired.Estado != domain.SubastaCerrada || current.Estado != domain.SubastaActiva {
		t.Fatalf("unexpected expired auction result: closed=%d expired=%s current=%s",
			closed, expired.Estado, current.Estado)
	}
}

type indexedExpiredRepo struct {
	*MockRepo
	indexedCalls int
}

func (r *indexedExpiredRepo) ListarVencidas(_ context.Context, now time.Time) ([]*domain.Subasta, error) {
	r.indexedCalls++
	var result []*domain.Subasta
	for _, s := range r.subastas {
		if s.Estado.EsAceptandoPujas() && !s.FechaCierre.After(now) {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestAuctionCatalogUsesIndexedExpiredQueryWhenSupported(t *testing.T) {
	now := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	expired := domain.NuevaSubasta("expired", "Obra", "", 100, now.Add(-2*time.Hour), now.Add(-time.Minute))
	expired.Estado = domain.SubastaActiva
	repo := &indexedExpiredRepo{
		MockRepo: &MockRepo{subastas: map[string]*domain.Subasta{expired.ID: expired}},
	}
	service := services.NuevoAuctionCatalogService(repo)

	closed, err := service.CerrarVencidas(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if closed != 1 || repo.indexedCalls != 1 {
		t.Fatalf("expected ListarVencidas to be called once and close 1 auction, got closed=%d calls=%d", closed, repo.indexedCalls)
	}
}

type mockAuditTrailRepo struct {
	hashes  []string
	chain   []domain.AuditLogEntry
	receipt *domain.OnChainSettlementReceipt
}

func (m *mockAuditTrailRepo) ObtenerHashesAuditoria(context.Context, string) ([]string, error) {
	return m.hashes, nil
}

func (m *mockAuditTrailRepo) ObtenerCadenaAuditoria(context.Context, string) ([]domain.AuditLogEntry, error) {
	return m.chain, nil
}

func (m *mockAuditTrailRepo) GuardarReciboOnChain(_ context.Context, r domain.OnChainSettlementReceipt) error {
	m.receipt = &r
	return nil
}

func (m *mockAuditTrailRepo) ObtenerReciboOnChain(context.Context, string) (*domain.OnChainSettlementReceipt, error) {
	return m.receipt, nil
}

func (m *mockAuditTrailRepo) AppendAuditEvent(_ context.Context, _ string, event string, _ map[string]any) error {
	prev := "GENESIS"
	if len(m.chain) > 0 {
		prev = m.chain[len(m.chain)-1].CurrentHash
	}
	m.chain = append(m.chain, domain.AuditLogEntry{
		Event:        event,
		PreviousHash: prev,
		CurrentHash:  "hash_" + event,
	})
	return nil
}

func TestAuctionCloseAnchorsMerkleRootOnChainAndWeb3AuditInspectsProof(t *testing.T) {
	start := time.Now()
	auction := domain.NuevaSubasta("sub-web3", "Arte NFT", "", 1000, start, start.Add(time.Hour))
	auction.Estado = domain.SubastaActiva
	auction.PostorLiderID = "usr-winner"
	auction.PrecioActual = 1500

	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-web3": auction}}
	auditRepo := &mockAuditTrailRepo{
		hashes: []string{
			"1111111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222",
		},
		chain: []domain.AuditLogEntry{
			{Event: "PUJA_REGISTRADA", PreviousHash: "GENESIS", CurrentHash: "h1"},
			{Event: "PUJA_REGISTRADA", PreviousHash: "h1", CurrentHash: "h2"},
		},
	}
	escrow := web3.NewEVMEscrowAdapter(8453, "0x1111111111111111111111111111111111111111", "")
	catalog := services.NuevoAuctionCatalogService(repo).WithWeb3Settlement(auditRepo, escrow)

	if err := catalog.Cerrar(context.Background(), "sub-web3"); err != nil {
		t.Fatalf("close auction with web3 settlement failed: %v", err)
	}
	if auditRepo.receipt == nil || auditRepo.receipt.TxHash == "" {
		t.Fatalf("expected on-chain receipt to be stored on auction close, got %+v", auditRepo.receipt)
	}

	auditService := services.NewWeb3AuditService(repo, auditRepo)
	leafIdx := 1
	report, err := auditService.InspectAuction(context.Background(), "sub-web3", &leafIdx)
	if err != nil {
		t.Fatalf("inspect auction failed: %v", err)
	}
	if !report.ChainIntact || report.MerkleProof == nil || !report.MerkleProof.Verified {
		t.Fatalf("expected intact hash chain and verified Merkle proof, got %+v", report)
	}
	if report.OnChainReceipt == nil || report.OnChainReceipt.MerkleRoot != report.MerkleRoot {
		t.Fatalf("expected onchain receipt Merkle root to match report root, got %+v", report)
	}
	if len(report.AuditChain) != 2 {
		t.Fatalf("expected 2 entries in AuditChain, got %d", len(report.AuditChain))
	}
}

func TestAuctionCatalogUpdatesManagerLotDetailsAndStartsHammerCountdown(t *testing.T) {
	start := time.Now().Add(time.Hour)
	auction := domain.NuevaSubasta("sub-mgr", "Excavadora CAT 320", "Lote industrial", 25000, start, start.Add(2*time.Hour))
	auction.Estado = domain.SubastaActiva
	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-mgr": auction}}
	catalog := services.NuevoAuctionCatalogService(repo)

	updatedMgr, err := catalog.ActualizarEncargado(context.Background(), "sub-mgr", domain.EncargadoSubasta{
		MartilleroID: "MART-01",
		Nombre:       "Carlos Mendoza",
		Cargo:        "Martillero Público Oficial",
		Licencia:     "MART-REG-2026-099",
		Contacto:     "cmendoza@livebid.pe",
		Experiencia:  "Martillero Público con 15 años en liquidación de activos pesados.",
	})
	if err != nil {
		t.Fatalf("update encargado failed: %v", err)
	}
	if updatedMgr.MartilleroID != "MART-01" || updatedMgr.Encargado.Licencia != "MART-REG-2026-099" {
		t.Fatalf("unexpected manager details: %+v", updatedMgr)
	}

	updatedLot, err := catalog.ActualizarDetallesLote(context.Background(), "sub-mgr", domain.DetallesLote{
		Categoria:              "MAQUINARIA_PESADA",
		Ubicacion:              "Lima - Callao",
		PrecioReserva:          32000,
		InspeccionadoPor:       "Ing. Roberto Salazar (CIP 88412)",
		CertificadoPeritajeURL: "https://docs.livebid.pe/tasaciones/cat320.pdf",
		Especificaciones:       map[string]string{"motor": "C7.1 ACERT", "horas": "4100"},
	})
	if err != nil {
		t.Fatalf("update lot details failed: %v", err)
	}
	if updatedLot.DetallesLote.PrecioReserva != 32000 {
		t.Fatalf("unexpected lot details: %+v", updatedLot)
	}

	if err := catalog.IniciarConteo(context.Background(), "sub-mgr"); err != nil {
		t.Fatalf("start countdown failed: %v", err)
	}
	if auction.Estado != domain.SubastaEnConteo {
		t.Fatalf("expected EN_CONTEO state, got %s", auction.Estado)
	}
}
