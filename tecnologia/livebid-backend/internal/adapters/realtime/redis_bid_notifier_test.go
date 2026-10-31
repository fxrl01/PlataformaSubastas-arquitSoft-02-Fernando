package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

func TestBidEventContainsPublicAuctionData(t *testing.T) {
	event := BidEvent{
		Type: "bid.accepted", AuctionID: "sub-1", BidID: "bid-1",
		BidderID: "usr-1", Amount: 2750, Extended: true,
		CloseAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "bid.accepted" || decoded["auction_id"] != "sub-1" {
		t.Fatalf("unexpected event: %s", payload)
	}
	if _, exists := decoded[string(domain.RoleBidder)]; exists {
		t.Fatal("event must not expose role fields")
	}
}

func TestAuctionClosedEventContainsWinnerWhenPresent(t *testing.T) {
	event := AuctionClosedEvent{
		Type: "auction.closed", AuctionID: "sub-1",
		WinnerID: "usr-1", Amount: 2750,
		ClosedAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "auction.closed" || decoded["winner_id"] != "usr-1" ||
		decoded["amount"] != 2750.0 {
		t.Fatalf("unexpected closure event: %s", payload)
	}
}

func TestAuctionClosedEventOmitsWinnerWhenDeserted(t *testing.T) {
	payload, err := json.Marshal(AuctionClosedEvent{
		Type: "auction.closed", AuctionID: "sub-1",
		ClosedAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, exists := decoded["winner_id"]; exists {
		t.Fatalf("deserted auction must omit winner: %s", payload)
	}
}
