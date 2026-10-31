package services

import (
	"context"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type BidHistoryService struct {
	auctions domain.SubastaRepository
	bids     domain.BidRepository
}

type BidHistoryView struct {
	ID        string  `json:"id"`
	AuctionID string  `json:"subasta_id"`
	BidderID  string  `json:"postor_id"`
	Amount    float64 `json:"monto"`
	Sequence  int     `json:"correlativo"`
	Timestamp string  `json:"timestamp"`
}

func NewBidHistoryService(auctions domain.SubastaRepository, bids domain.BidRepository) *BidHistoryService {
	return &BidHistoryService{auctions: auctions, bids: bids}
}

func (s *BidHistoryService) List(ctx context.Context, auctionID string) ([]BidHistoryView, error) {
	if _, err := s.auctions.ObtenerPorID(ctx, auctionID); err != nil {
		return nil, err
	}
	bids, err := s.bids.ListarPorSubasta(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	views := make([]BidHistoryView, 0, len(bids))
	for index, bid := range bids {
		views = append(views, BidHistoryView{
			ID: bid.ID, AuctionID: bid.SubastaID, BidderID: bid.PostorID,
			Amount: bid.Monto, Sequence: index + 1, Timestamp: bid.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		})
	}
	return views, nil
}
