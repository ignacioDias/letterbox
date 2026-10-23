package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetReviewsFromMovieByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name       string
		movieID    string
		query      string
		reviews    []*domains.Review
		reviewsErr error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			movieID:    "10",
			reviews:    wantReviews,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":1,"user_id":42,"movie_id":10,"score":8,"content":"","contains_spoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:       "non-numeric movie id",
			movieID:    "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero movie id",
			movieID:    "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative movie id",
			movieID:    "-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "movie id above int32",
			movieID:    "2147483648",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid pagination",
			movieID:    "10",
			query:      "?limit=0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repository error",
			movieID:    "10",
			reviewsErr: genericErr,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromMovieByIDFunc: func(context.Context, int32, int, int) ([]*domains.Review, error) {
					return tt.reviews, tt.reviewsErr
				},
			}

			handler := NewReviewHandler(reviewRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+tt.movieID+"/reviews"+tt.query, nil)
			req.SetPathValue("movie_id", tt.movieID)
			rec := httptest.NewRecorder()

			handler.GetReviewsFromMovieByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func authenticatedCreateReviewRequest(t *testing.T, handler http.HandlerFunc, userID int64, movieID, body string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies/"+movieID+"/reviews", strings.NewReader(body))
	req.SetPathValue("movie_id", movieID)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestCreateReviewForMovieByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		movieID    string
		body       string
		repoErr    error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			movieID:    "10",
			body:       `{"score":8,"content":"great","contains_spoilers":false}`,
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":1,"user_id":7,"movie_id":10,"score":8,"content":"great","contains_spoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:       "invalid movie id",
			movieID:    "abc",
			body:       `{"score":8}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			movieID:    "10",
			body:       `{"score":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid score",
			movieID:    "10",
			body:       `{"score":0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "duplicate review",
			movieID:    "10",
			body:       `{"score":8}`,
			repoErr:    repositories.ErrReviewAlreadyExists,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "movie not found",
			movieID:    "10",
			body:       `{"score":8}`,
			repoErr:    repositories.ErrMovieNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "repository error",
			movieID:    "10",
			body:       `{"score":8}`,
			repoErr:    genericErr,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReview *domains.Review
			reviewRepo := &testutil.FakeReviewRepository{
				CreateReviewForMovieByIDFunc: func(_ context.Context, review *domains.Review) (*domains.Review, error) {
					gotReview = review
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					review.ID = 1
					return review, nil
				},
			}

			handler := NewReviewHandler(reviewRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
			rec := authenticatedCreateReviewRequest(t, handler.CreateReviewForMovieByID, 7, tt.movieID, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantStatus == http.StatusCreated {
				if gotReview == nil || gotReview.UserID != 7 || gotReview.MovieID != 10 {
					t.Fatalf("service received %v; want user 7 and movie 10", gotReview)
				}
			}
		})
	}
}

func TestCreateReviewForMovieByIDHandlerWithoutAuth(t *testing.T) {
	handler := NewReviewHandler(&testutil.FakeReviewRepository{}, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies/10/reviews", strings.NewReader(`{"score":8}`))
	req.SetPathValue("movie_id", "10")
	rec := httptest.NewRecorder()

	handler.CreateReviewForMovieByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}
