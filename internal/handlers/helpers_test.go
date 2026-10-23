package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/middleware"
	"letterbox/internal/testutil"
)

func authenticatedRequestWithBody(t *testing.T, handler http.HandlerFunc, body string, userID int64) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		maxBytes   int64
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"score":8}`,
			maxBytes:   1 << 10,
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed",
			body:       `{`,
			maxBytes:   1 << 10,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "too large",
			body:       `{"score":` + strings.Repeat("1", 1<<10) + `}`,
			maxBytes:   1 << 10,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst ReviewReq
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			if got := decodeJSON(rec, req, &dst, tt.maxBytes); got != tt.wantOK {
				t.Fatalf("decodeJSON() = %v; want %v", got, tt.wantOK)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
