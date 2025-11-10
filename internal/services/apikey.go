package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// APIKeyService manages API keys for REST API authentication
type APIKeyService struct {
	db *sql.DB
}

// APIKey represents an API key in the system
type APIKey struct {
	ID          int
	KeyHash     string
	DiscordID   string
	Name        string
	Scopes      []string
	RateLimit   int
	LastUsed    *time.Time
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(db *sql.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// GenerateAPIKey generates a new cryptographically secure API key
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, discordID, name string, scopes []string, rateLimit int, ttl *time.Duration) (string, *APIKey, error) {
	// Generate 32-byte random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random key: %v", err)
	}
	
	// Encode as base64 for transmission
	apiKey := fmt.Sprintf("agis_%s", base64.URLEncoding.EncodeToString(keyBytes))
	
	// Hash the key for storage (SHA256)
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := fmt.Sprintf("%x", hash[:])
	
	// Calculate expiration
	var expiresAt *time.Time
	if ttl != nil {
		exp := time.Now().Add(*ttl)
		expiresAt = &exp
	}
	
	// Default scopes if none provided
	if len(scopes) == 0 {
		scopes = []string{"read:servers"}
	}
	
	// Default rate limit (100 req/hour)
	if rateLimit == 0 {
		rateLimit = 100
	}
	
	// Insert into database
	var id int
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO api_keys (key_hash, discord_id, name, scopes, rate_limit, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, keyHash, discordID, name, pq.Array(scopes), rateLimit, expiresAt).Scan(&id, &createdAt)
	
	if err != nil {
		return "", nil, fmt.Errorf("failed to store API key: %v", err)
	}
	
	key := &APIKey{
		ID:        id,
		KeyHash:   keyHash,
		DiscordID: discordID,
		Name:      name,
		Scopes:    scopes,
		RateLimit: rateLimit,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	
	return apiKey, key, nil
}

// ValidateAPIKey validates an API key and returns the associated metadata
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, apiKey string) (*APIKey, error) {
	// Check format
	if !strings.HasPrefix(apiKey, "agis_") {
		return nil, fmt.Errorf("invalid API key format")
	}
	
	// Hash the provided key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := fmt.Sprintf("%x", hash[:])
	
	// Query database
	var key APIKey
	var scopes string
	var lastUsed sql.NullTime
	var expiresAt sql.NullTime
	
	err := s.db.QueryRowContext(ctx, `
		SELECT id, key_hash, discord_id, name, scopes, rate_limit, last_used, created_at, expires_at
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(
		&key.ID,
		&key.KeyHash,
		&key.DiscordID,
		&key.Name,
		&scopes,
		&key.RateLimit,
		&lastUsed,
		&key.CreatedAt,
		&expiresAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}
	
	// Parse scopes (stored as PostgreSQL array)
	key.Scopes = strings.Split(strings.Trim(scopes, "{}"), ",")
	
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
		// Check if expired
		if time.Now().After(*key.ExpiresAt) {
			return nil, fmt.Errorf("API key expired")
		}
	}
	
	// Update last used timestamp
	go s.updateLastUsed(key.ID)
	
	return &key, nil
}

// updateLastUsed updates the last_used timestamp asynchronously
func (s *APIKeyService) updateLastUsed(keyID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, _ = s.db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used = NOW()
		WHERE id = $1
	`, keyID)
}

// ListAPIKeys lists all API keys for a user
func (s *APIKeyService) ListAPIKeys(ctx context.Context, discordID string) ([]*APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key_hash, discord_id, name, scopes, rate_limit, last_used, created_at, expires_at
		FROM api_keys
		WHERE discord_id = $1
		ORDER BY created_at DESC
	`, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %v", err)
	}
	defer rows.Close()
	
	keys := make([]*APIKey, 0)
	for rows.Next() {
		var key APIKey
		var scopes string
		var lastUsed sql.NullTime
		var expiresAt sql.NullTime
		
		err := rows.Scan(
			&key.ID,
			&key.KeyHash,
			&key.DiscordID,
			&key.Name,
			&scopes,
			&key.RateLimit,
			&lastUsed,
			&key.CreatedAt,
			&expiresAt,
		)
		if err != nil {
			continue
		}
		
		key.Scopes = strings.Split(strings.Trim(scopes, "{}"), ",")
		if lastUsed.Valid {
			key.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		
		keys = append(keys, &key)
	}
	
	return keys, nil
}

// RevokeAPIKey revokes (deletes) an API key
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, keyID int, discordID string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM api_keys
		WHERE id = $1 AND discord_id = $2
	`, keyID, discordID)
	
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %v", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("API key not found or unauthorized")
	}
	
	return nil
}

// HasScope checks if an API key has a specific scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}
