package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type authUserRepo struct {
	user *domain.User
}

func (r *authUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if r.user == nil || r.user.Email != email {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func (r *authUserRepo) Save(_ context.Context, user *domain.User) error {
	r.user = user
	return nil
}

type authHasher struct{}

func (authHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", domain.ErrInvalidPassword
	}
	return "hash:" + password, nil
}

func (authHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return domain.ErrInvalidCredentials
	}
	return nil
}

type authTokenIssuer struct{}

func (authTokenIssuer) Issue(claims domain.TokenClaims, expiresAt time.Time) (string, error) {
	return claims.UserID + "|" + string(claims.Role) + "|" + expiresAt.UTC().Format(time.RFC3339), nil
}

func TestRegisterCreatesPendingBidderWithoutPasswordInView(t *testing.T) {
	repo := &authUserRepo{}
	service := services.NewAuthService(repo, authHasher{}, authTokenIssuer{})

	view, err := service.Register(context.Background(), services.RegisterUserCmd{
		Name: "Ana", Email: "ANA@example.com", Password: "secret123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if view.Email != "ana@example.com" || view.Role != domain.RoleBidder || view.Status != domain.UserPending {
		t.Fatalf("unexpected user view: %+v", view)
	}
	if repo.user.PasswordHash == "secret" {
		t.Fatal("plaintext password was persisted")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	now := time.Now()
	existing, _ := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash:secret123", now)
	service := services.NewAuthService(&authUserRepo{user: existing}, authHasher{}, authTokenIssuer{})

	_, err := service.Register(context.Background(), services.RegisterUserCmd{
		Name: "Otra", Email: "ana@example.com", Password: "secret123",
	})
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestLoginRejectsInactiveUserWithoutRevealingStatus(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash:secret123", now)
	service := services.NewAuthService(&authUserRepo{user: user}, authHasher{}, authTokenIssuer{})

	_, err := service.Login(context.Background(), services.LoginCmd{
		Email: "ana@example.com", Password: "secret123",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected generic credentials error, got %v", err)
	}
}

func TestLoginIssuesTokenForActiveUser(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash:secret123", now)
	user.Status = domain.UserActive
	service := services.NewAuthService(&authUserRepo{user: user}, authHasher{}, authTokenIssuer{})

	response, err := service.Login(context.Background(), services.LoginCmd{
		Email: "ana@example.com", Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if response.AccessToken == "" || response.User.ID != "usr-1" {
		t.Fatalf("unexpected login response: %+v", response)
	}
}
