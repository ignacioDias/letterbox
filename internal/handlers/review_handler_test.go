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

func authenticatedReviewRequest(t *testing.T, handler http.HandlerFunc, userID int64, username, query string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+username+"/reviews"+query, nil)
	req.SetPathValue("username", username)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestGetReviewsFromUserByUsernameHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name        string
		username    string
		query       string
		requesterID int64
		userID      int64
		userErr     error
		blocked     bool
		blockErr    error
		reviews     []*domains.Review
		reviewsErr  error
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "success",
			username:    "grace",
			requesterID: 7,
			userID:      42,
			reviews:     wantReviews,
			wantStatus:  http.StatusOK,
			wantBody:    `[{"id":1,"user_id":42,"movie_id":10,"score":8,"content":"","contains_spoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:        "invalid username",
			username:    "a",
			requesterID: 7,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "user not found",
			username:    "ghost",
			requesterID: 7,
			userErr:     repositories.ErrUserNotFound,
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "blocked",
			username:    "grace",
			requesterID: 7,
			userID:      42,
			blocked:     true,
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "block repository error",
			username:    "grace",
			requesterID: 7,
			userID:      42,
			blockErr:    genericErr,
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "review repository error",
			username:    "grace",
			requesterID: 7,
			userID:      42,
			reviewsErr:  genericErr,
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "invalid pagination",
			username:    "grace",
			requesterID: 7,
			query:       "?limit=0",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
					return &domains.User{ID: tt.userID, Username: username}, tt.userErr
				},
			}
			blockRepo := &testutil.FakeBlockRepository{
				IsBlockedBetweenFunc: func(context.Context, int64, int64) (bool, error) {
					return tt.blocked, tt.blockErr
				},
			}
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromUserByUserIDFunc: func(context.Context, int64, int, int) ([]*domains.Review, error) {
					return tt.reviews, tt.reviewsErr
				},
			}

			handler := NewReviewHandler(reviewRepo, userRepo, blockRepo, nil)
			rec := authenticatedReviewRequest(t, handler.GetReviewsFromUserByUsername, tt.requesterID, tt.username, tt.query)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestGetReviewsFromUserByUsernameHandlerWithoutAuth(t *testing.T) {
	handler := NewReviewHandler(&testutil.FakeReviewRepository{}, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("username", "grace")
	rec := httptest.NewRecorder()

	handler.GetReviewsFromUserByUsername(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}
