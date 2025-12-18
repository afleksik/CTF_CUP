package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*limiterEntry
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*limiterEntry),
	}
}

func (rl *RateLimiter) GetLimiter(vsID, userID string, requestsPerWindow, windowSec int) *rate.Limiter {
	key := fmt.Sprintf("%s:%s", vsID, userID)

	rl.mu.RLock()
	entry, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		rl.mu.Lock()
		entry.lastUsed = time.Now()
		rl.mu.Unlock()
		return entry.limiter
	}

	rps := float64(requestsPerWindow) / float64(windowSec)
	burst := requestsPerWindow

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if entry, exists := rl.limiters[key]; exists {
		entry.lastUsed = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	rl.limiters[key] = &limiterEntry{
		limiter:  limiter,
		lastUsed: time.Now(),
	}

	return limiter
}

func (rl *RateLimiter) Allow(vsID, userID string, requestsPerWindow, windowSec int) bool {
	limiter := rl.GetLimiter(vsID, userID, requestsPerWindow, windowSec)
	return limiter.Allow()
}

func (rl *RateLimiter) Cleanup(maxIdleTime time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, entry := range rl.limiters {
		if now.Sub(entry.lastUsed) > maxIdleTime {
			delete(rl.limiters, key)
		}
	}
}
