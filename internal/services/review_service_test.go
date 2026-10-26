package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestGetReviewsFromUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name           string
		username       string
		requesterID    int64
		userID         int64
		userErr        error
		blocked        bool
		blockErr       error
		reviews        []*domains.Review
		reviewsErr     error
		wantErr        error
		wantUsername   string
		wantUserID     int64
		wantLimit      int
		wantOffset     int
		wantReviews    []*domains.Review
		wantLookupRun  bool
		wantBlockCheck bool
	}{
		{
			name:           "success",
			username:       "grace",
			requesterID:    7,
			userID:         42,
			reviews:        wantReviews,
			wantUsername:   "grace",
			wantUserID:     42,
			wantLimit:      20,
			wantOffset:     5,
			wantReviews:    wantReviews,
			wantLookupRun:  true,
			wantBlockCheck: true,
		},
		{
			name:           "trims username",
			username:       "  grace  ",
			requesterID:    7,
			userID:         42,
			reviews:        wantReviews,
			wantUsername:   "grace",
			wantUserID:     42,
			wantLimit:      20,
			wantOffset:     0,
			wantReviews:    wantReviews,
			wantLookupRun:  true,
			wantBlockCheck: true,
		},
		{
			name:     "invalid username",
			username: "a",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:          "user not found",
			username:      "ghost",
			requesterID:   7,
			userErr:       repositories.ErrUserNotFound,
			wantErr:       repositories.ErrUserNotFound,
			wantUsername:  "ghost",
			wantLookupRun: true,
		},
		{
			name:           "blocked",
			username:       "grace",
			requesterID:    7,
			userID:         42,
			blocked:        true,
			wantErr:        repositories.ErrUserBlocked,
			wantUsername:   "grace",
			wantLookupRun:  true,
			wantBlockCheck: true,
		},
		{
			name:           "block repository error",
			username:       "grace",
			requesterID:    7,
			userID:         42,
			blockErr:       genericErr,
			wantErr:        genericErr,
			wantUsername:   "grace",
			wantLookupRun:  true,
			wantBlockCheck: true,
		},
		{
			name:           "review repository error",
			username:       "grace",
			requesterID:    7,
			userID:         42,
			reviewsErr:     genericErr,
			wantErr:        genericErr,
			wantUsername:   "grace",
			wantUserID:     42,
			wantLookupRun:  true,
			wantBlockCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUsername string
			var gotUserID int64
			var gotLimit, gotOffset int
			var gotBlockPair [2]int64
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
					if !tt.wantLookupRun {
						t.Fatalf("GetUserByUsername(%q) called; want no lookup", username)
					}
					gotUsername = username
					return &domains.User{ID: tt.userID, Username: username}, tt.userErr
				},
			}
			blockRepo := &testutil.FakeBlockRepository{
				IsBlockedBetweenFunc: func(_ context.Context, userID1, userID2 int64) (bool, error) {
					if !tt.wantBlockCheck {
						t.Fatalf("IsBlockedBetween(%d, %d) called; want no block check", userID1, userID2)
					}
					gotBlockPair = [2]int64{userID1, userID2}
					return tt.blocked, tt.blockErr
				},
			}
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromUserByUserIDFunc: func(_ context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
					gotUserID, gotLimit, gotOffset = userID, limit, offset
					return tt.reviews, tt.reviewsErr
				},
			}

			svc := NewReviewService(reviewRepo, userRepo, blockRepo, nil)
			reviews, err := svc.GetReviewsFromUserByUsername(context.Background(), tt.requesterID, tt.username, 20, tt.wantOffset)

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
			if gotBlockPair != [2]int64{tt.requesterID, tt.userID} {
				t.Fatalf("IsBlockedBetween() called with %v; want %v", gotBlockPair, [2]int64{tt.requesterID, tt.userID})
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

func TestGetReviewsFromMovieByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}

	tests := []struct {
		name        string
		movieID     int32
		limit       int
		offset      int
		reviews     []*domains.Review
		reviewsErr  error
		wantErr     error
		wantReviews []*domains.Review
	}{
		{
			name:        "success",
			movieID:     10,
			limit:       20,
			offset:      5,
			reviews:     wantReviews,
			wantReviews: wantReviews,
		},
		{
			name:       "repository error",
			movieID:    10,
			limit:      20,
			reviewsErr: genericErr,
			wantErr:    genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMovieID int32
			var gotLimit, gotOffset int
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsFromMovieByIDFunc: func(_ context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
					gotMovieID, gotLimit, gotOffset = movieID, limit, offset
					return tt.reviews, tt.reviewsErr
				},
			}

			svc := NewReviewService(reviewRepo, nil, nil, nil)
			reviews, err := svc.GetReviewsFromMovieByID(context.Background(), tt.movieID, tt.limit, tt.offset)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetReviewsFromMovieByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetReviewsFromMovieByID() error = %v; want nil", err)
			}
			if gotMovieID != tt.movieID || gotLimit != tt.limit || gotOffset != tt.offset {
				t.Fatalf("GetReviewsFromMovieByID() called with (%d, %d, %d); want (%d, %d, %d)",
					gotMovieID, gotLimit, gotOffset, tt.movieID, tt.limit, tt.offset)
			}
			if len(reviews) != len(tt.wantReviews) || reviews[0].ID != tt.wantReviews[0].ID {
				t.Fatalf("GetReviewsFromMovieByID() = %v; want %v", reviews, tt.wantReviews)
			}
		})
	}
}

func TestGetReviewByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	wantReview := &domains.Review{ID: 10, UserID: 42, MovieID: 3, Score: 8}

	tests := []struct {
		name       string
		reviewID   int64
		review     *domains.Review
		reviewErr  error
		wantErr    error
		wantReview *domains.Review
	}{
		{
			name:       "success",
			reviewID:   10,
			review:     wantReview,
			wantReview: wantReview,
		},
		{
			name:      "review not found",
			reviewID:  10,
			reviewErr: repositories.ErrReviewNotFound,
			wantErr:   repositories.ErrReviewNotFound,
		},
		{
			name:      "repository error",
			reviewID:  10,
			reviewErr: genericErr,
			wantErr:   genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReviewID int64
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewByIDFunc: func(_ context.Context, reviewID int64) (*domains.Review, error) {
					gotReviewID = reviewID
					return tt.review, tt.reviewErr
				},
			}

			svc := NewReviewService(reviewRepo, nil, nil, nil)
			review, err := svc.GetReviewByID(context.Background(), tt.reviewID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetReviewByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetReviewByID() error = %v; want nil", err)
			}
			if gotReviewID != tt.reviewID {
				t.Fatalf("GetReviewByID() called with %d; want %d", gotReviewID, tt.reviewID)
			}
			if review != tt.wantReview {
				t.Fatalf("GetReviewByID() = %v; want %v", review, tt.wantReview)
			}
		})
	}
}

func TestCreateReviewForMovieByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name         string
		score        int16
		repoErr      error
		wantErr      error
		wantRepoCall bool
	}{
		{
			name:         "success",
			score:        8,
			wantRepoCall: true,
		},
		{
			name:         "repository error",
			score:        8,
			repoErr:      genericErr,
			wantErr:      genericErr,
			wantRepoCall: true,
		},
		{
			name:    "score below minimum",
			score:   0,
			wantErr: ErrInvalidScore,
		},
		{
			name:    "score above maximum",
			score:   11,
			wantErr: ErrInvalidScore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReview *domains.Review
			reviewRepo := &testutil.FakeReviewRepository{
				CreateReviewForMovieByIDFunc: func(_ context.Context, review *domains.Review) (*domains.Review, error) {
					if !tt.wantRepoCall {
						t.Fatalf("CreateReviewForMovieByID() called; want no repository call")
					}
					gotReview = review
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					review.ID = 1
					return review, nil
				},
			}

			svc := NewReviewService(reviewRepo, nil, nil, nil)
			review := domains.NewReview(42, 10, tt.score, "great", false)
			created, err := svc.CreateReviewForMovieByID(context.Background(), review)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateReviewForMovieByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateReviewForMovieByID() error = %v; want nil", err)
			}
			if gotReview != review {
				t.Fatalf("CreateReviewForMovieByID() passed %v; want %v", gotReview, review)
			}
			if created.ID != 1 {
				t.Fatalf("CreateReviewForMovieByID() returned ID %d; want 1", created.ID)
			}
		})
	}
}

func TestDeleteReviewByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		userID      int64
		reviewID    int64
		repoErr     error
		wantErr     error
		wantRepoRun bool
	}{
		{
			name:        "success",
			userID:      42,
			reviewID:    10,
			wantRepoRun: true,
		},
		{
			name:        "review not found",
			userID:      42,
			reviewID:    10,
			repoErr:     repositories.ErrReviewNotFound,
			wantErr:     repositories.ErrReviewNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			userID:      42,
			reviewID:    10,
			repoErr:     genericErr,
			wantErr:     genericErr,
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

			svc := NewReviewService(reviewRepo, nil, nil, nil)
			err := svc.DeleteReviewByID(context.Background(), tt.userID, tt.reviewID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DeleteReviewByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteReviewByID() error = %v; want nil", err)
			}
			if gotUserID != tt.userID || gotReviewID != tt.reviewID {
				t.Fatalf("DeleteReviewByID() called with (%d, %d); want (%d, %d)",
					gotUserID, gotReviewID, tt.userID, tt.reviewID)
			}
		})
	}
}

func TestUpdateReviewByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	validScore := int16(9)
	validContent := "updated"
	validSpoilers := true
	invalidScore := int16(11)
	tooLongContent := strings.Repeat("a", 10001)
	wantReview := &domains.Review{
		ID:               1,
		UserID:           42,
		MovieID:          10,
		Score:            validScore,
		Content:          validContent,
		ContainsSpoilers: validSpoilers,
	}

	tests := []struct {
		name             string
		userID           int64
		reviewID         int64
		containsSpoilers *bool
		content          *string
		score            *int16
		repoErr          error
		wantErr          error
		wantRepoRun      bool
	}{
		{
			name:             "success with all fields",
			userID:           42,
			reviewID:         10,
			containsSpoilers: &validSpoilers,
			content:          &validContent,
			score:            &validScore,
			wantRepoRun:      true,
		},
		{
			name:        "success with only score",
			userID:      42,
			reviewID:    10,
			score:       &validScore,
			wantRepoRun: true,
		},
		{
			name:        "invalid score",
			userID:      42,
			reviewID:    10,
			score:       &invalidScore,
			wantErr:     ErrInvalidScore,
			wantRepoRun: false,
		},
		{
			name:        "content too long",
			userID:      42,
			reviewID:    10,
			content:     &tooLongContent,
			wantErr:     ErrReviewContentTooLong,
			wantRepoRun: false,
		},
		{
			name:        "review not found",
			userID:      42,
			reviewID:    10,
			score:       &validScore,
			repoErr:     repositories.ErrReviewNotFound,
			wantErr:     repositories.ErrReviewNotFound,
			wantRepoRun: true,
		},
		{
			name:        "repository error",
			userID:      42,
			reviewID:    10,
			score:       &validScore,
			repoErr:     genericErr,
			wantErr:     genericErr,
			wantRepoRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			var gotReviewID int64
			var gotContainsSpoilers *bool
			var gotContent *string
			var gotScore *int16
			reviewRepo := &testutil.FakeReviewRepository{
				UpdateReviewByIDFunc: func(_ context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
					if !tt.wantRepoRun {
						t.Fatalf("UpdateReviewByID() called; want no repository call")
					}
					gotUserID, gotReviewID = userID, reviewID
					gotContainsSpoilers, gotContent, gotScore = containsSpoilers, content, score
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return wantReview, nil
				},
			}

			svc := NewReviewService(reviewRepo, nil, nil, nil)
			review, err := svc.UpdateReviewByID(context.Background(), tt.userID, tt.reviewID, tt.containsSpoilers, tt.content, tt.score)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateReviewByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateReviewByID() error = %v; want nil", err)
			}
			if review != wantReview {
				t.Fatalf("UpdateReviewByID() = %v; want %v", review, wantReview)
			}
			if gotUserID != tt.userID || gotReviewID != tt.reviewID {
				t.Fatalf("UpdateReviewByID() called with user %d and review %d; want user %d and review %d",
					gotUserID, gotReviewID, tt.userID, tt.reviewID)
			}
			if gotContainsSpoilers != tt.containsSpoilers || gotContent != tt.content || gotScore != tt.score {
				t.Fatalf("UpdateReviewByID() forwarded wrong fields")
			}
		})
	}
}

func TestGetAverageRatingForMovieByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		movieID    int32
		rating     float64
		repoErr    error
		wantErr    error
		wantRating float64
	}{
		{name: "success", movieID: 550, rating: 7.5, wantRating: 7.5},
		{name: "no reviews", movieID: 550, rating: 0, wantRating: 0},
		{
			name:    "movie not found",
			movieID: 550,
			repoErr: repositories.ErrMovieNotFound,
			wantErr: repositories.ErrMovieNotFound,
		},
		{
			name:    "repository error",
			movieID: 550,
			repoErr: genericErr,
			wantErr: genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMovieID int32
			reviewRepo := &testutil.FakeReviewRepository{
				GetAverageRatingForMovieByIDFunc: func(_ context.Context, movieID int32) (float64, error) {
					gotMovieID = movieID
					return tt.rating, tt.repoErr
				},
			}
			svc := NewReviewService(reviewRepo, nil, nil, nil)

			got, err := svc.GetAverageRatingForMovieByID(context.Background(), tt.movieID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetAverageRatingForMovieByID() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetAverageRatingForMovieByID() error = %v; want nil", err)
			}
			if got != tt.wantRating {
				t.Fatalf("GetAverageRatingForMovieByID() = %v; want %v", got, tt.wantRating)
			}
			if gotMovieID != tt.movieID {
				t.Fatalf("GetAverageRatingForMovieByID() called with movieID %d; want %d", gotMovieID, tt.movieID)
			}
		})
	}
}

func TestGetReviewsCountForMovieByID(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		movieID   int32
		count     int
		repoErr   error
		wantErr   error
		wantCount int
	}{
		{name: "success", movieID: 550, count: 3, wantCount: 3},
		{name: "no reviews", movieID: 550, count: 0, wantCount: 0},
		{
			name:    "movie not found",
			movieID: 550,
			repoErr: repositories.ErrMovieNotFound,
			wantErr: repositories.ErrMovieNotFound,
		},
		{
			name:    "repository error",
			movieID: 550,
			repoErr: genericErr,
			wantErr: genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMovieID int32
			reviewRepo := &testutil.FakeReviewRepository{
				GetReviewsCountForMovieByIDFunc: func(_ context.Context, movieID int32) (int, error) {
					gotMovieID = movieID
					return tt.count, tt.repoErr
				},
			}
			svc := NewReviewService(reviewRepo, nil, nil, nil)

			got, err := svc.GetReviewsCountForMovieByID(context.Background(), tt.movieID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetReviewsCountForMovieByID() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetReviewsCountForMovieByID() error = %v; want nil", err)
			}
			if got != tt.wantCount {
				t.Fatalf("GetReviewsCountForMovieByID() = %v; want %v", got, tt.wantCount)
			}
			if gotMovieID != tt.movieID {
				t.Fatalf("GetReviewsCountForMovieByID() called with movieID %d; want %d", gotMovieID, tt.movieID)
			}
		})
	}
}
