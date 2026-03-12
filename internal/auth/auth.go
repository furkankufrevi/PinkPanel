// Package auth handles JWT token generation, validation, and refresh.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AdminID  int64  `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager creates a new JWT manager. If secretFile is empty or missing,
// a random secret is generated (suitable for dev; tokens won't survive restart).
func NewJWTManager(secretFile string, accessTTL, refreshTTL time.Duration) (*JWTManager, error) {
	secret, err := loadOrGenerateSecret(secretFile)
	if err != nil {
		return nil, err
	}

	return &JWTManager{
		secret:          secret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}, nil
}

// GenerateTokenPair creates an access token + refresh token for the given admin.
func (m *JWTManager) GenerateTokenPair(adminID int64, username, role string) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenTTL)

	// Default role for backward compatibility with old tokens
	if role == "" {
		role = "super_admin"
	}

	claims := Claims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "pinkpanel",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshTokenTTL returns the configured refresh token duration.
func (m *JWTManager) RefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}

// HashToken returns a SHA-256 hash of a token string (for DB storage).
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func loadOrGenerateSecret(secretFile string) ([]byte, error) {
	if secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err == nil {
			secret := strings.TrimSpace(string(data))
			if len(secret) >= 32 {
				return []byte(secret), nil
			}
		}
		// File doesn't exist or too short — generate and write
		secret, err := generateRandomToken(32)
		if err != nil {
			return nil, fmt.Errorf("generating secret: %w", err)
		}
		if writeErr := os.WriteFile(secretFile, []byte(secret), 0600); writeErr != nil {
			// Can't write? Use the generated secret anyway (won't persist)
			return []byte(secret), nil
		}
		return []byte(secret), nil
	}

	// No file specified — generate ephemeral secret
	secret, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generating ephemeral secret: %w", err)
	}
	return []byte(secret), nil
}

func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
