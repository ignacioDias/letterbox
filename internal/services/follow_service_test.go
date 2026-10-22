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

func TestFollowUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		username  string
		follower  int64
		targetID  int64
		targetErr error
		blocked   bool
		blockErr  error
		followErr error
		wantErr   error
		wantTuple [2]int64
	}{
		{
			name:      "success",
			username:  "grace",
			follower:  7,
			targetID:  42,
			wantTuple: [2]int64{7, 42},
		},
		{
			name:     "cannot follow self",
			username: "ada",
			follower: 7,
			targetID: 7,
			wantErr:  ErrCannotFollowSelf,
		},
		{
			name:     "invalid username",
			username: "a",
			follower: 7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:      "target not found",
			username:  "ghost",
			follower:  7,
			targetErr: repositories.ErrUserNotFound,
			wantErr:   repositories.ErrUserNotFound,
		},
		{
			name:     "blocked by target",
			username: "grace",
			follower: 7,
			targetID: 42,
			blocked:  true,
			wantErr:  repositories.ErrUserBlocked,
		},
		{
			name:     "block check fails",
			username: "grace",
			follower: 7,
			targetID: 42,
			blockErr: genericErr,
			wantErr:  genericErr,
		},
		{
			name:      "already following",
			username:  "grace",
			follower:  7,
			targetID:  42,
			followErr: repositories.ErrAlreadyFollowing,
			wantErr:   repositories.ErrAlreadyFollowing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTuple [2]int64
			followRepo := &testutil.FakeFollowRepository{
				FollowUserFunc: func(_ context.Context, follower, following int64) error {
					gotTuple = [2]int64{follower, following}
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

			svc := NewFollowService(followRepo, userRepo, blockRepo)
			err := svc.FollowUserByUsername(context.Background(), tt.follower, tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FollowUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FollowUserByUsername() error = %v; want nil", err)
			}
			if gotTuple != tt.wantTuple {
				t.Fatalf("FollowUser() called with %v; want %v", gotTuple, tt.wantTuple)
			}
		})
	}
}

func TestUnfollowUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		username  string
		follower  int64
		targetID  int64
		targetErr error
		unfollow  error
		wantErr   error
		wantTuple [2]int64
	}{
		{
			name:      "success",
			username:  "grace",
			follower:  7,
			targetID:  42,
			wantTuple: [2]int64{7, 42},
		},
		{
			name:     "cannot unfollow self",
			username: "ada",
			follower: 7,
			targetID: 7,
			wantErr:  ErrCannotUnfollowSelf,
		},
		{
			name:     "invalid username",
			username: "",
			follower: 7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:      "target not found",
			username:  "ghost",
			follower:  7,
			targetErr: repositories.ErrUserNotFound,
			wantErr:   repositories.ErrUserNotFound,
		},
		{
			name:     "not following",
			username: "grace",
			follower: 7,
			targetID: 42,
			unfollow: repositories.ErrNotFollowing,
			wantErr:  repositories.ErrNotFollowing,
		},
		{
			name:     "unexpected repository error",
			username: "grace",
			follower: 7,
			targetID: 42,
			unfollow: genericErr,
			wantErr:  genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTuple [2]int64
			followRepo := &testutil.FakeFollowRepository{
				UnfollowUserFunc: func(_ context.Context, follower, following int64) error {
					gotTuple = [2]int64{follower, following}
					return tt.unfollow
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			svc := NewFollowService(followRepo, userRepo, &testutil.FakeBlockRepository{})
			err := svc.UnfollowUserByUsername(context.Background(), tt.follower, tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UnfollowUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnfollowUserByUsername() error = %v; want nil", err)
			}
			if gotTuple != tt.wantTuple {
				t.Fatalf("UnfollowUser() called with %v; want %v", gotTuple, tt.wantTuple)
			}
		})
	}
}

func TestFollowListings(t *testing.T) {
	users := []*domains.PublicUser{{ID: 1, Username: "grace"}}

	followRepo := &testutil.FakeFollowRepository{
		GetFollowersByUsernameFunc: func(_ context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
			if username != "ada" || limit != 10 || offset != 5 {
				t.Fatalf("GetFollowersByUsername(%q, %d, %d); want ada, 10, 5", username, limit, offset)
			}
			return users, nil
		},
		GetFollowingByUsernameFunc: func(_ context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
			if username != "ada" || limit != 10 || offset != 5 {
				t.Fatalf("GetFollowingByUsername(%q, %d, %d); want ada, 10, 5", username, limit, offset)
			}
			return users, nil
		},
	}
	svc := NewFollowService(followRepo, &testutil.FakeUserRepository{}, &testutil.FakeBlockRepository{})
	ctx := context.Background()

	got, err := svc.GetFollowersByUsername(ctx, "  ada  ", 10, 5)
	if err != nil {
		t.Fatalf("GetFollowersByUsername: %v", err)
	}
	if len(got) != 1 || got[0].Username != "grace" {
		t.Fatalf("GetFollowersByUsername returned %+v", got)
	}

	got, err = svc.GetFollowingByUsername(ctx, "ada", 10, 5)
	if err != nil {
		t.Fatalf("GetFollowingByUsername: %v", err)
	}
	if len(got) != 1 || got[0].Username != "grace" {
		t.Fatalf("GetFollowingByUsername returned %+v", got)
	}

	if _, err := svc.GetFollowersByUsername(ctx, "a", 10, 0); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("GetFollowersByUsername invalid error = %v; want ErrInvalidUsername", err)
	}
	if _, err := svc.GetFollowingByUsername(ctx, "a", 10, 0); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("GetFollowingByUsername invalid error = %v; want ErrInvalidUsername", err)
	}
}

func testFollowCount(t *testing.T, call func(context.Context, FollowService, string) (int, error)) {
	t.Helper()

	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		username  string
		userID    int64
		userErr   error
		count     int
		countErr  error
		wantCount int
		wantErr   error
	}{
		{name: "success", username: "ada", userID: 7, count: 3, wantCount: 3},
		{name: "trims username", username: "  ada  ", userID: 7, count: 3, wantCount: 3},
		{name: "empty username", username: "", wantErr: ErrInvalidUsername},
		{name: "whitespace username", username: "   ", wantErr: ErrInvalidUsername},
		{name: "short username", username: "a", wantErr: ErrInvalidUsername},
		{name: "user not found", username: "ghost", userErr: repositories.ErrUserNotFound, wantErr: repositories.ErrUserNotFound},
		{name: "user repository error", username: "ada", userErr: genericErr, wantErr: genericErr},
		{name: "count repository error", username: "ada", userID: 7, countErr: genericErr, wantErr: genericErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			followRepo := &testutil.FakeFollowRepository{
				GetFollowersCountByUserIDFunc: func(_ context.Context, userID int64) (int, error) {
					gotUserID = userID
					return tt.count, tt.countErr
				},
				GetFollowingCountByUserIDFunc: func(_ context.Context, userID int64) (int, error) {
					gotUserID = userID
					return tt.count, tt.countErr
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(_ context.Context, username string) (int64, error) {
					if errors.Is(tt.wantErr, ErrInvalidUsername) {
						t.Fatalf("GetUserIDByUsername(%q) called for invalid username", username)
					}
					if username != strings.TrimSpace(tt.username) {
						t.Fatalf("GetUserIDByUsername(%q); want %q", username, strings.TrimSpace(tt.username))
					}
					return tt.userID, tt.userErr
				},
			}

			svc := NewFollowService(followRepo, userRepo, &testutil.FakeBlockRepository{})
			got, err := call(context.Background(), svc, tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v; want nil", err)
			}
			if got != tt.wantCount {
				t.Fatalf("count = %d; want %d", got, tt.wantCount)
			}
			if gotUserID != tt.userID {
				t.Fatalf("count called with userID %d; want %d", gotUserID, tt.userID)
			}
		})
	}
}

func TestGetFollowersCountByUsername(t *testing.T) {
	testFollowCount(t, func(ctx context.Context, svc FollowService, username string) (int, error) {
		return svc.GetFollowersCountByUsername(ctx, username)
	})
}

func TestGetFollowingCountByUsername(t *testing.T) {
	testFollowCount(t, func(ctx context.Context, svc FollowService, username string) (int, error) {
		return svc.GetFollowingCountByUsername(ctx, username)
	})
}
