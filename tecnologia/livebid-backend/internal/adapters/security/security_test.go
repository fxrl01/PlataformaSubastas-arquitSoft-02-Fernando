package security_test

import (
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherHashesAndVerifiesPasswords(t *testing.T) {
	hasher := security.NewBcryptHasher(bcrypt.MinCost)

	hash, err := hasher.Hash("ClaveSeguraLiveBid2026!")
	if err != nil {
		t.Fatalf("expected hash to succeed, got: %v", err)
	}
	if hash == "" || hash == "ClaveSeguraLiveBid2026!" {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}

	if err := hasher.Compare(hash, "ClaveSeguraLiveBid2026!"); err != nil {
		t.Fatalf("expected valid password comparison, got: %v", err)
	}
	if err := hasher.Compare(hash, "ClaveIncorrecta"); err == nil {
		t.Fatal("expected invalid password to fail comparison")
	}
}

func TestJWTIssuerIssuesAndVerifiesRoleClaims(t *testing.T) {
	if _, err := security.NewJWTIssuer(""); err != security.ErrEmptyJWTSecret {
		t.Fatalf("expected ErrEmptyJWTSecret on empty secret, got: %v", err)
	}

	issuer, err := security.NewJWTIssuer("super-secret-jwt-key-livebid-2026")
	if err != nil {
		t.Fatalf("unexpected error creating JWTIssuer: %v", err)
	}

	token, err := issuer.Issue(domain.TokenClaims{
		UserID: "usr-admin-01",
		Role:   domain.RoleAdmin,
	}, time.Now().Add(15*time.Minute))
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	verified, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("verify valid token failed: %v", err)
	}
	if verified.UserID != "usr-admin-01" || verified.Role != domain.RoleAdmin {
		t.Fatalf("unexpected verified claims: %+v", verified)
	}

	// Expired token must be rejected
	expiredToken, err := issuer.Issue(domain.TokenClaims{
		UserID: "usr-expired",
		Role:   domain.RoleBidder,
	}, time.Now().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("issue expired token failed: %v", err)
	}
	if _, err := issuer.Verify(expiredToken); err == nil {
		t.Fatal("expected expired JWT token to be rejected")
	}

	// Token verified with a different secret must be rejected
	otherIssuer, _ := security.NewJWTIssuer("different-secret-key")
	if _, err := otherIssuer.Verify(token); err == nil {
		t.Fatal("expected token signed with different secret to be rejected")
	}
}
