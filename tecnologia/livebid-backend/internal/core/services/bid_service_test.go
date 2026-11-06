package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type mockBidRepository struct {
	registered []domain.Puja
	idempotent map[string]domain.Puja
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

func (r *mockBidRepository) BuscarPorIdempotencyKey(_ context.Context, _, _, key string) (*domain.Puja, error) {
	bid, ok := r.idempotent[key]
	if !ok {
		return nil, nil
	}
	return &bid, nil
}

func (r *mockBidRepository) RegistrarPujaAtomica(_ context.Context, _ string, puja domain.Puja, _ float64, _ time.Time, _ bool, key string) error {
	r.registered = append(r.registered, puja)
	if key != "" {
		if r.idempotent == nil {
			r.idempotent = make(map[string]domain.Puja)
		}
		r.idempotent[key] = puja
	}
	return nil
}

func (r *mockBidRepository) ListarPorSubasta(_ context.Context, subastaID, bidderID string, minimumAmount float64, limit, offset int) ([]domain.Puja, int, error) {
	result := make([]domain.Puja, 0)
	for _, bid := range r.registered {
		if bid.SubastaID == subastaID && (bidderID == "" || bid.PostorID == bidderID) &&
			(minimumAmount == 0 || bid.Monto >= minimumAmount) {
			result = append(result, bid)
		}
	}
	if offset > len(result) {
		offset = len(result)
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], len(result), nil
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

func TestPlaceBidReplaysIdempotentRequest(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	_ = auction.Activar()
	bidRepo := &mockBidRepository{}
	service := services.NewAuctionBidService(&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, bidRepo)
	cmd := services.PlaceBidCmd{SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 3000, IdempotencyKey: "request-1"}

	first, err := service.PlaceBid(context.Background(), cmd)
	if err != nil {
		t.Fatalf("first bid failed: %v", err)
	}
	second, err := service.PlaceBid(context.Background(), cmd)
	if err != nil {
		t.Fatalf("replayed bid failed: %v", err)
	}
	if first.PujaID != second.PujaID || len(bidRepo.registered) != 1 {
		t.Fatalf("request was not replayed: first=%+v second=%+v bids=%d", first, second, len(bidRepo.registered))
	}
}

func TestPlaceBidRejectsIdempotencyKeyWithDifferentAmount(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	_ = auction.Activar()
	bidRepo := &mockBidRepository{}
	service := services.NewAuctionBidService(&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, bidRepo)
	_, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 3000, IdempotencyKey: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-1", PostorID: "usr-bidder", Monto: 3500, IdempotencyKey: "request-1",
	})
	if err != domain.ErrIdempotencyKeyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestPlaceBidVerifiesEIP712SignatureWhenProvided(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	_ = auction.Activar()
	bidRepo := &mockBidRepository{}
	verifier := web3.NewEIP712Verifier(8453, "0x1111111111111111111111111111111111111111")
	service := services.NewAuctionBidService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, bidRepo,
	).WithWeb3Verifier(verifier)

	wallet := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
	deadline := now.Add(10 * time.Minute).Unix()

	// Invalid signature must be rejected before persisting
	_, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID:     "sub-1",
		PostorID:      "usr-bidder",
		Monto:         3000,
		WalletAddress: wallet,
		Signature:     "0xdeadbeef",
		Nonce:         "n-1",
		Deadline:      deadline,
	})
	if err != domain.ErrInvalidWeb3Signature || len(bidRepo.registered) != 0 {
		t.Fatalf("expected ErrInvalidWeb3Signature and 0 persisted bids, got err=%v len=%d", err, len(bidRepo.registered))
	}

	// Valid EIP-712 signature must succeed and return eip712_digest + hash_auditoria
	validSig := verifier.SignBidEnvelope(domain.EIP712BidEnvelope{
		AuctionID:     "sub-1",
		BidderID:      "usr-bidder",
		WalletAddress: wallet,
		Amount:        3000,
		Nonce:         "n-1",
		Deadline:      deadline,
	})
	res, err := service.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID:     "sub-1",
		PostorID:      "usr-bidder",
		Monto:         3000,
		WalletAddress: wallet,
		Signature:     validSig,
		Nonce:         "n-1",
		Deadline:      deadline,
	})
	if err != nil {
		t.Fatalf("expected valid EIP-712 bid to succeed, got %v", err)
	}
	if res.EIP712Digest == "" || res.HashAuditoria == "" {
		t.Fatalf("expected EIP712Digest and HashAuditoria in response, got %+v", res)
	}
}

func TestAgentDelegationEnforcesBudgetCeilingAndExecutesAutonomousBids(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-ai", "Servidor GPU", "", 10000, now, now.Add(time.Hour))
	_ = auction.Activar()
	auctionRepo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-ai": auction}}
	bidRepo := &mockBidRepository{}
	bidService := services.NewAuctionBidService(auctionRepo, bidRepo)
	delegation := services.NewAgentDelegationService(auctionRepo, bidService)

	// Authorize an AI agent for usr-principal with a strict ceiling of 11,000
	mandate, err := delegation.CreateMandate(context.Background(), services.CreateMandateCmd{
		AuctionID:           "sub-ai",
		PrincipalBidderID:   "usr-principal",
		AgentID:             "agt_gemini_01",
		MaxAuthorizedAmount: 11000,
		Strategy:            domain.StrategyMinimumIncrement,
	})
	if err != nil {
		t.Fatalf("create mandate failed: %v", err)
	}

	// First autonomous bid: 10,000 + 500 = 10,500 (<= 11,000 ceiling) -> succeeds
	firstAuto, err := delegation.ExecuteAutonomousBid(context.Background(), services.ExecuteAutoBidCmd{
		AuctionID: "sub-ai",
		BidderID:  "usr-principal",
		MandateID: mandate.ID,
	})
	if err != nil || firstAuto.PrecioActual != 10500 || firstAuto.AgentMandateID != mandate.ID {
		t.Fatalf("expected autonomous bid at 10500 to succeed, got res=%+v err=%v", firstAuto, err)
	}

	// Rival bidder raises price to 11,000
	_, err = bidService.PlaceBid(context.Background(), services.PlaceBidCmd{
		SubastaID: "sub-ai",
		PostorID:  "usr-rival",
		Monto:     11000,
	})
	if err != nil {
		t.Fatalf("rival bid failed: %v", err)
	}

	// Next autonomous bid would require 11,500 (> 11,000 ceiling) -> must be rejected by Zero-Trust mandate!
	_, err = delegation.ExecuteAutonomousBid(context.Background(), services.ExecuteAutoBidCmd{
		AuctionID: "sub-ai",
		BidderID:  "usr-principal",
		MandateID: mandate.ID,
	})
	if err != domain.ErrAgentMandateExceeded {
		t.Fatalf("expected ErrAgentMandateExceeded when next bid exceeds 11000 ceiling, got %v", err)
	}
}
