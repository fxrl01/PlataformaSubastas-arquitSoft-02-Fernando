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

	result, err := service.ListPage(context.Background(), "sub-1", services.BidHistoryQuery{
		Page: 1, PerPage: 1, MinimumAmount: 100,
	})
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].ID != "bid-1" ||
		result.TotalPages != 2 {
		t.Fatalf("unexpected bid history: %+v", result)
	}
}

func TestBidHistoryRejectsInvalidPageSize(t *testing.T) {
	service := services.NewBidHistoryService(&MockRepo{}, &mockBidRepository{})
	_, err := service.ListPage(context.Background(), "sub-1", services.BidHistoryQuery{
		Page: 1, PerPage: 101,
	})
	if err != services.ErrInvalidBidHistoryQuery {
		t.Fatalf("expected invalid query error, got %v", err)
	}
}
