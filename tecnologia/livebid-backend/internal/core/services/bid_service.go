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
	lock     domain.BidLockPort
	notifier domain.BidNotifierPort
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

func NewAuctionBidService(auctions domain.SubastaRepository, bids domain.BidRepository, locks ...domain.BidLockPort) *AuctionBidService {
	var lock domain.BidLockPort
	if len(locks) > 0 {
		lock = locks[0]
	}
	return &AuctionBidService{auctions: auctions, bids: bids, lock: lock, now: time.Now}
}

func NewAuctionBidServiceWithNotifier(auctions domain.SubastaRepository, bids domain.BidRepository, lock domain.BidLockPort, notifier domain.BidNotifierPort) *AuctionBidService {
	return &AuctionBidService{auctions: auctions, bids: bids, lock: lock, notifier: notifier, now: time.Now}
}

func (s *AuctionBidService) PlaceBid(ctx context.Context, cmd PlaceBidCmd) (BidResponse, error) {
	var unlock domain.UnlockFunc
	var err error
	if s.lock != nil {
		unlock, err = s.lock.AdquirirLock(ctx, cmd.SubastaID, 1500*time.Millisecond)
		if err != nil {
			return BidResponse{}, domain.ErrLockTimeout
		}
		defer func() { _ = unlock() }()
	}
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
	if s.notifier != nil {
		if err := s.notifier.BroadcastNuevaPuja(ctx, subasta.ID, *puja, extended, subasta.FechaCierre); err != nil {
			return BidResponse{}, err
		}
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
