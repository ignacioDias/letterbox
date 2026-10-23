package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestGetReviewsFromUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name          string
		username      string
		userID        int64
		userErr       error
		reviews       []*domains.Review
		reviewsErr    error
		wantErr       error
		wantUsername  string
		wantUserID    int64
		wantLimit     int
		wantOffset    int
		wantReviews   []*domains.Review
		wantLookupRun bool
	}{
		{
			name:          "success",
			username:      "grace",
			userID:        42,
			reviews:       wantReviews,
			wantUsername:  "grace",
			wantUserID:    42,
			wantLimit:     20,
			wantOffset:    5,
			wantReviews:   wantReviews,
			wantLookupRun: true,
		},
		{
			name:          "trims username",
			username:      "  grace  ",
			userID:        42,
			reviews:       wantReviews,
			wantUsername:  "grace",
			wantUserID:    42,
			wantLimit:     20,
			wantOffset:    0,
			wantReviews:   wantReviews,
			wantLookupRun: true,
		},
		{
			name:     "invalid username",
			username: "a",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:          "user not found",
			username:      "ghost",
			userErr:       repositories.ErrUserNotFound,
			wantErr:       repositories.ErrUserNotFound,
			wantUsername:  "ghost",
			wantLookupRun: true,
		},
		{
			name:          "review repository error",
			username:      "grace",
			userID:        42,
			reviewsErr:    genericErr,
			wantErr:       genericErr,
			wantUsername:  "grace",
			wantUserID:    42,
			wantLookupRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUsername string
			var gotUserID int64
			var gotLimit, gotOffset int
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
					if !tt.wantLookupRun {
						t.Fatalf("GetUserByUsername(%q) called; want no lookup", username)
					}
					gotUsername = username
					return &domains.User{ID: tt.userID, Username: username}, tt.userErr
				},
			}
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromUserByUserIDFunc: func(_ context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
					gotUserID, gotLimit, gotOffset = userID, limit, offset
					return tt.reviews, tt.reviewsErr
				},
			}

			svc := NewReviewService(reviewRepo, userRepo, nil)
			reviews, err := svc.GetReviewsFromUserByUsername(context.Background(), tt.username, 20, tt.wantOffset)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetReviewsFromUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetReviewsFromUserByUsername() error = %v; want nil", err)
			}
			if gotUsername != tt.wantUsername {
				t.Fatalf("GetUserByUsername() called with %q; want %q", gotUsername, tt.wantUsername)
			}
			if gotUserID != tt.wantUserID || gotLimit != tt.wantLimit || gotOffset != tt.wantOffset {
				t.Fatalf("GetReviewsFromUserByUserID() called with (%d, %d, %d); want (%d, %d, %d)",
					gotUserID, gotLimit, gotOffset, tt.wantUserID, tt.wantLimit, tt.wantOffset)
			}
			if len(reviews) != len(tt.wantReviews) || reviews[0].ID != tt.wantReviews[0].ID {
				t.Fatalf("GetReviewsFromUserByUsername() = %v; want %v", reviews, tt.wantReviews)
			}
		})
	}
}
