package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type AuthService struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
	tokens domain.TokenIssuer
	now    func() time.Time
}

type RegisterUserCmd struct {
	Name     string
	Email    string
	Password string
}

type LoginCmd struct {
	Email    string
	Password string
}

type AuthResponse struct {
	AccessToken string
	ExpiresAt   time.Time
	User        UserView
}

type UserView struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Email  string            `json:"email"`
	Role   domain.UserRole   `json:"role"`
	Status domain.UserStatus `json:"status"`
}

func NewAuthService(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenIssuer) *AuthService {
	return &AuthService{users: users, hasher: hasher, tokens: tokens, now: time.Now}
}

func (s *AuthService) Register(ctx context.Context, cmd RegisterUserCmd) (UserView, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if len(cmd.Password) < 8 {
		return UserView{}, domain.ErrInvalidPassword
	}
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return UserView{}, domain.ErrUserAlreadyExists
	} else if !errors.Is(err, domain.ErrUserNotFound) && err != nil {
		return UserView{}, err
	}
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return UserView{}, err
	}
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		return UserView{}, err
	}
	now := s.now()
	user, err := domain.NewUser("usr_"+hex.EncodeToString(idBytes), cmd.Name, email, hash, now)
	if err != nil {
		return UserView{}, err
	}
	if err := s.users.Save(ctx, user); err != nil {
		return UserView{}, err
	}
	return toUserView(user), nil
}

func (s *AuthService) Login(ctx context.Context, cmd LoginCmd) (AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(cmd.Email)))
	if err != nil || user == nil {
		return AuthResponse{}, domain.ErrInvalidCredentials
	}
	if user.Status != domain.UserActive {
		return AuthResponse{}, domain.ErrInvalidCredentials
	}
	if err := s.hasher.Compare(user.PasswordHash, cmd.Password); err != nil {
		return AuthResponse{}, domain.ErrInvalidCredentials
	}
	expiresAt := s.now().Add(15 * time.Minute)
	token, err := s.tokens.Issue(domain.TokenClaims{UserID: user.ID, Role: user.Role}, expiresAt)
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{AccessToken: token, ExpiresAt: expiresAt, User: toUserView(user)}, nil
}

func toUserView(user *domain.User) UserView {
	return UserView{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status}
}
