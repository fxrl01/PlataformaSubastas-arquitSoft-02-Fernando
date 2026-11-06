package realtime

import (
	"context"
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

type fakePubSubStream struct {
	ch     chan []byte
	closed bool
}

func (f *fakePubSubStream) Messages() <-chan []byte { return f.ch }
func (f *fakePubSubStream) Close() error {
	f.closed = true
	return nil
}

func TestAuctionRoomHubMultiplexesSingleSubscriptionAcrossMultipleClients(t *testing.T) {
	subscribeCount := 0
	var activeStream *fakePubSubStream

	hub := NewAuctionRoomHub(func(_ context.Context, _ string) (PubSubStream, error) {
		subscribeCount++
		activeStream = &fakePubSubStream{ch: make(chan []byte, 8)}
		return activeStream, nil
	})

	ch1, leave1, err := hub.Join("sub-1")
	if err != nil {
		t.Fatalf("join 1 failed: %v", err)
	}
	ch2, leave2, err := hub.Join("sub-1")
	if err != nil {
		t.Fatalf("join 2 failed: %v", err)
	}

	if subscribeCount != 1 || hub.ActiveRooms() != 1 {
		t.Fatalf("expected 1 broker subscription for 2 clients in same room, got subs=%d rooms=%d", subscribeCount, hub.ActiveRooms())
	}

	activeStream.ch <- []byte(`{"type":"bid.accepted","amount":5000}`)

	for i, ch := range []<-chan []byte{ch1, ch2} {
		select {
		case msg := <-ch:
			if string(msg) != `{"type":"bid.accepted","amount":5000}` {
				t.Fatalf("client %d got unexpected payload: %s", i+1, msg)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d timed out waiting for fan-out message", i+1)
		}
	}

	leave1()
	if hub.ActiveRooms() != 1 || activeStream.closed {
		t.Fatal("room should remain active while client 2 is still connected")
	}

	leave2()
	if hub.ActiveRooms() != 0 || !activeStream.closed {
		t.Fatal("room should close broker subscription when last client leaves")
	}
}
