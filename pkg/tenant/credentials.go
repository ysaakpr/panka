package tenant

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 10
	
	// SecretLength is the length of generated secrets (32 characters)
	SecretLength = 32
)

// GenerateCredentials generates new tenant credentials
func GenerateCredentials(tenantID string) (*TenantCredentials, error) {
	// Generate random secret
	secret, err := generateSecret(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}
	
	// Hash the secret with bcrypt
	hash, err := hashSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to hash secret: %w", err)
	}
	
	return &TenantCredentials{
		TenantID: tenantID,
		Secret:   secret,
		Hash:     hash,
	}, nil
}

// VerifyCredentials verifies a tenant secret against the stored hash
func VerifyCredentials(secret, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
	return err == nil
}

// RotateCredentials generates new credentials for an existing tenant
func RotateCredentials(tenantID string) (*TenantCredentials, error) {
	return GenerateCredentials(tenantID)
}

// generateSecret generates a random secret with tenant-specific prefix
func generateSecret(tenantID string) (string, error) {
	// Generate prefix from tenant ID (first 4 chars or "pnka")
	prefix := generatePrefix(tenantID)
	
	// Generate random bytes
	randomBytes := make([]byte, 24) // 24 bytes = 32 base64 chars
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	
	// Encode to base64 and remove padding
	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)
	
	// Format: prefix_randomchars
	// Example: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5a
	return fmt.Sprintf("%s_%s", prefix, encoded), nil
}

// generatePrefix creates a 4-character prefix from tenant ID
func generatePrefix(tenantID string) string {
	// Extract meaningful characters from tenant ID
	// e.g., "notifications-team" -> "ntfy"
	parts := strings.Split(tenantID, "-")
	
	if len(parts) == 1 {
		// Single word, take first 4 chars
		if len(tenantID) >= 4 {
			return tenantID[:4]
		}
		return padPrefix(tenantID)
	}
	
	// Multiple words, take first char of each (up to 4)
	var prefix strings.Builder
	for i := 0; i < len(parts) && i < 4; i++ {
		if len(parts[i]) > 0 {
			prefix.WriteByte(parts[i][0])
		}
	}
	
	result := prefix.String()
	if len(result) < 4 {
		return padPrefix(result)
	}
	
	return result
}

// padPrefix pads a prefix to 4 characters
func padPrefix(s string) string {
	if len(s) >= 4 {
		return s[:4]
	}
	return s + strings.Repeat("x", 4-len(s))
}

// hashSecret creates a bcrypt hash of the secret
func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ValidateSecretFormat checks if a secret has the correct format
func ValidateSecretFormat(secret string) bool {
	// Format: prefix_32chars
	// Total length should be 4 (prefix) + 1 (underscore) + 32 (random) = 37
	if len(secret) < 37 {
		return false
	}
	
	parts := strings.Split(secret, "_")
	if len(parts) != 2 {
		return false
	}
	
	if len(parts[0]) != 4 {
		return false
	}
	
	if len(parts[1]) != 32 {
		return false
	}
	
	return true
}

