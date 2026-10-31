package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type mockGuaranteeRepository struct {
	active           *domain.Guarantee
	saved            *domain.Guarantee
	settledAuctionID string
	settledWinnerID  string
}

func (r *mockGuaranteeRepository) FindActive(context.Context, string, string) (*domain.Guarantee, error) {
	if r.active == nil {
		return nil, domain.ErrUserNotFound
	}
	return r.active, nil
}

func (r *mockGuaranteeRepository) Save(_ context.Context, guarantee *domain.Guarantee) error {
	r.saved = guarantee
	r.active = guarantee
	return nil
}

func (r *mockGuaranteeRepository) SettleAuction(_ context.Context, auctionID, winnerID string) error {
	r.settledAuctionID = auctionID
	r.settledWinnerID = winnerID
	return nil
}

func TestGuaranteeRequiresTenPercentOfBasePrice(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{}
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, guarantees,
	)

	view, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "usr-1", Amount: 250,
	})
	if err != nil {
		t.Fatalf("create guarantee failed: %v", err)
	}
	if view.Amount != 250 || view.Status != string(domain.GuaranteeHeld) {
		t.Fatalf("unexpected guarantee: %+v", view)
	}
}

func TestGuaranteeRejectsDuplicateActiveGuarantee(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{active: &domain.Guarantee{ID: "gar-1"}}
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, guarantees,
	)

	_, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "usr-1", Amount: 250,
	})
	if err != domain.ErrGuaranteeAlreadyActive {
		t.Fatalf("expected duplicate guarantee error, got %v", err)
	}
}
