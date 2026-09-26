package ratelimiter

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	limit   int
	window  time.Duration
}

type client struct {
	count       int
	windowStart time.Time
}

func New(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*client),
		limit:   limit,
		window:  window,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	entry, ok := r.clients[key]

	if !ok || now.Sub(entry.windowStart) >= r.window {
		r.clients[key] = &client{
			count:       1,
			windowStart: now,
		}

		return true
	}

	if entry.count >= r.limit {
		return false
	}

	entry.count++

	return true
}
