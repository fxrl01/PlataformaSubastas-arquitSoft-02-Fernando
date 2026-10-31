package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/redis/go-redis/v9"
)

type BidEvent struct {
	Type      string    `json:"type"`
	AuctionID string    `json:"auction_id"`
	BidID     string    `json:"bid_id"`
	BidderID  string    `json:"bidder_id"`
	Amount    float64   `json:"amount"`
	Extended  bool      `json:"extended"`
	CloseAt   time.Time `json:"close_at"`
}

type RedisBidNotifier struct {
	client *redis.Client
	prefix string
}

func NewRedisBidNotifier(client *redis.Client) *RedisBidNotifier {
	return &RedisBidNotifier{client: client, prefix: "livebid:auction-events:"}
}

func (n *RedisBidNotifier) BroadcastNuevaPuja(ctx context.Context, auctionID string, bid domain.Puja, extended bool, closeAt time.Time) error {
	event, err := json.Marshal(BidEvent{
		Type: "bid.accepted", AuctionID: auctionID, BidID: bid.ID,
		BidderID: bid.PostorID, Amount: bid.Monto, Extended: extended, CloseAt: closeAt,
	})
	if err != nil {
		return fmt.Errorf("marshal bid event: %w", err)
	}
	if err := n.client.Publish(ctx, n.prefix+auctionID, event).Err(); err != nil {
		return fmt.Errorf("publish bid event: %w", err)
	}
	return nil
}

func RedisChannelPrefix() string {
	return "livebid:auction-events:"
}
