package auth

import (
	"context"
	"net/http"
	"strings"
)

type TelegramIDContextKey struct{}

var (
	// TelegramIDKey is the context key for storing the Telegram user ID
	telegramIDKey = TelegramIDContextKey{}
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
func (m *AuthMiddleware) WithAuth(next http.HandlerFunc) http.HandlerFunc {
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

		// Add telegram ID to request context
		ctx := context.WithValue(r.Context(), telegramIDKey, telegramID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTelegramID extracts the Telegram ID from the context
func GetTelegramID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(telegramIDKey).(int64)
	return id, ok
}
