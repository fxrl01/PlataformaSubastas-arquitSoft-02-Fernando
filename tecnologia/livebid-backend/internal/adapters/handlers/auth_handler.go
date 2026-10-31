package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type authContextKey struct{}

type AuthenticatedUser struct {
	ID   string          `json:"id"`
	Role domain.UserRole `json:"role"`
}

func RequireAuth(issuer *security.JWTIssuer, users domain.UserRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		claims, err := issuer.Verify(header[len(prefix):])
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authentication token"})
			return
		}
		user, err := users.FindByID(r.Context(), claims.UserID)
		if err != nil || user.Status != domain.UserActive || user.Role != claims.Role {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authentication token"})
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, AuthenticatedUser{ID: user.ID, Role: user.Role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentUser(r *http.Request) (AuthenticatedUser, bool) {
	user, ok := r.Context().Value(authContextKey{}).(AuthenticatedUser)
	return user, ok
}

func RequireRoles(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := h.auth.Register(r.Context(), services.RegisterUserCmd{
		Name: request.Name, Email: request.Email, Password: request.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.auth.Login(r.Context(), services.LoginCmd{
		Email: request.Email, Password: request.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	response, err := h.auth.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.ActivateUser(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return false
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return false
	}
	return true
}

func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidUser),
		errors.Is(err, domain.ErrInvalidPassword):
		status, message = http.StatusBadRequest, "invalid registration data"
	case errors.Is(err, domain.ErrUserAlreadyExists):
		status, message = http.StatusConflict, "user already registered"
	case errors.Is(err, domain.ErrInvalidCredentials):
		status, message = http.StatusUnauthorized, "invalid credentials"
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
