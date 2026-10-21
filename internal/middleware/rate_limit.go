package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateLimitClient struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*rateLimitClient
	requests    int
	window      time.Duration
	lastCleanup time.Time
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	if requests <= 0 {
		panic("rate limiter: requests must be positive")
	}
	if window <= 0 {
		panic("rate limiter: window must be positive")
	}

	return &RateLimiter{
		clients:  make(map[string]*rateLimitClient),
		requests: requests,
		window:   window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := rl.allow(clientIP(r), time.Now())
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string, now time.Time) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.Sub(rl.lastCleanup) >= rl.window {
		for key, client := range rl.clients {
			if !now.Before(client.resetAt) {
				delete(rl.clients, key)
			}
		}
		rl.lastCleanup = now
	}

	client, ok := rl.clients[ip]
	if !ok || !now.Before(client.resetAt) {
		client = &rateLimitClient{
			resetAt: now.Add(rl.window),
		}
		rl.clients[ip] = client
	}

	if client.count >= rl.requests {
		return false, client.resetAt.Sub(now)
	}

	client.count++
	return true, 0
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
