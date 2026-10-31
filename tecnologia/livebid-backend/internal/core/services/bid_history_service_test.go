package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

func TestBidHistoryListsBidsInRepositoryOrder(t *testing.T) {
	now := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	auction := domain.NuevaSubasta("sub-1", "Obra", "", 100, now, now.Add(time.Hour))
	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}
	bids := &mockBidRepository{registered: []domain.Puja{
		domain.NuevaPuja("bid-1", "sub-1", "usr-1", 110, now),
		domain.NuevaPuja("bid-2", "sub-1", "usr-2", 125, now.Add(time.Minute)),
	}}
	service := services.NewBidHistoryService(repo, bids)

	result, err := service.List(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if len(result) != 2 || result[0].ID != "bid-1" || result[1].Sequence != 2 {
		t.Fatalf("unexpected bid history: %+v", result)
	}
}
