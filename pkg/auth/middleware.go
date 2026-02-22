package auth

import (
	"ads-mrkt/pkg/auth/role"
	"context"
	"net/http"
	"slices"
	"strings"
)

type TelegramIDContextKey struct{}
type RoleContextKey struct{}

var (
	// TelegramIDKey is the context key for storing the Telegram user ID
	telegramIDKey = TelegramIDContextKey{}
	// RoleKey is the context key for storing the user role
	roleKey = RoleContextKey{}
)

// AuthMiddleware handles JWT authentication for HTTP requests
type AuthMiddleware struct {
	jwtManager *JWTManager
}

// NewAuthMiddleware creates a new instance of AuthMiddleware
func NewAuthMiddleware(jwtManager *JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// WithAuth Middleware function to handle JWT authentication
func (m *AuthMiddleware) WithAuth(next http.HandlerFunc, allowedRoles ...role.Role) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Extract and validate token
		tokenStr := parts[1]
		telegramID, err := m.jwtManager.ExtractTelegramID(tokenStr)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		role, err := m.jwtManager.ExtractRole(tokenStr)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		if len(allowedRoles) > 0 && !slices.Contains(allowedRoles, role) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add telegram ID to request context
		ctx := context.WithValue(r.Context(), telegramIDKey, telegramID)
		ctx = context.WithValue(ctx, roleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTelegramID extracts the Telegram ID from the context
func GetTelegramID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(telegramIDKey).(int64)
	return id, ok
}

// GetRole extracts the user role from the context
func GetRole(ctx context.Context) (role.Role, bool) {
	role, ok := ctx.Value(roleKey).(role.Role)
	return role, ok
}
