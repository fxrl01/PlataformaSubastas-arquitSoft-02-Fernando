package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type middlewareUserRepo struct {
	user *domain.User
}

func (r middlewareUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r middlewareUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func (r middlewareUserRepo) Save(context.Context, *domain.User) error {
	return nil
}

func (r middlewareUserRepo) UpdateStatus(context.Context, string, domain.UserStatus) error {
	return nil
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	issuer, err := security.NewJWTIssuer("test-secret")
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("protected handler should not be called")
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	response := httptest.NewRecorder()

	handlers.RequireAuth(issuer, middlewareUserRepo{}, next).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestRequireAuthAcceptsActiveUser(t *testing.T) {
	now := time.Now()
	user, err := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash", now)
	if err != nil {
		t.Fatal(err)
	}
	user.Status = domain.UserActive
	issuer, err := security.NewJWTIssuer("test-secret")
	if err != nil {
		t.Fatal(err)
	}
	rawToken, err := issuer.Issue(domain.TokenClaims{UserID: user.ID, Role: user.Role}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := handlers.CurrentUser(r)
		if !ok || current.ID != user.ID || current.Role != user.Role {
			t.Fatal("authenticated identity was not propagated")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+rawToken)
	response := httptest.NewRecorder()

	handlers.RequireAuth(issuer, middlewareUserRepo{user: user}, next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}
