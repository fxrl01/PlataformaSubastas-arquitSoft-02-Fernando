package services_test

import (
	"context"
	"testing"
	"time"

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
