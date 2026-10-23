package handlers

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

func TestGetReviewsFromUserByUsernameHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name       string
		username   string
		query      string
		userID     int64
		userErr    error
		reviews    []*domains.Review
		reviewsErr error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			username:   "grace",
			userID:     42,
			reviews:    wantReviews,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":1,"user_id":42,"movie_id":10,"score":8,"content":"","contains_spoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:       "invalid username",
			username:   "a",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "user not found",
			username:   "ghost",
			userErr:    repositories.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "review repository error",
			username:   "grace",
			userID:     42,
			reviewsErr: genericErr,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid pagination",
			username:   "grace",
			query:      "?limit=0",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
					return &domains.User{ID: tt.userID, Username: username}, tt.userErr
				},
			}
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromUserByUserIDFunc: func(context.Context, int64, int, int) ([]*domains.Review, error) {
					return tt.reviews, tt.reviewsErr
				},
			}

			handler := NewReviewHandler(reviewRepo, userRepo, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+tt.username+"/reviews"+tt.query, nil)
			req.SetPathValue("username", tt.username)
			rec := httptest.NewRecorder()

			handler.GetReviewsFromUserByUsername(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
