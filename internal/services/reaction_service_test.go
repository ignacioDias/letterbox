package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestCreateReactionForReviewByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReaction := &domains.Reaction{TargetID: 10, UserID: 7, Value: 1}

	tests := []struct {
		name         string
		reviewID     int64
		userID       int64
		value        int16
		repoReaction *domains.Reaction
		repoErr      error
		wantErr      error
		wantRepoRun  bool
	}{
		{
			name:         "like",
			reviewID:     10,
			userID:       7,
			value:        1,
			repoReaction: wantReaction,
			wantRepoRun:  true,
		},
		{
			name:         "dislike",
			reviewID:     10,
			userID:       7,
			value:        -1,
			repoReaction: &domains.Reaction{TargetID: 10, UserID: 7, Value: -1},
			wantRepoRun:  true,
		},
		{
			name:     "invalid value",
			reviewID: 10,
			userID:   7,
			value:    0,
			wantErr:  ErrInvalidReactionValue,
		},
		{
			name:        "review not found",
			reviewID:    10,
			userID:      7,
			value:       1,
			repoErr:     repositories.ErrReviewNotFound,
			wantErr:     repositories.ErrReviewNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			reviewID:    10,
			userID:      7,
			value:       1,
			repoErr:     genericErr,
			wantErr:     genericErr,
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
					return tt.repoReaction, tt.repoErr
				},
			}

			svc := NewReactionService(reactionRepo)
			reaction, err := svc.CreateReactionForReviewByID(context.Background(), tt.reviewID, tt.userID, tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateReactionForReviewByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateReactionForReviewByID() error = %v; want nil", err)
			}
			if gotReaction == nil {
				t.Fatalf("CreateReactionForReview() not called; want call")
			}
			if gotReaction.TargetID != tt.reviewID || gotReaction.UserID != tt.userID || gotReaction.Value != tt.value {
				t.Fatalf("CreateReactionForReview() received %v; want target %d, user %d, value %d",
					gotReaction, tt.reviewID, tt.userID, tt.value)
			}
			if reaction != tt.repoReaction {
				t.Fatalf("CreateReactionForReviewByID() = %v; want %v", reaction, tt.repoReaction)
			}
		})
	}
}

func TestRemoveReactionFromReviewByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		reviewID    int64
		userID      int64
		repoErr     error
		wantErr     error
		wantRepoRun bool
	}{
		{
			name:        "success",
			reviewID:    10,
			userID:      7,
			wantRepoRun: true,
		},
		{
			name:        "review not found",
			reviewID:    10,
			userID:      7,
			repoErr:     repositories.ErrReviewNotFound,
			wantErr:     repositories.ErrReviewNotFound,
			wantRepoRun: true,
		},
		{
			name:        "reaction not found",
			reviewID:    10,
			userID:      7,
			repoErr:     repositories.ErrReactionNotFound,
			wantErr:     repositories.ErrReactionNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			reviewID:    10,
			userID:      7,
			repoErr:     genericErr,
			wantErr:     genericErr,
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

			svc := NewReactionService(reactionRepo)
			err := svc.RemoveReactionFromReviewByID(context.Background(), tt.reviewID, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RemoveReactionFromReviewByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RemoveReactionFromReviewByID() error = %v; want nil", err)
			}
			if gotReviewID != tt.reviewID || gotUserID != tt.userID {
				t.Fatalf("RemoveReactionFromReview() received (%d, %d); want (%d, %d)",
					gotReviewID, gotUserID, tt.reviewID, tt.userID)
			}
		})
	}
}

func TestCreateReactionForCommentByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReaction := &domains.Reaction{TargetID: 20, UserID: 7, Value: 1}

	tests := []struct {
		name         string
		commentID    int64
		userID       int64
		value        int16
		repoReaction *domains.Reaction
		repoErr      error
		wantErr      error
		wantRepoRun  bool
	}{
		{
			name:         "like",
			commentID:    20,
			userID:       7,
			value:        1,
			repoReaction: wantReaction,
			wantRepoRun:  true,
		},
		{
			name:         "dislike",
			commentID:    20,
			userID:       7,
			value:        -1,
			repoReaction: &domains.Reaction{TargetID: 20, UserID: 7, Value: -1},
			wantRepoRun:  true,
		},
		{
			name:      "invalid value",
			commentID: 20,
			userID:    7,
			value:     0,
			wantErr:   ErrInvalidReactionValue,
		},
		{
			name:        "comment not found",
			commentID:   20,
			userID:      7,
			value:       1,
			repoErr:     repositories.ErrCommentNotFound,
			wantErr:     repositories.ErrCommentNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			commentID:   20,
			userID:      7,
			value:       1,
			repoErr:     genericErr,
			wantErr:     genericErr,
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
					return tt.repoReaction, tt.repoErr
				},
			}

			svc := NewReactionService(reactionRepo)
			reaction, err := svc.CreateReactionForCommentByID(context.Background(), tt.commentID, tt.userID, tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateReactionForCommentByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateReactionForCommentByID() error = %v; want nil", err)
			}
			if gotReaction == nil {
				t.Fatalf("CreateReactionForComment() not called; want call")
			}
			if gotReaction.TargetID != tt.commentID || gotReaction.UserID != tt.userID || gotReaction.Value != tt.value {
				t.Fatalf("CreateReactionForComment() received %v; want target %d, user %d, value %d",
					gotReaction, tt.commentID, tt.userID, tt.value)
			}
			if reaction != tt.repoReaction {
				t.Fatalf("CreateReactionForCommentByID() = %v; want %v", reaction, tt.repoReaction)
			}
		})
	}
}

func TestRemoveReactionFromCommentByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		commentID   int64
		userID      int64
		repoErr     error
		wantErr     error
		wantRepoRun bool
	}{
		{
			name:        "success",
			commentID:   20,
			userID:      7,
			wantRepoRun: true,
		},
		{
			name:        "comment not found",
			commentID:   20,
			userID:      7,
			repoErr:     repositories.ErrCommentNotFound,
			wantErr:     repositories.ErrCommentNotFound,
			wantRepoRun: true,
		},
		{
			name:        "reaction not found",
			commentID:   20,
			userID:      7,
			repoErr:     repositories.ErrReactionNotFound,
			wantErr:     repositories.ErrReactionNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			commentID:   20,
			userID:      7,
			repoErr:     genericErr,
			wantErr:     genericErr,
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

			svc := NewReactionService(reactionRepo)
			err := svc.RemoveReactionFromCommentByID(context.Background(), tt.commentID, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RemoveReactionFromCommentByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RemoveReactionFromCommentByID() error = %v; want nil", err)
			}
			if gotCommentID != tt.commentID || gotUserID != tt.userID {
				t.Fatalf("RemoveReactionFromComment() received (%d, %d); want (%d, %d)",
					gotCommentID, gotUserID, tt.commentID, tt.userID)
			}
		})
	}
}
