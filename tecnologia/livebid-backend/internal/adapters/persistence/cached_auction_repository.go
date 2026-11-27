package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	auctionListCacheKey   = "livebid:cache:subastas:list"
	auctionItemKeyPrefix  = "livebid:cache:subasta:"
	defaultAuctionCacheTT = 15 * time.Second
)

// DistributedCache abstrae operaciones clave-valor efímeras para el patrón Cache-Aside.
type DistributedCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// RedisDistributedCache implementa DistributedCache sobre Redis 7.
type RedisDistributedCache struct {
	client *redis.Client
}

func NewRedisDistributedCache(client *redis.Client) *RedisDistributedCache {
	return &RedisDistributedCache{client: client}
}

func (c *RedisDistributedCache) Get(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("redis cache disabled")
	}
	return c.client.Get(ctx, key).Bytes()
}

func (c *RedisDistributedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisDistributedCache) Del(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// CachedAuctionRepository aplica el patrón Cache-Aside distribuido en Redis con coalescencia singleflight
// (anti Cache Stampede / Thundering Herd) sobre un domain.SubastaRepository, degradando a PostgreSQL ante fallos.
type CachedAuctionRepository struct {
	inner domain.SubastaRepository
	cache DistributedCache
	ttl   time.Duration
	sf    singleflight.Group
}

func NewCachedAuctionRepository(inner domain.SubastaRepository, cache DistributedCache, ttl time.Duration) *CachedAuctionRepository {
	if ttl <= 0 {
		ttl = defaultAuctionCacheTT
	}
	return &CachedAuctionRepository{
		inner: inner,
		cache: cache,
		ttl:   ttl,
	}
}

func cloneAuction(src *domain.Subasta) *domain.Subasta {
	if src == nil {
		return nil
	}
	cp := *src
	if cp.Pujas == nil {
		cp.Pujas = make([]domain.Puja, 0)
	}
	return &cp
}

func (r *CachedAuctionRepository) ObtenerPorID(ctx context.Context, id string) (*domain.Subasta, error) {
	if r.cache != nil && id != "" {
		if raw, err := r.cache.Get(ctx, auctionItemKeyPrefix+id); err == nil && len(raw) > 0 {
			var cached domain.Subasta
			if json.Unmarshal(raw, &cached) == nil {
				if cached.Pujas == nil {
					cached.Pujas = make([]domain.Puja, 0)
				}
				return &cached, nil
			}
		}
	}

	val, err, _ := r.sf.Do(auctionItemKeyPrefix+id, func() (any, error) {
		subasta, err := r.inner.ObtenerPorID(ctx, id)
		if err != nil {
			return nil, err
		}
		if r.cache != nil && subasta != nil {
			if encoded, mErr := json.Marshal(subasta); mErr == nil {
				_ = r.cache.Set(ctx, auctionItemKeyPrefix+id, encoded, r.ttl)
			}
		}
		return subasta, nil
	})
	if err != nil {
		return nil, err
	}
	return cloneAuction(val.(*domain.Subasta)), nil
}

func (r *CachedAuctionRepository) Listar(ctx context.Context) ([]*domain.Subasta, error) {
	if r.cache != nil {
		if raw, err := r.cache.Get(ctx, auctionListCacheKey); err == nil && len(raw) > 0 {
			var cached []*domain.Subasta
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}

	val, err, _ := r.sf.Do(auctionListCacheKey, func() (any, error) {
		list, err := r.inner.Listar(ctx)
		if err != nil {
			return nil, err
		}
		if r.cache != nil {
			if encoded, mErr := json.Marshal(list); mErr == nil {
				_ = r.cache.Set(ctx, auctionListCacheKey, encoded, r.ttl)
			}
		}
		return list, nil
	})
	if err != nil {
		return nil, err
	}
	src := val.([]*domain.Subasta)
	out := make([]*domain.Subasta, len(src))
	for i, s := range src {
		out[i] = cloneAuction(s)
	}
	return out, nil
}

func (r *CachedAuctionRepository) ListarVencidas(ctx context.Context, now time.Time) ([]*domain.Subasta, error) {
	if finder, ok := r.inner.(domain.ExpiredSubastaRepository); ok {
		return finder.ListarVencidas(ctx, now)
	}
	return r.inner.Listar(ctx)
}

func (r *CachedAuctionRepository) Guardar(ctx context.Context, subasta *domain.Subasta) error {
	if err := r.inner.Guardar(ctx, subasta); err != nil {
		return err
	}
	if subasta != nil {
		r.InvalidateAuction(ctx, subasta.ID)
	}
	return nil
}

// InvalidateAuction elimina las claves en Redis de la subasta y del listado de catálogo.
func (r *CachedAuctionRepository) InvalidateAuction(ctx context.Context, id string) {
	if r.cache == nil {
		return
	}
	keys := []string{auctionListCacheKey}
	if id != "" {
		keys = append(keys, auctionItemKeyPrefix+id)
	}
	_ = r.cache.Del(ctx, keys...)
}
