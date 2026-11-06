package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/persistence"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type countingUserRepo struct {
	user       *domain.User
	findIDHits int
}

func (r *countingUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return r.user, nil
}

func (r *countingUserRepo) FindByID(context.Context, string) (*domain.User, error) {
	r.findIDHits++
	if r.user == nil {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func (r *countingUserRepo) Save(_ context.Context, u *domain.User) error {
	r.user = u
	return nil
}

func (r *countingUserRepo) UpdateStatus(_ context.Context, _ string, status domain.UserStatus) error {
	r.user.Status = status
	return nil
}

func TestCachedUserRepositoryCachesFindByIDAndInvalidatesOnUpdateStatus(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("usr-1", "Fernando", "f@livebid.pe", "hash", now)
	user.Status = domain.UserPending

	inner := &countingUserRepo{user: user}
	cached := persistence.NewCachedUserRepository(inner, time.Minute)

	_, _ = cached.FindByID(context.Background(), "usr-1")
	_, _ = cached.FindByID(context.Background(), "usr-1")
	_, _ = cached.FindByID(context.Background(), "usr-1")

	if inner.findIDHits != 1 {
		t.Fatalf("expected 1 database lookup for 3 cached calls, got %d", inner.findIDHits)
	}

	if err := cached.UpdateStatus(context.Background(), "usr-1", domain.UserActive); err != nil {
		t.Fatalf("update status failed: %v", err)
	}

	updated, err := cached.FindByID(context.Background(), "usr-1")
	if err != nil {
		t.Fatalf("find after update failed: %v", err)
	}
	if updated.Status != domain.UserActive || inner.findIDHits != 2 {
		t.Fatalf("expected cache invalidation after status change, status=%s hits=%d", updated.Status, inner.findIDHits)
	}
}

