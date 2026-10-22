package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	now := time.Now()

	if allowed, _ := rl.allow("1.2.3.4", now); !allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed, _ := rl.allow("1.2.3.4", now); !allowed {
		t.Fatal("second request should be allowed")
	}
	allowed, retryAfter := rl.allow("1.2.3.4", now)
	if allowed {
		t.Fatal("third request should be denied")
	}
	if retryAfter != time.Minute {
		t.Fatalf("retryAfter = %v; want 1m", retryAfter)
	}
	if allowed, _ := rl.allow("5.6.7.8", now); !allowed {
		t.Fatal("a different client should be allowed")
	}
	if allowed, _ := rl.allow("1.2.3.4", now.Add(time.Minute)); !allowed {
		t.Fatal("request after the window should be allowed")
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	nextCalls := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalls++
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRequest(http.MethodGet, "/", nil)
	first.RemoteAddr = "1.2.3.4:1234"
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("first request status = %d; want 200", firstRec.Code)
	}
	if nextCalls != 1 {
		t.Fatalf("next called %d times; want 1", nextCalls)
	}

	second := httptest.NewRequest(http.MethodGet, "/", nil)
	second.RemoteAddr = "1.2.3.4:1234"
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)

	if secondRec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d; want 429", secondRec.Code)
	}
	if nextCalls != 1 {
		t.Fatalf("next called %d times; want 1", nextCalls)
	}
	if retryAfter := secondRec.Header().Get("Retry-After"); retryAfter != "60" {
		t.Fatalf("Retry-After = %q; want 60", retryAfter)
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{name: "host and port", remoteAddr: "1.2.3.4:5678", want: "1.2.3.4"},
		{name: "host only", remoteAddr: "1.2.3.4", want: "1.2.3.4"},
		{name: "ipv6", remoteAddr: "[::1]:5678", want: "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if got := clientIP(req); got != tt.want {
				t.Fatalf("clientIP(%q) = %q; want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

func TestNewRateLimiterRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		requests int
		window   time.Duration
	}{
		{name: "zero requests", requests: 0, window: time.Minute},
		{name: "negative requests", requests: -1, window: time.Minute},
		{name: "zero window", requests: 1, window: 0},
		{name: "negative window", requests: 1, window: -time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("NewRateLimiter did not panic")
				}
			}()
			NewRateLimiter(tt.requests, tt.window)
		})
	}
}
