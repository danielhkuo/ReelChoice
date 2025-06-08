package party

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// AuthToken represents a session token for a participant
type AuthToken struct {
	Token     string    `json:"token"`
	PartyID   string    `json:"party_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IsHost    bool      `json:"is_host"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RedisTokenStore interface for Redis operations needed by TokenManager
type RedisTokenStore interface {
	SaveAuthToken(ctx context.Context, token *AuthToken) error
	GetAuthToken(ctx context.Context, tokenStr string) (*AuthToken, error)
	RevokeAuthToken(ctx context.Context, tokenStr string) error
}

// TokenManager handles creation and validation of auth tokens using Redis
type TokenManager struct {
	redis RedisTokenStore
}

// NewTokenManager creates a new token manager with Redis backend
func NewTokenManager(redis RedisTokenStore) *TokenManager {
	return &TokenManager{
		redis: redis,
	}
}

// CreateToken generates a new authentication token for a participant
func (tm *TokenManager) CreateToken(ctx context.Context, partyID, userID, username string, isHost bool) (*AuthToken, error) {
	// Generate secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	token := hex.EncodeToString(tokenBytes)

	authToken := &AuthToken{
		Token:     token,
		PartyID:   partyID,
		UserID:    userID,
		Username:  username,
		IsHost:    isHost,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour expiration
	}

	// Save token to Redis with automatic expiration
	if err := tm.redis.SaveAuthToken(ctx, authToken); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return authToken, nil
}

// ValidateToken validates a token and returns the associated auth info
func (tm *TokenManager) ValidateToken(ctx context.Context, token string) (*AuthToken, error) {
	// Fetch token from Redis
	authToken, err := tm.redis.GetAuthToken(ctx, token)
	if err != nil {
		return nil, err // Error already includes "invalid token" or other specific message
	}

	return authToken, nil
}

// RevokeToken removes a token from Redis
func (tm *TokenManager) RevokeToken(ctx context.Context, token string) error {
	return tm.redis.RevokeAuthToken(ctx, token)
}

// CleanupExpiredTokens is no longer needed as Redis handles expiration automatically
// Keeping this method for backward compatibility, but it's now a no-op
func (tm *TokenManager) CleanupExpiredTokens() {
	// Redis handles token expiration automatically via TTL
	// This method is kept for backward compatibility but does nothing
}
