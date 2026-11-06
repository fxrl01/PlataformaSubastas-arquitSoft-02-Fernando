package services

import (
	"context"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type AgentDelegationService struct {
	mu       sync.RWMutex
	auctions domain.SubastaRepository
	bids     *AuctionBidService
	mandates map[string]*domain.AgentMandate
	now      func() time.Time
}

type CreateMandateCmd struct {
	AuctionID           string
	PrincipalBidderID   string
	AgentID             string
	MaxAuthorizedAmount float64
	Strategy            domain.AgentStrategy
	TTL                 time.Duration
}

type ExecuteAutoBidCmd struct {
	AuctionID string
	BidderID  string
	MandateID string
}

func NewAgentDelegationService(auctions domain.SubastaRepository, bids *AuctionBidService) *AgentDelegationService {
	s := &AgentDelegationService{
		auctions: auctions,
		bids:     bids,
		mandates: make(map[string]*domain.AgentMandate),
		now:      time.Now,
	}
	if bids != nil {
		bids.WithMandateValidator(s.ValidateMandateForBid)
	}
	return s
}

func (s *AgentDelegationService) CreateMandate(ctx context.Context, cmd CreateMandateCmd) (*domain.AgentMandate, error) {
	auction, err := s.auctions.ObtenerPorID(ctx, cmd.AuctionID)
	if err != nil {
		return nil, err
	}
	if !auction.Estado.EsAceptandoPujas() {
		return nil, domain.ErrSubastaCerrada
	}
	minRequired := auction.PrecioActual + auction.CalcularIncrementoMinimo()
	if cmd.AgentID == "" || cmd.MaxAuthorizedAmount < minRequired {
		return nil, domain.ErrAgentMandateInvalid
	}
	strategy := cmd.Strategy
	if strategy == "" {
		strategy = domain.StrategyMinimumIncrement
	}
	ttl := cmd.TTL
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	id, err := newID("mnd_")
	if err != nil {
		return nil, err
	}
	now := s.now()
	mandate := &domain.AgentMandate{
		ID:                  id,
		AuctionID:           cmd.AuctionID,
		PrincipalBidderID:   cmd.PrincipalBidderID,
		AgentID:             cmd.AgentID,
		MaxAuthorizedAmount: cmd.MaxAuthorizedAmount,
		Strategy:            strategy,
		ExpiresAt:           now.Add(ttl),
		CreatedAt:           now,
	}
	s.mu.Lock()
	s.mandates[mandate.ID] = mandate
	s.mu.Unlock()
	return mandate, nil
}

func (s *AgentDelegationService) ValidateMandateForBid(mandateID, auctionID, bidderID string, amount float64, now time.Time) error {
	s.mu.RLock()
	mandate, ok := s.mandates[mandateID]
	s.mu.RUnlock()
	if !ok || mandate.AuctionID != auctionID || mandate.PrincipalBidderID != bidderID {
		return domain.ErrAgentMandateInvalid
	}
	return mandate.CanAuthorizeBid(amount, now)
}

func (s *AgentDelegationService) ExecuteAutonomousBid(ctx context.Context, cmd ExecuteAutoBidCmd) (BidResponse, error) {
	s.mu.RLock()
	mandate, ok := s.mandates[cmd.MandateID]
	s.mu.RUnlock()
	if !ok || mandate.AuctionID != cmd.AuctionID || mandate.PrincipalBidderID != cmd.BidderID {
		return BidResponse{}, domain.ErrAgentMandateInvalid
	}
	auction, err := s.auctions.ObtenerPorID(ctx, cmd.AuctionID)
	if err != nil {
		return BidResponse{}, err
	}
	nextAmount := auction.PrecioActual + auction.CalcularIncrementoMinimo()
	if err := mandate.CanAuthorizeBid(nextAmount, s.now()); err != nil {
		return BidResponse{}, err
	}
	return s.bids.PlaceBid(ctx, PlaceBidCmd{
		SubastaID:      cmd.AuctionID,
		PostorID:       cmd.BidderID,
		Monto:          nextAmount,
		AgentMandateID: mandate.ID,
	})
}

func (s *AgentDelegationService) CountActiveMandates(auctionID string, now time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, m := range s.mandates {
		if m.AuctionID == auctionID && m.IsValidAt(now) {
			count++
		}
	}
	return count
}

