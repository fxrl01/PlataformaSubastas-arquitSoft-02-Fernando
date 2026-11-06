package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func (r *authUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, domain.ErrUserNotFound
	}
	return r.user, nil
}

func (r *authUserRepo) Save(_ context.Context, user *domain.User) error {
	r.user = user
	return nil
}

func (r *authUserRepo) UpdateStatus(_ context.Context, id string, status domain.UserStatus) error {
	if r.user == nil || r.user.ID != id {
		return domain.ErrUserNotFound
	}
	r.user.Status = status
	return nil
}

func (r *authUserRepo) ListUsers(_ context.Context, statusFilter domain.UserStatus, roleFilter domain.UserRole) ([]*domain.User, error) {
	if r.user == nil {
		return nil, nil
	}
	if roleFilter != "" && r.user.Role != roleFilter {
		return nil, nil
	}
	if statusFilter != "" && r.user.Status != statusFilter {
		return nil, nil
	}
	return []*domain.User{r.user}, nil
}

func (r *authUserRepo) UpdateRole(_ context.Context, id string, role domain.UserRole) error {
	if r.user == nil || r.user.ID != id {
		return domain.ErrUserNotFound
	}
	r.user.Role = role
	return nil
}

func (r *authUserRepo) UpdateKYCAndStatus(_ context.Context, id string, status domain.UserStatus, kycApproved bool) error {
	if r.user == nil || r.user.ID != id {
		return domain.ErrUserNotFound
	}
	r.user.Status = status
	r.user.KYCApproved = kycApproved
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

type refreshStore struct {
	tokens map[string]string
}

func (s *refreshStore) Save(_ context.Context, tokenHash, userID string, _ time.Time) error {
	if s.tokens == nil {
		s.tokens = map[string]string{}
	}
	s.tokens[tokenHash] = userID
	return nil
}

func (s *refreshStore) Consume(_ context.Context, tokenHash string) (string, error) {
	userID, ok := s.tokens[tokenHash]
	if !ok {
		return "", domain.ErrRefreshTokenInvalid
	}
	delete(s.tokens, tokenHash)
	return userID, nil
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

func TestRefreshTokenCanOnlyBeConsumedOnce(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash:secret123", now)
	user.Status = domain.UserActive
	store := &refreshStore{}
	service := services.NewAuthService(&authUserRepo{user: user}, authHasher{}, authTokenIssuer{}, store)

	login, err := service.Login(context.Background(), services.LoginCmd{Email: user.Email, Password: "secret123"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if login.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if _, err := service.Refresh(context.Background(), login.RefreshToken); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	hash := sha256.Sum256([]byte(login.RefreshToken))
	if _, err := store.Consume(context.Background(), hex.EncodeToString(hash[:])); err == nil {
		t.Fatal("refresh token was not revoked after use")
	}
}

func TestActivateUserChangesStatus(t *testing.T) {
	now := time.Now()
	user, _ := domain.NewUser("usr-1", "Ana", "ana@example.com", "hash:secret123", now)
	repo := &authUserRepo{user: user}
	service := services.NewAuthService(repo, authHasher{}, authTokenIssuer{})

	if err := service.ActivateUser(context.Background(), user.ID); err != nil {
		t.Fatalf("activation failed: %v", err)
	}
	if repo.user.Status != domain.UserActive {
		t.Fatalf("expected active status, got %s", repo.user.Status)
	}
}

func TestBootstrapAdminCreatesActiveAdminAndUpdatesRoleAndKYC(t *testing.T) {
	repo := &authUserRepo{}
	service := services.NewAuthService(repo, authHasher{}, authTokenIssuer{})

	if _, err := service.BootstrapAdmin(context.Background(), "Admin Root", "admin@livebid.pe", "AdminLiveBid2026!"); err != nil {
		t.Fatalf("bootstrap admin failed: %v", err)
	}
	if repo.user == nil || repo.user.Role != domain.RoleAdmin || repo.user.Status != domain.UserActive || !repo.user.KYCApproved {
		t.Fatalf("unexpected bootstrapped admin: %+v", repo.user)
	}

	view, err := service.UpdateUserRole(context.Background(), repo.user.ID, domain.RoleAuctioneer)
	if err != nil {
		t.Fatalf("update role failed: %v", err)
	}
	if view.Role != domain.RoleAuctioneer {
		t.Fatalf("expected role MARTILLERO, got %s", view.Role)
	}

	kycFalse := false
	statusView, err := service.UpdateUserStatus(context.Background(), repo.user.ID, domain.UserSuspended, &kycFalse)
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if statusView.Status != domain.UserSuspended || statusView.KYCApproved {
		t.Fatalf("expected suspended status and kyc=false, got %+v", statusView)
	}

	users, err := service.ListUsers(context.Background(), domain.UserSuspended, domain.RoleAuctioneer)
	if err != nil || len(users) != 1 {
		t.Fatalf("expected 1 listed user, err=%v len=%d", err, len(users))
	}
}
