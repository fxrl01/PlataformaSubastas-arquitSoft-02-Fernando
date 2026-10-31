package services

import (
	"context"
	"errors"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

var ErrInvalidBidHistoryQuery = errors.New("service: invalid bid history query")

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

type BidHistoryQuery struct {
	Page          int
	PerPage       int
	BidderID      string
	MinimumAmount float64
}

type BidHistoryResult struct {
	Items      []BidHistoryView `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"pagina"`
	PerPage    int              `json:"por_pagina"`
	TotalPages int              `json:"paginas"`
}

func NewBidHistoryService(auctions domain.SubastaRepository, bids domain.BidRepository) *BidHistoryService {
	return &BidHistoryService{auctions: auctions, bids: bids}
}

func (s *BidHistoryService) List(ctx context.Context, auctionID string) ([]BidHistoryView, error) {
	result, err := s.ListPage(ctx, auctionID, BidHistoryQuery{Page: 1, PerPage: 50})
	return result.Items, err
}

func (s *BidHistoryService) ListPage(ctx context.Context, auctionID string, query BidHistoryQuery) (BidHistoryResult, error) {
	if query.Page < 1 || query.PerPage < 1 || query.PerPage > 100 ||
		query.MinimumAmount < 0 {
		return BidHistoryResult{}, ErrInvalidBidHistoryQuery
	}
	if _, err := s.auctions.ObtenerPorID(ctx, auctionID); err != nil {
		return BidHistoryResult{}, err
	}
	offset := (query.Page - 1) * query.PerPage
	bids, total, err := s.bids.ListarPorSubasta(
		ctx, auctionID, query.BidderID, query.MinimumAmount, query.PerPage, offset,
	)
	if err != nil {
		return BidHistoryResult{}, err
	}
	views := make([]BidHistoryView, 0, len(bids))
	for index, bid := range bids {
		views = append(views, BidHistoryView{
			ID: bid.ID, AuctionID: bid.SubastaID, BidderID: bid.PostorID,
			Amount: bid.Monto, Sequence: offset + index + 1,
			Timestamp: bid.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		})
	}
	pages := 0
	if total > 0 {
		pages = (total + query.PerPage - 1) / query.PerPage
	}
	return BidHistoryResult{
		Items: views, Total: total, Page: query.Page,
		PerPage: query.PerPage, TotalPages: pages,
	}, nil
}
