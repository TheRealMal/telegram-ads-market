package auth

import (
	"ads-mrkt/pkg/auth/role"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims structure
type Claims struct {
	TelegramID int64     `json:"telegram_id"`
	Role       role.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager creates a new instance of JWTManager
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// GenerateToken creates a new JWT token for a given Telegram user ID
func (m *JWTManager) GenerateToken(telegramID int64, role role.Role) (string, error) {
	claims := Claims{
		TelegramID: telegramID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken verifies and parses the JWT token
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected token signing method: %v", token.Header["alg"])
			}
			return m.secretKey, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ExtractTelegramID extracts Telegram ID from token string
func (m *JWTManager) ExtractTelegramID(tokenStr string) (int64, error) {
	claims, err := m.ValidateToken(tokenStr)
	if err != nil {
		return 0, err
	}
	if m.IsExpired(claims) {
		return 0, fmt.Errorf("session_expired")
	}
	return claims.TelegramID, nil
}

// ExtractRole extracts Role from token string
func (m *JWTManager) ExtractRole(tokenStr string) (role.Role, error) {
	claims, err := m.ValidateToken(tokenStr)
	if err != nil {
		return role.EmptyRole, err
	}
	return claims.Role, nil
}

// IsExpired checks if the token is expired
func (m *JWTManager) IsExpired(claims *Claims) bool {
	return claims.ExpiresAt.Before(time.Now())
}
