package services

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	cleanup *time.Ticker
}

type tokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate int // tokens per hour
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter with automatic cleanup
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		cleanup: time.NewTicker(10 * time.Minute),
	}
	
	// Background cleanup of old buckets
	go func() {
		for range rl.cleanup.C {
			rl.cleanupOldBuckets()
		}
	}()
	
	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     limit,
			maxTokens:  limit,
			refillRate: limit,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	
	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed.Hours() * float64(bucket.refillRate))
	
	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > bucket.maxTokens {
			bucket.tokens = bucket.maxTokens
		}
		bucket.lastRefill = now
	}
	
	// Check if request can proceed
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// GetRemaining returns the remaining tokens for a key
func (rl *RateLimiter) GetRemaining(key string, limit int) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		return limit
	}
	
	// Refill calculation without modifying state
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed.Hours() * float64(bucket.refillRate))
	
	remaining := bucket.tokens + tokensToAdd
	if remaining > bucket.maxTokens {
		remaining = bucket.maxTokens
	}
	
	return remaining
}

// ResetAfter returns when the bucket will have at least one token
func (rl *RateLimiter) ResetAfter(key string) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		return 0
	}
	
	if bucket.tokens > 0 {
		return 0
	}
	
	// Calculate time until next token
	secondsUntilToken := 3600.0 / float64(bucket.refillRate)
	return time.Duration(secondsUntilToken * float64(time.Second))
}

// cleanupOldBuckets removes buckets that haven't been used in 1 hour
func (rl *RateLimiter) cleanupOldBuckets() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	threshold := time.Now().Add(-1 * time.Hour)
	for key, bucket := range rl.buckets {
		if bucket.lastRefill.Before(threshold) {
			delete(rl.buckets, key)
		}
	}
}

// Stop stops the cleanup ticker
func (rl *RateLimiter) Stop() {
	rl.cleanup.Stop()
}
