package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/reelchoice/backend/internal/party"
)

// RedisClient wraps the Redis client with our application logic
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(redisURL string) (*RedisClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// GetParty fetches a party from Redis by ID
func (r *RedisClient) GetParty(ctx context.Context, partyID string) (*party.Party, error) {
	key := fmt.Sprintf("party:%s", partyID)

	jsonData, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Party not found
		}
		return nil, fmt.Errorf("failed to get party from Redis: %w", err)
	}

	var p party.Party
	if err := json.Unmarshal([]byte(jsonData), &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal party data: %w", err)
	}

	return &p, nil
}

// SaveParty saves a party to Redis
func (r *RedisClient) SaveParty(ctx context.Context, p *party.Party) error {
	key := fmt.Sprintf("party:%s", p.ID)

	jsonData, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal party data: %w", err)
	}

	// Set with 24 hour expiration for parties
	if err := r.client.Set(ctx, key, jsonData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save party to Redis: %w", err)
	}

	return nil
}

// DeleteParty removes a party from Redis
func (r *RedisClient) DeleteParty(ctx context.Context, partyID string) error {
	key := fmt.Sprintf("party:%s", partyID)
	return r.client.Del(ctx, key).Err()
}

// GetCachedTMDBData gets cached TMDB search results
func (r *RedisClient) GetCachedTMDBData(ctx context.Context, query string) ([]byte, error) {
	key := fmt.Sprintf("tmdb:search:%s", query)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	return []byte(data), nil
}

// SetCachedTMDBData caches TMDB search results
func (r *RedisClient) SetCachedTMDBData(ctx context.Context, query string, data []byte) error {
	key := fmt.Sprintf("tmdb:search:%s", query)
	// Cache for 1 hour
	return r.client.Set(ctx, key, data, time.Hour).Err()
}

// Token management methods

// SaveAuthToken stores an authentication token in Redis with expiration
func (r *RedisClient) SaveAuthToken(ctx context.Context, token *party.AuthToken) error {
	key := fmt.Sprintf("token:%s", token.Token)

	jsonData, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal auth token: %w", err)
	}

	// Calculate TTL based on token expiration
	ttl := time.Until(token.ExpiresAt)
	if ttl < 0 {
		return fmt.Errorf("token is already expired")
	}

	if err := r.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save auth token to Redis: %w", err)
	}

	return nil
}

// GetAuthToken retrieves an authentication token from Redis
func (r *RedisClient) GetAuthToken(ctx context.Context, tokenStr string) (*party.AuthToken, error) {
	key := fmt.Sprintf("token:%s", tokenStr)

	jsonData, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("failed to get auth token from Redis: %w", err)
	}

	var token party.AuthToken
	if err := json.Unmarshal([]byte(jsonData), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth token: %w", err)
	}

	// Double-check expiration (Redis should auto-expire, but be safe)
	if time.Now().After(token.ExpiresAt) {
		r.client.Del(ctx, key) // Clean up expired token
		return nil, fmt.Errorf("token expired")
	}

	return &token, nil
}

// RevokeAuthToken removes an authentication token from Redis
func (r *RedisClient) RevokeAuthToken(ctx context.Context, tokenStr string) error {
	key := fmt.Sprintf("token:%s", tokenStr)
	return r.client.Del(ctx, key).Err()
}

// Distributed locking methods

// AcquireLock attempts to acquire a distributed lock for a party
func (r *RedisClient) AcquireLock(ctx context.Context, partyID string, lockDuration time.Duration) (bool, error) {
	key := fmt.Sprintf("party:lock:%s", partyID)

	// Use SET with NX (only if not exists) and EX (expiration) options
	result := r.client.SetNX(ctx, key, "locked", lockDuration)
	if result.Err() != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", result.Err())
	}

	return result.Val(), nil
}

// ReleaseLock releases a distributed lock for a party
func (r *RedisClient) ReleaseLock(ctx context.Context, partyID string) error {
	key := fmt.Sprintf("party:lock:%s", partyID)
	return r.client.Del(ctx, key).Err()
}
