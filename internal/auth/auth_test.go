package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	manager, err := NewJWTManager("", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	pair, err := manager.GenerateTokenPair(1, "admin")
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("Access token is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}
	if pair.ExpiresAt == 0 {
		t.Error("ExpiresAt is zero")
	}

	// Validate access token
	claims, err := manager.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.AdminID != 1 {
		t.Errorf("Expected admin_id 1, got %d", claims.AdminID)
	}
	if claims.Username != "admin" {
		t.Errorf("Expected username admin, got %s", claims.Username)
	}
}

func TestInvalidToken(t *testing.T) {
	manager, err := NewJWTManager("", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	_, err = manager.ValidateAccessToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestExpiredToken(t *testing.T) {
	manager, err := NewJWTManager("", -1*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	pair, err := manager.GenerateTokenPair(1, "admin")
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	_, err = manager.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestDifferentSecrets(t *testing.T) {
	manager1, _ := NewJWTManager("", 15*time.Minute, 7*24*time.Hour)
	manager2, _ := NewJWTManager("", 15*time.Minute, 7*24*time.Hour)

	pair, _ := manager1.GenerateTokenPair(1, "admin")
	_, err := manager2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Error("Expected error when validating with different secret")
	}
}

func TestHashToken(t *testing.T) {
	hash1 := HashToken("test-token")
	hash2 := HashToken("test-token")
	hash3 := HashToken("different-token")

	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
	if len(hash1) != 64 {
		t.Errorf("Hash should be 64 chars (SHA-256 hex), got %d", len(hash1))
	}
}
