package persistence

import (
	"context"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type cachedUserEntry struct {
	user      domain.User
	expiresAt time.Time
}

// CachedUserRepository decora un domain.UserRepository con caché TTL en memoria
// para evitar consultas repetitivas a PostgreSQL en el middleware RequireAuth durante ráfagas de pujas.
type CachedUserRepository struct {
	mu      sync.RWMutex
	inner   domain.UserRepository
	ttl     time.Duration
	entries map[string]cachedUserEntry
	now     func() time.Time
}

func NewCachedUserRepository(inner domain.UserRepository, ttl time.Duration) *CachedUserRepository {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedUserRepository{
		inner:   inner,
		ttl:     ttl,
		entries: make(map[string]cachedUserEntry),
		now:     time.Now,
	}
}

func (r *CachedUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.inner.FindByEmail(ctx, email)
}

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	now := r.now()
	r.mu.RLock()
	entry, ok := r.entries[id]
	r.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		copyUser := entry.user
		return &copyUser, nil
	}

	user, err := r.inner.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.entries[id] = cachedUserEntry{
		user:      *user,
		expiresAt: now.Add(r.ttl),
	}
	r.mu.Unlock()

	copyUser := *user
	return &copyUser, nil
}

func (r *CachedUserRepository) ListUsers(ctx context.Context, status domain.UserStatus, role domain.UserRole) ([]*domain.User, error) {
	if adminRepo, ok := r.inner.(domain.AdminUserRepository); ok {
		return adminRepo.ListUsers(ctx, status, role)
	}
	return []*domain.User{}, nil
}

func (r *CachedUserRepository) Save(ctx context.Context, user *domain.User) error {
	if err := r.inner.Save(ctx, user); err != nil {
		return err
	}
	if user != nil {
		r.Invalidate(user.ID)
	}
	return nil
}

func (r *CachedUserRepository) UpdateStatus(ctx context.Context, id string, status domain.UserStatus) error {
	if err := r.inner.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	r.Invalidate(id)
	return nil
}

func (r *CachedUserRepository) UpdateKYCAndStatus(ctx context.Context, id string, status domain.UserStatus, kycApproved bool) error {
	if adminRepo, ok := r.inner.(domain.AdminUserRepository); ok {
		if err := adminRepo.UpdateKYCAndStatus(ctx, id, status, kycApproved); err != nil {
			return err
		}
		r.Invalidate(id)
		return nil
	}
	return r.UpdateStatus(ctx, id, status)
}

func (r *CachedUserRepository) UpdateRole(ctx context.Context, id string, role domain.UserRole) error {
	if adminRepo, ok := r.inner.(domain.AdminUserRepository); ok {
		if err := adminRepo.UpdateRole(ctx, id, role); err != nil {
			return err
		}
		r.Invalidate(id)
		return nil
	}
	return nil
}

func (r *CachedUserRepository) Invalidate(id string) {
	r.mu.Lock()
	delete(r.entries, id)
	r.mu.Unlock()
}
