package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func authenticatedRequest(t *testing.T, handler http.HandlerFunc, userID int64, username string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.SetPathValue("username", username)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestFollowUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		username   string
		userID     int64
		targetID   int64
		targetErr  error
		blocked    bool
		blockErr   error
		followErr  error
		wantStatus int
	}{
		{name: "success", username: "grace", userID: 7, targetID: 42, wantStatus: http.StatusNoContent},
		{name: "cannot follow self", username: "ada", userID: 7, targetID: 7, wantStatus: http.StatusBadRequest},
		{name: "invalid username", username: "a", userID: 7, wantStatus: http.StatusBadRequest},
		{name: "target not found", username: "ghost", userID: 7, targetErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "blocked", username: "grace", userID: 7, targetID: 42, blocked: true, wantStatus: http.StatusForbidden},
		{name: "block check fails", username: "grace", userID: 7, targetID: 42, blockErr: genericErr, wantStatus: http.StatusInternalServerError},
		{name: "already following", username: "grace", userID: 7, targetID: 42, followErr: repositories.ErrAlreadyFollowing, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followRepo := &testutil.FakeFollowRepository{
				FollowUserFunc: func(context.Context, int64, int64) error {
					return tt.followErr
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}
			blockRepo := &testutil.FakeBlockRepository{
				IsBlockedBetweenFunc: func(context.Context, int64, int64) (bool, error) {
					return tt.blocked, tt.blockErr
				},
			}

			handler := NewFollowHandler(followRepo, userRepo, blockRepo)
			rec := authenticatedRequest(t, handler.FollowUserByUsername, tt.userID, tt.username)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestFollowUserHandlerWithoutAuth(t *testing.T) {
	handler := NewFollowHandler(&testutil.FakeFollowRepository{}, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.SetPathValue("username", "grace")

	handler.FollowUserByUsername(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func TestUnfollowUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		username   string
		userID     int64
		targetID   int64
		targetErr  error
		unfollow   error
		wantStatus int
	}{
		{name: "success", username: "grace", userID: 7, targetID: 42, wantStatus: http.StatusNoContent},
		{name: "cannot unfollow self", username: "ada", userID: 7, targetID: 7, wantStatus: http.StatusBadRequest},
		{name: "invalid username", username: "a", userID: 7, wantStatus: http.StatusBadRequest},
		{name: "target not found", username: "ghost", userID: 7, targetErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "not following", username: "grace", userID: 7, targetID: 42, unfollow: repositories.ErrNotFollowing, wantStatus: http.StatusConflict},
		{name: "repository error", username: "grace", userID: 7, targetID: 42, unfollow: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followRepo := &testutil.FakeFollowRepository{
				UnfollowUserFunc: func(context.Context, int64, int64) error {
					return tt.unfollow
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			handler := NewFollowHandler(followRepo, userRepo, &testutil.FakeBlockRepository{})
			rec := authenticatedRequest(t, handler.UnfollowUserByUsername, tt.userID, tt.username)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestGetFollowersHandler(t *testing.T) {
	users := []*domains.PublicUser{{ID: 1, Username: "grace"}}
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		query      string
		username   string
		users      []*domains.PublicUser
		repoErr    error
		wantStatus int
	}{
		{name: "success", username: "ada", users: users, wantStatus: http.StatusOK},
		{name: "invalid username", username: "a", wantStatus: http.StatusBadRequest},
		{name: "invalid limit", query: "?limit=abc", username: "ada", wantStatus: http.StatusBadRequest},
		{name: "limit too high", query: "?limit=101", username: "ada", wantStatus: http.StatusBadRequest},
		{name: "invalid offset", query: "?offset=-1", username: "ada", wantStatus: http.StatusBadRequest},
		{name: "user not found", username: "ghost", repoErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", username: "ada", repoErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followRepo := &testutil.FakeFollowRepository{
				GetFollowersByUsernameFunc: func(context.Context, string, int, int) ([]*domains.PublicUser, error) {
					return tt.users, tt.repoErr
				},
			}
			handler := NewFollowHandler(followRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{})

			req := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			req.SetPathValue("username", tt.username)
			rec := httptest.NewRecorder()
			handler.GetFollowersByUsername(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestGetFollowingHandler(t *testing.T) {
	users := []*domains.PublicUser{{ID: 1, Username: "grace"}}
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		query      string
		username   string
		users      []*domains.PublicUser
		repoErr    error
		wantStatus int
	}{
		{name: "success", username: "ada", users: users, wantStatus: http.StatusOK},
		{name: "invalid username", username: "a", wantStatus: http.StatusBadRequest},
		{name: "invalid limit", query: "?limit=0", username: "ada", wantStatus: http.StatusBadRequest},
		{name: "user not found", username: "ghost", repoErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", username: "ada", repoErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followRepo := &testutil.FakeFollowRepository{
				GetFollowingByUsernameFunc: func(context.Context, string, int, int) ([]*domains.PublicUser, error) {
					return tt.users, tt.repoErr
				},
			}
			handler := NewFollowHandler(followRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{})

			req := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			req.SetPathValue("username", tt.username)
			rec := httptest.NewRecorder()
			handler.GetFollowingByUsername(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
