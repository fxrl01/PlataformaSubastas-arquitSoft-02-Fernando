package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type AuthService struct {
	users         domain.UserRepository
	hasher        domain.PasswordHasher
	tokens        domain.TokenIssuer
	refreshTokens domain.RefreshTokenStore
	now           func() time.Time
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
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserView  `json:"user"`
}

type UserView struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Email  string            `json:"email"`
	Role   domain.UserRole   `json:"role"`
	Status domain.UserStatus `json:"status"`
}

func NewAuthService(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenIssuer, refreshTokens ...domain.RefreshTokenStore) *AuthService {
	var store domain.RefreshTokenStore
	if len(refreshTokens) > 0 {
		store = refreshTokens[0]
	}
	return &AuthService{users: users, hasher: hasher, tokens: tokens, refreshTokens: store, now: time.Now}
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
	response := AuthResponse{AccessToken: token, ExpiresAt: expiresAt, User: toUserView(user)}
	if s.refreshTokens != nil {
		refreshToken, err := s.createRefreshToken(ctx, user.ID)
		if err != nil {
			return AuthResponse{}, err
		}
		response.RefreshToken = refreshToken
	}
	return response, nil
}

func (s *AuthService) ActivateUser(ctx context.Context, id string) error {
	if _, err := s.users.FindByID(ctx, id); err != nil {
		return err
	}
	return s.users.UpdateStatus(ctx, id, domain.UserActive)
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (AuthResponse, error) {
	if s.refreshTokens == nil || rawToken == "" {
		return AuthResponse{}, domain.ErrRefreshTokenInvalid
	}
	hash := sha256.Sum256([]byte(rawToken))
	userID, err := s.refreshTokens.Consume(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		return AuthResponse{}, domain.ErrRefreshTokenInvalid
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil || user.Status != domain.UserActive {
		return AuthResponse{}, domain.ErrRefreshTokenInvalid
	}
	expiresAt := s.now().Add(15 * time.Minute)
	accessToken, err := s.tokens.Issue(domain.TokenClaims{UserID: user.ID, Role: user.Role}, expiresAt)
	if err != nil {
		return AuthResponse{}, err
	}
	refreshToken, err := s.createRefreshToken(ctx, user.ID)
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt, User: toUserView(user)}, nil
}

func (s *AuthService) createRefreshToken(ctx context.Context, userID string) (string, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(rawBytes)
	hash := sha256.Sum256([]byte(raw))
	expiresAt := s.now().Add(7 * 24 * time.Hour)
	if err := s.refreshTokens.Save(ctx, hex.EncodeToString(hash[:]), userID, expiresAt); err != nil {
		return "", err
	}
	return raw, nil
}

func toUserView(user *domain.User) UserView {
	return UserView{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status}
}
