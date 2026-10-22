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
