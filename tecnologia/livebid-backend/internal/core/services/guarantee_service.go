package services

import (
	"context"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type GuaranteeService struct {
	auctions   domain.SubastaRepository
	guarantees domain.GuaranteeRepository
	now        func() time.Time
}

type CreateGuaranteeCmd struct {
	AuctionID string
	BidderID  string
	Amount    float64
}

type GuaranteeView struct {
	ID              string  `json:"id"`
	AuctionID       string  `json:"subasta_id"`
	BidderID        string  `json:"postor_id"`
	Amount          float64 `json:"monto"`
	AuthorizationID string  `json:"autorizacion_id"`
	Status          string  `json:"estado"`
	TrustScore      int     `json:"score_confianza"`
}

func NewGuaranteeService(auctions domain.SubastaRepository, guarantees domain.GuaranteeRepository) *GuaranteeService {
	return &GuaranteeService{auctions: auctions, guarantees: guarantees, now: time.Now}
}

func (s *GuaranteeService) Create(ctx context.Context, cmd CreateGuaranteeCmd) (GuaranteeView, error) {
	auction, err := s.auctions.ObtenerPorID(ctx, cmd.AuctionID)
	if err != nil {
		return GuaranteeView{}, err
	}
	required := domain.RequiredGuarantee(auction.PrecioBase)
	if cmd.Amount < required {
		return GuaranteeView{}, domain.ErrGuaranteeInsufficient
	}
	if active, err := s.guarantees.FindActive(ctx, cmd.AuctionID, cmd.BidderID); err == nil && active != nil {
		return GuaranteeView{}, domain.ErrGuaranteeAlreadyActive
	}
	id, err := newID("gar_")
	if err != nil {
		return GuaranteeView{}, err
	}
	authorization, err := newID("auth_")
	if err != nil {
		return GuaranteeView{}, err
	}
	guarantee := &domain.Guarantee{
		ID: id, AuctionID: cmd.AuctionID, BidderID: cmd.BidderID,
		Amount: cmd.Amount, AuthorizationID: authorization,
		Status: domain.GuaranteeHeld, TrustScore: 75, CreatedAt: s.now(),
	}
	if err := s.guarantees.Save(ctx, guarantee); err != nil {
		return GuaranteeView{}, err
	}
	return toGuaranteeView(guarantee), nil
}

func toGuaranteeView(g *domain.Guarantee) GuaranteeView {
	return GuaranteeView{ID: g.ID, AuctionID: g.AuctionID, BidderID: g.BidderID,
		Amount: g.Amount, AuthorizationID: g.AuthorizationID, Status: string(g.Status),
		TrustScore: g.TrustScore}
}
