package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type mockBidRepository struct {
	registered []domain.Puja
}

type mockBidLock struct {
	locked bool
}

func (l *mockBidLock) AdquirirLock(_ context.Context, _ string, _ time.Duration) (domain.UnlockFunc, error) {
	if l.locked {
		return nil, domain.ErrLockTimeout
	}
	l.locked = true
	return func() error {
		l.locked = false
		return nil
	}, nil
}

func (r *mockBidRepository) RegistrarPujaAtomica(_ context.Context, _ string, puja domain.Puja, _ float64, _ time.Time, _ bool) error {
	r.registered = append(r.registered, puja)
	return nil
}

func TestPlaceBidUsesAuthenticatedBidderAndPersistsBid(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	if err := auction.Activar(); err != nil {
		t.Fatal(err)
	}
	auctionRepo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}
	bidRepo := &mockBidRepository{}
	service := services.NewAuctionBidService(auctionRepo, bidRepo)

	response, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 3000,
	})
	if err != nil {
		t.Fatalf("place bid failed: %v", err)
	}
	if response.PrecioActual != 3000 || response.PostorLiderID != "usr-bidder" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if len(bidRepo.registered) != 1 || bidRepo.registered[0].PostorID != "usr-bidder" {
		t.Fatalf("bid was not persisted with authenticated bidder: %+v", bidRepo.registered)
	}
}

func TestPlaceBidRejectsInsufficientAmount(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	_ = auction.Activar()
	bidRepo := &mockBidRepository{}
	service := services.NewAuctionBidService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, bidRepo,
	)

	_, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 2500,
	})
	if err != domain.ErrPujaInferior {
		t.Fatalf("expected insufficient bid error, got %v", err)
	}
	if len(bidRepo.registered) != 0 {
		t.Fatal("rejected bid was persisted")
	}
}

func TestPlaceBidRejectsWhenAuctionLockIsUnavailable(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	_ = auction.Activar()
	lock := &mockBidLock{locked: true}
	service := services.NewAuctionBidService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}},
		&mockBidRepository{}, lock,
	)

	_, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 3000,
	})
	if err != domain.ErrLockTimeout {
		t.Fatalf("expected lock timeout, got %v", err)
	}
}
