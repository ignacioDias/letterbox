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

func authenticatedReactionRequest(t *testing.T, handler http.HandlerFunc, userID int64, method, path, paramName, paramValue string) *httptest.ResponseRecorder {
	t.Helper()

	sessionRepo := &testutil.FakeSessionRepository{
		GetSessionByIDFunc: func(_ context.Context, sessionID string) (*domains.Session, error) {
			return &domains.Session{ID: sessionID, UserID: userID}, nil
		},
	}
	mw := middleware.NewAuthenticationMiddleware(sessionRepo)
	h := mw.Authenticate(handler)

	req := httptest.NewRequest(method, path, nil)
	req.SetPathValue(paramName, paramValue)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestCreateReactionHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		like        bool
		reviewID    string
		repoErr     error
		wantStatus  int
		wantValue   int16
		wantBody    string
		wantRepoRun bool
	}{
		{
			name:        "like",
			like:        true,
			reviewID:    "10",
			wantStatus:  http.StatusOK,
			wantValue:   1,
			wantBody:    `{"target_id":10,"user_id":7,"value":1,"created_at":"0001-01-01T00:00:00Z"}`,
			wantRepoRun: true,
		},
		{
			name:        "dislike",
			reviewID:    "10",
			wantStatus:  http.StatusOK,
			wantValue:   -1,
			wantBody:    `{"target_id":10,"user_id":7,"value":-1,"created_at":"0001-01-01T00:00:00Z"}`,
			wantRepoRun: true,
		},
		{
			name:       "invalid review id",
			like:       true,
			reviewID:   "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero review id",
			like:       true,
			reviewID:   "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "review not found",
			like:        true,
			reviewID:    "10",
			repoErr:     repositories.ErrReviewNotFound,
			wantStatus:  http.StatusNotFound,
			wantValue:   1,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			like:        true,
			reviewID:    "10",
			repoErr:     genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantValue:   1,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReaction *domains.Reaction
			reactionRepo := &testutil.FakeReactionRepository{
				CreateReactionForReviewFunc: func(_ context.Context, reaction *domains.Reaction) (*domains.Reaction, error) {
					if !tt.wantRepoRun {
						t.Fatalf("CreateReactionForReview() called; want no repository call")
					}
					gotReaction = reaction
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return reaction, nil
				},
			}

			handler := NewReactionHandler(reactionRepo)
			reactionHandler := handler.LikeReviewByID
			path := "/api/v1/reviews/" + tt.reviewID + "/like"
			if !tt.like {
				reactionHandler = handler.DislikeReviewByID
				path = "/api/v1/reviews/" + tt.reviewID + "/dislike"
			}
			rec := authenticatedReactionRequest(t, reactionHandler, 7, http.MethodPost, path, "review_id", tt.reviewID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantRepoRun {
				if gotReaction == nil || gotReaction.TargetID != 10 || gotReaction.UserID != 7 || gotReaction.Value != tt.wantValue {
					t.Fatalf("service received %v; want target 10, user 7, value %d", gotReaction, tt.wantValue)
				}
			}
		})
	}
}

func TestCreateReactionHandlerWithoutAuth(t *testing.T) {
	handler := NewReactionHandler(&testutil.FakeReactionRepository{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/10/like", nil)
	req.SetPathValue("review_id", "10")
	rec := httptest.NewRecorder()

	handler.LikeReviewByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func TestRemoveReactionHandler(t *testing.T) {
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
			name:       "invalid review id",
			reviewID:   "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero review id",
			reviewID:   "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "review not found",
			reviewID:    "10",
			repoErr:     repositories.ErrReviewNotFound,
			wantStatus:  http.StatusNotFound,
			wantRepoRun: true,
		},
		{
			name:        "reaction not found",
			reviewID:    "10",
			repoErr:     repositories.ErrReactionNotFound,
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
			var gotReviewID, gotUserID int64
			reactionRepo := &testutil.FakeReactionRepository{
				RemoveReactionFromReviewFunc: func(_ context.Context, reviewID, userID int64) error {
					if !tt.wantRepoRun {
						t.Fatalf("RemoveReactionFromReview() called; want no repository call")
					}
					gotReviewID, gotUserID = reviewID, userID
					return tt.repoErr
				},
			}

			handler := NewReactionHandler(reactionRepo)
			rec := authenticatedReactionRequest(t, handler.RemoveReactionFromReviewByID, 7, http.MethodDelete, "/api/v1/reviews/"+tt.reviewID+"/reaction", "review_id", tt.reviewID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantRepoRun && (gotReviewID != 10 || gotUserID != 7) {
				t.Fatalf("service received review %d and user %d; want review 10 and user 7", gotReviewID, gotUserID)
			}
		})
	}
}

func TestRemoveReactionHandlerWithoutAuth(t *testing.T) {
	handler := NewReactionHandler(&testutil.FakeReactionRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/reviews/10/reaction", nil)
	req.SetPathValue("review_id", "10")
	rec := httptest.NewRecorder()

	handler.RemoveReactionFromReviewByID(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func TestCreateReactionForCommentHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		like        bool
		commentID   string
		repoErr     error
		wantStatus  int
		wantValue   int16
		wantBody    string
		wantRepoRun bool
	}{
		{
			name:        "like",
			like:        true,
			commentID:   "20",
			wantStatus:  http.StatusOK,
			wantValue:   1,
			wantBody:    `{"target_id":20,"user_id":7,"value":1,"created_at":"0001-01-01T00:00:00Z"}`,
			wantRepoRun: true,
		},
		{
			name:        "dislike",
			commentID:   "20",
			wantStatus:  http.StatusOK,
			wantValue:   -1,
			wantBody:    `{"target_id":20,"user_id":7,"value":-1,"created_at":"0001-01-01T00:00:00Z"}`,
			wantRepoRun: true,
		},
		{
			name:       "invalid comment id",
			like:       true,
			commentID:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero comment id",
			like:       true,
			commentID:  "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "comment not found",
			like:        true,
			commentID:   "20",
			repoErr:     repositories.ErrCommentNotFound,
			wantStatus:  http.StatusNotFound,
			wantValue:   1,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			like:        true,
			commentID:   "20",
			repoErr:     genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantValue:   1,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReaction *domains.Reaction
			reactionRepo := &testutil.FakeReactionRepository{
				CreateReactionForCommentFunc: func(_ context.Context, reaction *domains.Reaction) (*domains.Reaction, error) {
					if !tt.wantRepoRun {
						t.Fatalf("CreateReactionForComment() called; want no repository call")
					}
					gotReaction = reaction
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return reaction, nil
				},
			}

			handler := NewReactionHandler(reactionRepo)
			reactionHandler := handler.LikeCommentByID
			path := "/api/v1/comments/" + tt.commentID + "/like"
			if !tt.like {
				reactionHandler = handler.DislikeCommentByID
				path = "/api/v1/comments/" + tt.commentID + "/dislike"
			}
			rec := authenticatedReactionRequest(t, reactionHandler, 7, http.MethodPost, path, "comment_id", tt.commentID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody+"\n" {
				t.Fatalf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantRepoRun {
				if gotReaction == nil || gotReaction.TargetID != 20 || gotReaction.UserID != 7 || gotReaction.Value != tt.wantValue {
					t.Fatalf("service received %v; want target 20, user 7, value %d", gotReaction, tt.wantValue)
				}
			}
		})
	}
}

func TestRemoveReactionFromCommentHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		commentID   string
		repoErr     error
		wantStatus  int
		wantRepoRun bool
	}{
		{
			name:        "success",
			commentID:   "20",
			wantStatus:  http.StatusNoContent,
			wantRepoRun: true,
		},
		{
			name:       "invalid comment id",
			commentID:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero comment id",
			commentID:  "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "comment not found",
			commentID:   "20",
			repoErr:     repositories.ErrCommentNotFound,
			wantStatus:  http.StatusNotFound,
			wantRepoRun: true,
		},
		{
			name:        "reaction not found",
			commentID:   "20",
			repoErr:     repositories.ErrReactionNotFound,
			wantStatus:  http.StatusNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			commentID:   "20",
			repoErr:     genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCommentID, gotUserID int64
			reactionRepo := &testutil.FakeReactionRepository{
				RemoveReactionFromCommentFunc: func(_ context.Context, commentID, userID int64) error {
					if !tt.wantRepoRun {
						t.Fatalf("RemoveReactionFromComment() called; want no repository call")
					}
					gotCommentID, gotUserID = commentID, userID
					return tt.repoErr
				},
			}

			handler := NewReactionHandler(reactionRepo)
			rec := authenticatedReactionRequest(t, handler.RemoveReactionFromCommentByID, 7, http.MethodDelete, "/api/v1/comments/"+tt.commentID+"/reaction", "comment_id", tt.commentID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantRepoRun && (gotCommentID != 20 || gotUserID != 7) {
				t.Fatalf("service received comment %d and user %d; want comment 20 and user 7", gotCommentID, gotUserID)
			}
		})
	}
}

func TestCommentReactionHandlersWithoutAuth(t *testing.T) {
	handler := NewReactionHandler(&testutil.FakeReactionRepository{})

	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
	}{
		{name: "like", handler: handler.LikeCommentByID, method: http.MethodPost},
		{name: "dislike", handler: handler.DislikeCommentByID, method: http.MethodPost},
		{name: "remove", handler: handler.RemoveReactionFromCommentByID, method: http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/comments/20/reaction", nil)
			req.SetPathValue("comment_id", "20")
			rec := httptest.NewRecorder()

			tt.handler(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", rec.Code)
			}
		})
	}
}
