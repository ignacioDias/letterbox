package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestAuthenticate(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name         string
		cookie       *http.Cookie
		session      *domains.Session
		repoErr      error
		wantStatus   int
		wantNextCall bool
		wantUserID   int64
	}{
		{
			name:       "missing cookie",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty cookie",
			cookie:     &http.Cookie{Name: SessionCookieName, Value: ""},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid session",
			cookie:     &http.Cookie{Name: SessionCookieName, Value: "bad"},
			repoErr:    repositories.ErrSessionNotFound,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "repository error",
			cookie:     &http.Cookie{Name: SessionCookieName, Value: "session-1"},
			repoErr:    genericErr,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "nil session",
			cookie:     &http.Cookie{Name: SessionCookieName, Value: "session-1"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:         "valid session",
			cookie:       &http.Cookie{Name: SessionCookieName, Value: "session-1"},
			session:      &domains.Session{ID: "session-1", UserID: 7},
			wantStatus:   http.StatusOK,
			wantNextCall: true,
			wantUserID:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionRepo := &testutil.FakeSessionRepository{
				GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
					if tt.cookie == nil || sessionID != tt.cookie.Value {
						t.Fatalf("GetSessionByID(%q); want %q", sessionID, tt.cookie.Value)
					}
					return tt.session, tt.repoErr
				},
			}
			nextCalled := false
			next := func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				userID, ok := GetUserID(r)
				if !ok || userID != tt.wantUserID {
					t.Fatalf("GetUserID() = %d, %v; want %d, true", userID, ok, tt.wantUserID)
				}
				w.WriteHeader(http.StatusOK)
			}

			mw := NewAuthenticationMiddleware(sessionRepo)
			handler := mw.Authenticate(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)

			if nextCalled != tt.wantNextCall {
				t.Fatalf("next called = %v; want %v", nextCalled, tt.wantNextCall)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestGetUserIDWithoutContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := GetUserID(req); ok {
		t.Fatal("GetUserID returned ok for a request without a user ID")
	}
}
