// Package ratelimit provides a token bucket rate limiter for throttling
// per-session client intents.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a token bucket rate limiter. Tokens are added at a
// fixed rate up to a maximum burst size. Each call to Allow consumes one
// token; if none are available the call returns false.
type Limiter struct {
	mu        sync.Mutex
	tokens    float64
	maxTokens float64
	rate      float64 // tokens per second
	lastTime  time.Time
}

// New creates a Limiter that allows rate events per second with a maximum
// burst size. The bucket starts full.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		tokens:    float64(burst),
		maxTokens: float64(burst),
		rate:      rate,
		lastTime:  time.Now(),
	}
}

// Allow reports whether an event may happen now. It consumes one token if
// available and returns true, otherwise returns false without consuming.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.lastTime = now

	l.tokens += elapsed * l.rate
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}

	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
