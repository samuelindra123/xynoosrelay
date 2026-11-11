package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// RateLimiter manages in-memory sliding-window request throttling.
type RateLimiter struct {
	mu      sync.Mutex
	history map[string][]time.Time
}

// NewRateLimiter creates a new rate limiter instance.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		history: make(map[string][]time.Time),
	}
}

// Allow checks if the given key has not exceeded maxEvents within window.
func (rl *RateLimiter) Allow(key string, maxEvents int, window time.Duration) (allowed bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	events := rl.history[key]
	var validEvents []time.Time
	for _, t := range events {
		if t.After(cutoff) {
			validEvents = append(validEvents, t)
		}
	}

	if len(validEvents) >= maxEvents {
		oldest := validEvents[0]
		wait := window - now.Sub(oldest)
		if wait < 0 {
			wait = time.Second
		}
		rl.history[key] = validEvents
		return false, wait
	}

	validEvents = append(validEvents, now)
	rl.history[key] = validEvents
	return true, 0
}

// HashKey computes a SHA-256 hex string to avoid storing raw emails or sensitive identifiers in rate limiter maps.
func HashKey(prefix, val string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(val))))
	return prefix + ":" + hex.EncodeToString(h[:])
}
