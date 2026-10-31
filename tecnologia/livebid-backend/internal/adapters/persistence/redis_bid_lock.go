package persistence

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/redis/go-redis/v9"
)

var ErrBidLockUnavailable = errors.New("redis bid lock unavailable")

const releaseLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

type RedisBidLock struct {
	client *redis.Client
	prefix string
}

func NewRedisBidLock(client *redis.Client) *RedisBidLock {
	return &RedisBidLock{client: client, prefix: "livebid:auction-lock:"}
}

func (l *RedisBidLock) AdquirirLock(ctx context.Context, auctionID string, ttl time.Duration) (domain.UnlockFunc, error) {
	valueBytes := make([]byte, 16)
	if _, err := rand.Read(valueBytes); err != nil {
		return nil, err
	}
	value := hex.EncodeToString(valueBytes)
	key := l.prefix + auctionID
	acquired, err := l.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrBidLockUnavailable
	}
	unlock := func() error {
		_, err := l.client.Eval(ctx, releaseLockScript, []string{key}, value).Result()
		return err
	}
	return unlock, nil
}
