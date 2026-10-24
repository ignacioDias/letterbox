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
			wantBody:    `[{"id":1,"user_id":42,"movie_id":10,"score":8,"content":"","containsSpoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`,
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
			wantBody:   `[{"id":1,"user_id":42,"movie_id":10,"score":8,"content":"","containsSpoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`,
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

func TestGetReviewByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReview := &domains.Review{ID: 10, UserID: 42, MovieID: 3, Score: 8}

	tests := []struct {
		name       string
		reviewID   string
		review     *domains.Review
		reviewErr  error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			reviewID:   "10",
			review:     wantReview,
			wantStatus: http.StatusOK,
			wantBody:   `{"id":10,"user_id":42,"movie_id":3,"score":8,"content":"","containsSpoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:       "non-numeric review id",
			reviewID:   "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero review id",
			reviewID:   "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative review id",
			reviewID:   "-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "review id above int64",
			reviewID:   "9223372036854775808",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "review not found",
			reviewID:   "10",
			reviewErr:  repositories.ErrReviewNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "repository error",
			reviewID:   "10",
			reviewErr:  genericErr,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewByIDFunc: func(context.Context, int64) (*domains.Review, error) {
					return tt.review, tt.reviewErr
				},
			}

			handler := NewReviewHandler(reviewRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/"+tt.reviewID, nil)
			req.SetPathValue("review_id", tt.reviewID)
			rec := httptest.NewRecorder()

			handler.GetReviewByID(rec, req)

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
			body:       `{"score":8,"content":"great","containsSpoilers":false}`,
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":1,"user_id":7,"movie_id":10,"score":8,"content":"great","containsSpoilers":false,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`,
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
			name:       "body too large",
			movieID:    "10",
			body:       `{"content":"` + strings.Repeat("a", reviewBodyLimit) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
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

func authenticatedDeleteReviewRequest(t *testing.T, handler http.HandlerFunc, userID int64, reviewID string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/reviews/"+reviewID, nil)
	req.SetPathValue("review_id", reviewID)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestDeleteReviewByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		reviewID    string
		repoErr     error
		wantStatus  int
		wantRepoRun bool
	}{
		{
			name:        "success",
			reviewID:    "10",
			wantStatus:  http.StatusNoContent,
			wantRepoRun: true,
		},
		{
			name:        "invalid review id",
			reviewID:    "abc",
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "zero review id",
			reviewID:    "0",
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "review not found",
			reviewID:    "10",
			repoErr:     repositories.ErrReviewNotFound,
			wantStatus:  http.StatusNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			reviewID:    "10",
			repoErr:     genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			var gotReviewID int64
			reviewRepo := &testutil.FakeReviewRepository{
				DeleteReviewByIDFunc: func(_ context.Context, userID int64, reviewID int64) error {
					if !tt.wantRepoRun {
						t.Fatalf("DeleteReviewByID() called; want no repository call")
					}
					gotUserID, gotReviewID = userID, reviewID
					return tt.repoErr
				},
			}

			handler := NewReviewHandler(reviewRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
			rec := authenticatedDeleteReviewRequest(t, handler.DeleteReviewByID, 7, tt.reviewID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantRepoRun && (gotUserID != 7 || gotReviewID != 10) {
				t.Fatalf("service received user %d and review %d; want user 7 and review 10", gotUserID, gotReviewID)
			}
		})
	}
}

func TestDeleteReviewByIDHandlerWithoutAuth(t *testing.T) {
	handler := NewReviewHandler(&testutil.FakeReviewRepository{}, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/reviews/10", nil)
	req.SetPathValue("review_id", "10")
	rec := httptest.NewRecorder()

	handler.DeleteReviewByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func authenticatedUpdateReviewRequest(t *testing.T, handler http.HandlerFunc, userID int64, reviewID, body string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/reviews/"+reviewID, strings.NewReader(body))
	req.SetPathValue("review_id", reviewID)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestUpdateReviewByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		reviewID    string
		body        string
		repoErr     error
		wantStatus  int
		wantRepoRun bool
	}{
		{
			name:        "success",
			reviewID:    "10",
			body:        `{"score":9}`,
			wantStatus:  http.StatusOK,
			wantRepoRun: true,
		},
		{
			name:        "no fields",
			reviewID:    "10",
			body:        `{}`,
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "invalid review id",
			reviewID:    "abc",
			body:        `{"score":9}`,
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "invalid body",
			reviewID:    "10",
			body:        `{"score":`,
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "body too large",
			reviewID:    "10",
			body:        `{"content":"` + strings.Repeat("a", reviewBodyLimit) + `"}`,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantRepoRun: false,
		},
		{
			name:        "invalid score",
			reviewID:    "10",
			body:        `{"score":0}`,
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "content too long",
			reviewID:    "10",
			body:        `{"content":"` + strings.Repeat("a", 10001) + `"}`,
			wantStatus:  http.StatusBadRequest,
			wantRepoRun: false,
		},
		{
			name:        "review not found",
			reviewID:    "10",
			body:        `{"score":9}`,
			repoErr:     repositories.ErrReviewNotFound,
			wantStatus:  http.StatusNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			reviewID:    "10",
			body:        `{"score":9}`,
			repoErr:     genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			var gotReviewID int64
			var gotScore *int16
			reviewRepo := &testutil.FakeReviewRepository{
				UpdateReviewByIDFunc: func(_ context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
					if !tt.wantRepoRun {
						t.Fatalf("UpdateReviewByID() called; want no repository call")
					}
					gotUserID, gotReviewID, gotScore = userID, reviewID, score
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return &domains.Review{ID: reviewID, UserID: userID, Score: *score}, nil
				},
			}

			handler := NewReviewHandler(reviewRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
			rec := authenticatedUpdateReviewRequest(t, handler.UpdateReviewByID, 7, tt.reviewID, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantRepoRun {
				if gotUserID != 7 || gotReviewID != 10 {
					t.Fatalf("service received user %d and review %d; want user 7 and review 10", gotUserID, gotReviewID)
				}
				if tt.body == `{"score":9}` && (gotScore == nil || *gotScore != 9) {
					t.Fatalf("service received score %v; want 9", gotScore)
				}
			}
		})
	}
}

func TestUpdateReviewByIDHandlerWithoutAuth(t *testing.T) {
	handler := NewReviewHandler(&testutil.FakeReviewRepository{}, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/reviews/10", strings.NewReader(`{"score":9}`))
	req.SetPathValue("review_id", "10")
	rec := httptest.NewRecorder()

	handler.UpdateReviewByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}
