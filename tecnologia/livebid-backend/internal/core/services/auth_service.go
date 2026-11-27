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
	Name          string
	Email         string
	Password      string
	RequestedRole domain.UserRole
	DocumentID    string
	Phone         string
	WalletAddress string
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
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	Role          domain.UserRole   `json:"role"`
	RequestedRole domain.UserRole   `json:"requested_role,omitempty"`
	Status        domain.UserStatus `json:"status"`
	KYCApproved   bool              `json:"kyc_aprobado"`
	DocumentID    string            `json:"documento_identidad,omitempty"`
	Phone         string            `json:"telefono,omitempty"`
	WalletAddress string            `json:"wallet_address,omitempty"`
	Bio           string            `json:"bio,omitempty"`
	CreatedAt     time.Time         `json:"created_at,omitempty"`
}

type UpdateProfileCmd struct {
	Name          string
	Phone         string
	DocumentID    string
	WalletAddress string
	Bio           string
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
	if cmd.RequestedRole != "" && !domain.IsValidRole(cmd.RequestedRole) {
		return UserView{}, domain.ErrInvalidRole
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
	if cmd.RequestedRole != "" && cmd.RequestedRole != domain.RoleAdmin {
		user.RequestedRole = cmd.RequestedRole
	}
	user.DocumentID = strings.TrimSpace(cmd.DocumentID)
	user.Phone = strings.TrimSpace(cmd.Phone)
	user.WalletAddress = strings.TrimSpace(cmd.WalletAddress)

	if err := s.users.Save(ctx, user); err != nil {
		return UserView{}, err
	}
	return toUserView(user), nil
}

// BootstrapAdmin garantiza que exista una cuenta ADMIN activa en el arranque del sistema.
func (s *AuthService) BootstrapAdmin(ctx context.Context, name, email, password string) (UserView, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return UserView{}, domain.ErrInvalidUser
	}
	existing, err := s.users.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		if adminRepo, ok := s.users.(domain.AdminUserRepository); ok {
			if existing.Role != domain.RoleAdmin {
				_ = adminRepo.UpdateRole(ctx, existing.ID, domain.RoleAdmin)
				existing.Role = domain.RoleAdmin
			}
			if existing.Status != domain.UserActive || !existing.KYCApproved {
				_ = adminRepo.UpdateKYCAndStatus(ctx, existing.ID, domain.UserActive, true)
				existing.Status = domain.UserActive
				existing.KYCApproved = true
			}
		} else if existing.Status != domain.UserActive {
			_ = s.users.UpdateStatus(ctx, existing.ID, domain.UserActive)
			existing.Status = domain.UserActive
		}
		return toUserView(existing), nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) && err != nil {
		return UserView{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return UserView{}, err
	}
	now := s.now()
	if strings.TrimSpace(name) == "" {
		name = "Administrador del Sistema"
	}
	adminUser, err := domain.NewUser("usr_admin_root", name, email, hash, now)
	if err != nil {
		return UserView{}, err
	}
	adminUser.Role = domain.RoleAdmin
	adminUser.RequestedRole = domain.RoleAdmin
	adminUser.Status = domain.UserActive
	adminUser.KYCApproved = true

	if err := s.users.Save(ctx, adminUser); err != nil {
		return UserView{}, err
	}
	return toUserView(adminUser), nil
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
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if adminRepo, ok := s.users.(domain.AdminUserRepository); ok {
		if user.RequestedRole != "" && user.RequestedRole != user.Role && user.RequestedRole != domain.RoleAdmin {
			_ = adminRepo.UpdateRole(ctx, id, user.RequestedRole)
		}
		return adminRepo.UpdateKYCAndStatus(ctx, id, domain.UserActive, true)
	}
	return s.users.UpdateStatus(ctx, id, domain.UserActive)
}

func (s *AuthService) ListUsers(ctx context.Context, status domain.UserStatus, role domain.UserRole) ([]UserView, error) {
	if status != "" && !domain.IsValidUserStatus(status) {
		return nil, domain.ErrInvalidUserStatus
	}
	if role != "" && !domain.IsValidRole(role) {
		return nil, domain.ErrInvalidRole
	}
	adminRepo, ok := s.users.(domain.AdminUserRepository)
	if !ok {
		return []UserView{}, nil
	}
	users, err := adminRepo.ListUsers(ctx, status, role)
	if err != nil {
		return nil, err
	}
	views := make([]UserView, 0, len(users))
	for _, u := range users {
		views = append(views, toUserView(u))
	}
	return views, nil
}

func (s *AuthService) UpdateUserRole(ctx context.Context, id string, role domain.UserRole) (UserView, error) {
	if !domain.IsValidRole(role) {
		return UserView{}, domain.ErrInvalidRole
	}
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	adminRepo, ok := s.users.(domain.AdminUserRepository)
	if !ok {
		user.Role = role
		return toUserView(user), nil
	}
	if err := adminRepo.UpdateRole(ctx, id, role); err != nil {
		return UserView{}, err
	}
	updated, err := s.users.FindByID(ctx, id)
	if err != nil {
		user.Role = role
		return toUserView(user), nil
	}
	return toUserView(updated), nil
}

func (s *AuthService) UpdateUserStatus(ctx context.Context, id string, status domain.UserStatus, kycApproved *bool) (UserView, error) {
	if !domain.IsValidUserStatus(status) {
		return UserView{}, domain.ErrInvalidUserStatus
	}
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	approved := user.KYCApproved
	if kycApproved != nil {
		approved = *kycApproved
	} else if status == domain.UserActive {
		approved = true
	}

	if adminRepo, ok := s.users.(domain.AdminUserRepository); ok {
		if err := adminRepo.UpdateKYCAndStatus(ctx, id, status, approved); err != nil {
			return UserView{}, err
		}
	} else {
		if err := s.users.UpdateStatus(ctx, id, status); err != nil {
			return UserView{}, err
		}
	}
	updated, err := s.users.FindByID(ctx, id)
	if err != nil {
		user.Status = status
		user.KYCApproved = approved
		return toUserView(user), nil
	}
	return toUserView(updated), nil
}

func (s *AuthService) GetProfile(ctx context.Context, id string) (UserView, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return toUserView(user), nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, id string, cmd UpdateProfileCmd) (UserView, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		name = user.Name
	}
	phone := strings.TrimSpace(cmd.Phone)
	doc := strings.TrimSpace(cmd.DocumentID)
	if doc == "" {
		doc = user.DocumentID
	}
	wallet := strings.TrimSpace(cmd.WalletAddress)
	bio := strings.TrimSpace(cmd.Bio)

	if updater, ok := s.users.(domain.UserProfileUpdater); ok {
		if err := updater.UpdateProfile(ctx, id, name, phone, doc, wallet, bio); err != nil {
			return UserView{}, err
		}
		if updated, err := s.users.FindByID(ctx, id); err == nil {
			return toUserView(updated), nil
		}
	}
	user.Name = name
	user.Phone = phone
	user.DocumentID = doc
	user.WalletAddress = wallet
	user.Bio = bio
	return toUserView(user), nil
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
	return UserView{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		Role:          user.Role,
		RequestedRole: user.RequestedRole,
		Status:        user.Status,
		KYCApproved:   user.KYCApproved,
		DocumentID:    user.DocumentID,
		Phone:         user.Phone,
		WalletAddress: user.WalletAddress,
		Bio:           user.Bio,
		CreatedAt:     user.CreatedAt,
	}
}
