package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"letterbox/internal/cache"
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

			svc := NewReviewService(reviewRepo, userRepo, blockRepo)
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

			svc := NewReviewService(reviewRepo, nil, nil)
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

			svc := NewReviewService(reviewRepo, nil, nil)
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

			svc := NewReviewService(reviewRepo, nil, nil)
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

			svc := NewReviewService(reviewRepo, nil, nil)
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

			svc := NewReviewService(reviewRepo, nil, nil)
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
			svc := NewReviewService(reviewRepo, nil, nil)

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
			svc := NewReviewService(reviewRepo, nil, nil)

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

func reviewTestCache(getFunc func(context.Context, string) (string, error), setFunc func(context.Context, string, string, int) error, deleteFunc func(context.Context, []string) (int64, error)) *testutil.FakeCache {
	return &testutil.FakeCache{
		GetFunc: getFunc,
		SetFunc: func(ctx context.Context, key string, value string, expiringTimeInSeconds int) error {
			if setFunc == nil {
				return nil
			}
			return setFunc(ctx, key, value, expiringTimeInSeconds)
		},
		DeleteFunc: func(ctx context.Context, keys []string) (int64, error) {
			if deleteFunc == nil {
				return 0, nil
			}
			return deleteFunc(ctx, keys)
		},
	}
}

func TestGetReviewsFromUserByUsernameCacheHit(t *testing.T) {
	reviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}
	value, err := generateCacheValueFromReviews(reviews)
	if err != nil {
		t.Fatalf("encode cached reviews: %v", err)
	}
	generation := "generation-1"
	listKey := getKeyForUserReviews(42, 20, 5, generation)
	generationKey := getKeyForUserReviewsGeneration(42)
	repoCalled := false

	fakeCache := reviewTestCache(func(_ context.Context, key string) (string, error) {
		switch key {
		case generationKey:
			return generation, nil
		case listKey:
			return value, nil
		default:
			return "", cache.ErrKeyNotFound
		}
	}, nil, nil)
	userRepo := &testutil.FakeUserRepository{
		GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
			return &domains.User{ID: 42, Username: username}, nil
		},
	}
	blockRepo := &testutil.FakeBlockRepository{
		IsBlockedBetweenFunc: func(context.Context, int64, int64) (bool, error) {
			return false, nil
		},
	}
	reviewRepo := &testutil.FakeReviewRepository{
		GetReviewsFromUserByUserIDFunc: func(context.Context, int64, int, int) ([]*domains.Review, error) {
			repoCalled = true
			return nil, errors.New("repository should not be called")
		},
	}
	svc := NewReviewService(reviewRepo, userRepo, blockRepo, fakeCache)

	got, err := svc.GetReviewsFromUserByUsername(context.Background(), 7, "grace", 20, 5)
	if err != nil {
		t.Fatalf("GetReviewsFromUserByUsername: %v", err)
	}
	if repoCalled {
		t.Fatal("GetReviewsFromUserByUsername called repository on cache hit")
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("GetReviewsFromUserByUsername returned %+v", got)
	}
}

func TestGetReviewsFromMovieByIDCacheMissCachesResult(t *testing.T) {
	reviews := []*domains.Review{{ID: 1, UserID: 42, MovieID: 10, Score: 8}}
	wantKey := getKeyForMovieReviews(10, 20, 0, "0")
	var setKey string
	var setTTL int

	fakeCache := reviewTestCache(func(context.Context, string) (string, error) {
		return "", cache.ErrKeyNotFound
	}, func(_ context.Context, key string, _ string, expiringTimeInSeconds int) error {
		setKey = key
		setTTL = expiringTimeInSeconds
		return nil
	}, nil)
	reviewRepo := &testutil.FakeReviewRepository{
		GetReviewsFromMovieByIDFunc: func(_ context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
			if movieID != 10 || limit != 20 || offset != 0 {
				t.Fatalf("GetReviewsFromMovieByID(%d, %d, %d); want 10, 20, 0", movieID, limit, offset)
			}
			return reviews, nil
		},
	}
	svc := NewReviewService(reviewRepo, nil, nil, fakeCache)

	got, err := svc.GetReviewsFromMovieByID(context.Background(), 10, 20, 0)
	if err != nil {
		t.Fatalf("GetReviewsFromMovieByID: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("GetReviewsFromMovieByID returned %+v", got)
	}
	if setKey != wantKey {
		t.Fatalf("cache.Set key = %q; want %q", setKey, wantKey)
	}
	if setTTL != reviewCacheTTLSeconds {
		t.Fatalf("cache.Set TTL = %d; want %d", setTTL, reviewCacheTTLSeconds)
	}
}

func TestGetReviewByIDCacheHit(t *testing.T) {
	review := &domains.Review{ID: 5, UserID: 42, MovieID: 10, Score: 8}
	value, err := generateCacheValueFromReview(review)
	if err != nil {
		t.Fatalf("encode cached review: %v", err)
	}
	repoCalled := false
	fakeCache := reviewTestCache(func(_ context.Context, key string) (string, error) {
		if key != getKeyForReviewByID(5) {
			t.Fatalf("cache.Get(%q); want %q", key, getKeyForReviewByID(5))
		}
		return value, nil
	}, nil, nil)
	reviewRepo := &testutil.FakeReviewRepository{
		GetReviewByIDFunc: func(context.Context, int64) (*domains.Review, error) {
			repoCalled = true
			return nil, errors.New("repository should not be called")
		},
	}
	svc := NewReviewService(reviewRepo, nil, nil, fakeCache)

	got, err := svc.GetReviewByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetReviewByID: %v", err)
	}
	if repoCalled {
		t.Fatal("GetReviewByID called repository on cache hit")
	}
	if got.ID != 5 {
		t.Fatalf("GetReviewByID returned %+v", got)
	}
}

func TestReviewAggregatesCacheHit(t *testing.T) {
	fakeCache := reviewTestCache(func(_ context.Context, key string) (string, error) {
		switch key {
		case getKeyForMovieReviewAverage(10):
			return "7.5", nil
		case getKeyForMovieReviewCount(10):
			return "3", nil
		default:
			return "", cache.ErrKeyNotFound
		}
	}, nil, nil)
	reviewRepo := &testutil.FakeReviewRepository{
		GetAverageRatingForMovieByIDFunc: func(context.Context, int32) (float64, error) {
			t.Fatal("GetAverageRatingForMovieByID called on cache hit")
			return 0, nil
		},
		GetReviewsCountForMovieByIDFunc: func(context.Context, int32) (int, error) {
			t.Fatal("GetReviewsCountForMovieByID called on cache hit")
			return 0, nil
		},
	}
	svc := NewReviewService(reviewRepo, nil, nil, fakeCache)

	rating, err := svc.GetAverageRatingForMovieByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetAverageRatingForMovieByID: %v", err)
	}
	if rating != 7.5 {
		t.Fatalf("average rating = %v; want 7.5", rating)
	}
	count, err := svc.GetReviewsCountForMovieByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetReviewsCountForMovieByID: %v", err)
	}
	if count != 3 {
		t.Fatalf("review count = %d; want 3", count)
	}
}

func TestCreateReviewForMovieByIDInvalidatesCache(t *testing.T) {
	var deletedKeys []string
	generationKeys := map[string]bool{}
	fakeCache := reviewTestCache(func(context.Context, string) (string, error) {
		return "", cache.ErrKeyNotFound
	}, func(_ context.Context, key string, _ string, expiringTimeInSeconds int) error {
		if expiringTimeInSeconds != reviewCacheTTLSeconds {
			t.Fatalf("cache.Set TTL = %d; want %d", expiringTimeInSeconds, reviewCacheTTLSeconds)
		}
		generationKeys[key] = true
		return nil
	}, func(_ context.Context, keys []string) (int64, error) {
		deletedKeys = append(deletedKeys, keys...)
		return int64(len(keys)), nil
	})
	reviewRepo := &testutil.FakeReviewRepository{
		CreateReviewForMovieByIDFunc: func(_ context.Context, review *domains.Review) (*domains.Review, error) {
			review.ID = 9
			return review, nil
		},
	}
	svc := NewReviewService(reviewRepo, nil, nil, fakeCache)

	created, err := svc.CreateReviewForMovieByID(context.Background(), domains.NewReview(42, 10, 8, "great", false))
	if err != nil {
		t.Fatalf("CreateReviewForMovieByID: %v", err)
	}
	if created.ID != 9 {
		t.Fatalf("CreateReviewForMovieByID returned ID %d; want 9", created.ID)
	}

	wantGenerations := []string{
		getKeyForUserReviewsGeneration(42),
		getKeyForMovieReviewsGeneration(10),
		getKeyForFollowingReviewsGeneration(),
	}
	for _, key := range wantGenerations {
		if !generationKeys[key] {
			t.Fatalf("cache.Set did not include %q (set %v)", key, generationKeys)
		}
	}
	wantDeleted := map[string]bool{
		getKeyForMovieReviewAverage(10): false,
		getKeyForMovieReviewCount(10):   false,
		getKeyForReviewByID(9):          false,
	}
	for _, key := range deletedKeys {
		if _, ok := wantDeleted[key]; ok {
			wantDeleted[key] = true
		}
	}
	for key, found := range wantDeleted {
		if !found {
			t.Fatalf("cache.Delete did not include %q (deleted %v)", key, deletedKeys)
		}
	}
}

func TestDeleteReviewByIDInvalidatesCache(t *testing.T) {
	var deletedKeys []string
	generationKeys := map[string]bool{}
	fakeCache := reviewTestCache(func(context.Context, string) (string, error) {
		return "", cache.ErrKeyNotFound
	}, func(_ context.Context, key string, _ string, _ int) error {
		generationKeys[key] = true
		return nil
	}, func(_ context.Context, keys []string) (int64, error) {
		deletedKeys = append(deletedKeys, keys...)
		return int64(len(keys)), nil
	})
	reviewRepo := &testutil.FakeReviewRepository{
		GetReviewByIDFunc: func(context.Context, int64) (*domains.Review, error) {
			return &domains.Review{ID: 9, UserID: 42, MovieID: 10}, nil
		},
		DeleteReviewByIDFunc: func(context.Context, int64, int64) error {
			return nil
		},
	}
	svc := NewReviewService(reviewRepo, nil, nil, fakeCache)

	if err := svc.DeleteReviewByID(context.Background(), 42, 9); err != nil {
		t.Fatalf("DeleteReviewByID: %v", err)
	}

	for _, key := range []string{
		getKeyForUserReviewsGeneration(42),
		getKeyForMovieReviewsGeneration(10),
		getKeyForFollowingReviewsGeneration(),
	} {
		if !generationKeys[key] {
			t.Fatalf("cache.Set did not include %q (set %v)", key, generationKeys)
		}
	}
	foundReviewKey := false
	for _, key := range deletedKeys {
		if key == getKeyForReviewByID(9) {
			foundReviewKey = true
		}
	}
	if !foundReviewKey {
		t.Fatalf("cache.Delete did not include %q (deleted %v)", getKeyForReviewByID(9), deletedKeys)
	}
}
