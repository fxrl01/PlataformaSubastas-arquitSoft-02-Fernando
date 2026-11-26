package persistence_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/persistence"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type stubAuctionRepo struct {
	auctions  map[string]*domain.Subasta
	getCalls  int32
	listCalls int32
	delay     time.Duration
}

func (r *stubAuctionRepo) ObtenerPorID(_ context.Context, id string) (*domain.Subasta, error) {
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	atomic.AddInt32(&r.getCalls, 1)
	s, ok := r.auctions[id]
	if !ok {
		return nil, domain.ErrSubastaNoEncontrada
	}
	copySub := *s
	return &copySub, nil
}

func (r *stubAuctionRepo) Guardar(_ context.Context, subasta *domain.Subasta) error {
	copySub := *subasta
	r.auctions[subasta.ID] = &copySub
	return nil
}

func (r *stubAuctionRepo) Listar(_ context.Context) ([]*domain.Subasta, error) {
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	atomic.AddInt32(&r.listCalls, 1)
	out := make([]*domain.Subasta, 0, len(r.auctions))
	for _, s := range r.auctions {
		copySub := *s
		out = append(out, &copySub)
	}
	return out, nil
}

type inMemoryDistCache struct {
	data    map[string][]byte
	failAll bool
}

func (c *inMemoryDistCache) Get(_ context.Context, key string) ([]byte, error) {
	if c.failAll {
		return nil, errors.New("redis down")
	}
	v, ok := c.data[key]
	if !ok {
		return nil, errors.New("cache miss")
	}
	return v, nil
}

func (c *inMemoryDistCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if c.failAll {
		return errors.New("redis down")
	}
	c.data[key] = value
	return nil
}

func (c *inMemoryDistCache) Del(_ context.Context, keys ...string) error {
	if c.failAll {
		return errors.New("redis down")
	}
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

func TestCachedAuctionRepositoryCacheAsideHitsAndInvalidates(t *testing.T) {
	now := time.Now()
	base := &stubAuctionRepo{
		auctions: map[string]*domain.Subasta{
			"sub-1": domain.NuevaSubasta("sub-1", "Reloj Vintage", "", 1500, now, now.Add(time.Hour)),
		},
	}
	cache := &inMemoryDistCache{data: make(map[string][]byte)}
	repo := persistence.NewCachedAuctionRepository(base, cache, 15*time.Second)

	// 1. Primera consulta -> Cache Miss (lee DB), segunda -> Cache Hit (no lee DB)
	first, err := repo.ObtenerPorID(context.Background(), "sub-1")
	if err != nil || first.Titulo != "Reloj Vintage" {
		t.Fatalf("unexpected first read: %v", err)
	}
	second, err := repo.ObtenerPorID(context.Background(), "sub-1")
	if err != nil || second.Titulo != "Reloj Vintage" {
		t.Fatalf("unexpected second read: %v", err)
	}
	if base.getCalls != 1 {
		t.Fatalf("expected 1 DB read on repeated ObtenerPorID, got %d", base.getCalls)
	}

	// 2. Listado -> Cache Miss luego Cache Hit
	if _, err := repo.Listar(context.Background()); err != nil {
		t.Fatalf("first list failed: %v", err)
	}
	if _, err := repo.Listar(context.Background()); err != nil {
		t.Fatalf("second list failed: %v", err)
	}
	if base.listCalls != 1 {
		t.Fatalf("expected 1 DB list call on cached Listar, got %d", base.listCalls)
	}

	// 3. Guardar invalida tanto el item como el listado
	first.PrecioActual = 2000
	if err := repo.Guardar(context.Background(), first); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	updated, err := repo.ObtenerPorID(context.Background(), "sub-1")
	if err != nil || updated.PrecioActual != 2000 || base.getCalls != 2 {
		t.Fatalf("expected cache invalidation after Guardar, got price=%.2f getCalls=%d err=%v", updated.PrecioActual, base.getCalls, err)
	}
}

func TestCachedAuctionRepositoryDegradesGracefullyWhenCacheFails(t *testing.T) {
	now := time.Now()
	base := &stubAuctionRepo{
		auctions: map[string]*domain.Subasta{
			"sub-2": domain.NuevaSubasta("sub-2", "Obra Arte", "", 3000, now, now.Add(time.Hour)),
		},
	}
	cache := &inMemoryDistCache{data: make(map[string][]byte), failAll: true}
	repo := persistence.NewCachedAuctionRepository(base, cache, 15*time.Second)

	got, err := repo.ObtenerPorID(context.Background(), "sub-2")
	if err != nil || got.ID != "sub-2" {
		t.Fatalf("expected transparent DB fallback when cache fails, got %+v err=%v", got, err)
	}
	list, err := repo.Listar(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("expected transparent DB list fallback when cache fails, got %d err=%v", len(list), err)
	}
}

func TestDefaultPoolOptionsFromEnvUsesSafeBounds(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "40")
	t.Setenv("DB_MIN_CONNS", "8")
	t.Setenv("DB_PGBOUNCER_MODE", "true")

	opts := persistence.DefaultPoolOptionsFromEnv()
	if opts.MaxConns != 40 || opts.MinConns != 8 || !opts.SimpleProtocol {
		t.Fatalf("unexpected pool options: %+v", opts)
	}
}

func TestCachedAuctionRepositorySingleflightPreventsCacheStampede(t *testing.T) {
	now := time.Now()
	base := &stubAuctionRepo{
		auctions: map[string]*domain.Subasta{
			"sub-sf": domain.NuevaSubasta("sub-sf", "Escultura Bronce", "", 5000, now, now.Add(time.Hour)),
		},
		delay: 25 * time.Millisecond,
	}
	// Cache deshabilitada (simula ráfaga de Cache Miss simultáneo) para verificar coalescencia pura
	cache := &inMemoryDistCache{data: make(map[string][]byte), failAll: true}
	repo := persistence.NewCachedAuctionRepository(base, cache, 15*time.Second)

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			if _, err := repo.Listar(context.Background()); err != nil {
				t.Errorf("concurrent Listar error: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := repo.ObtenerPorID(context.Background(), "sub-sf"); err != nil {
				t.Errorf("concurrent ObtenerPorID error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&base.listCalls); got != 1 {
		t.Fatalf("expected singleflight to coalesce 20 Listar calls into 1 DB call, got %d", got)
	}
	if got := atomic.LoadInt32(&base.getCalls); got != 1 {
		t.Fatalf("expected singleflight to coalesce 20 ObtenerPorID calls into 1 DB call, got %d", got)
	}
}
