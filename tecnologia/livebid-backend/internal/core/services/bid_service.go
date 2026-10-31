package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type AuctionBidService struct {
	auctions domain.SubastaRepository
	bids     domain.BidRepository
	now      func() time.Time
}

type PlaceBidCmd struct {
	SubastaID string
	PostorID  string
	Monto     float64
}

type BidResponse struct {
	PujaID          string               `json:"puja_id"`
	SubastaID       string               `json:"subasta_id"`
	PostorLiderID   string               `json:"postor_lider_id"`
	PrecioActual    float64              `json:"precio_actual"`
	TiempoExtendido bool                 `json:"tiempo_extendido"`
	FechaCierre     time.Time            `json:"fecha_cierre"`
	Estado          domain.EstadoSubasta `json:"estado"`
}

func NewAuctionBidService(auctions domain.SubastaRepository, bids domain.BidRepository) *AuctionBidService {
	return &AuctionBidService{auctions: auctions, bids: bids, now: time.Now}
}

func (s *AuctionBidService) PlaceBid(ctx context.Context, cmd PlaceBidCmd) (BidResponse, error) {
	subasta, err := s.auctions.ObtenerPorID(ctx, cmd.SubastaID)
	if err != nil {
		return BidResponse{}, err
	}
	now := s.now()
	precioAnterior := subasta.PrecioActual
	pujaID, err := newID("bid_")
	if err != nil {
		return BidResponse{}, err
	}
	puja, extended, err := subasta.ValidarYRegistrarPuja(pujaID, cmd.PostorID, cmd.Monto, now)
	if err != nil {
		return BidResponse{}, err
	}
	if err := s.bids.RegistrarPujaAtomica(ctx, subasta.ID, *puja, precioAnterior, subasta.FechaCierre, extended); err != nil {
		return BidResponse{}, err
	}
	return BidResponse{
		PujaID: puja.ID, SubastaID: subasta.ID, PostorLiderID: subasta.PostorLiderID,
		PrecioActual: subasta.PrecioActual, TiempoExtendido: extended,
		FechaCierre: subasta.FechaCierre, Estado: subasta.Estado,
	}, nil
}

func newID(prefix string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}
