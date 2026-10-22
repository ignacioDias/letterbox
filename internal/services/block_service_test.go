package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestBlockUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		username  string
		blocker   int64
		targetID  int64
		targetErr error
		blockErr  error
		wantErr   error
		wantTuple [2]int64
	}{
		{
			name:      "success",
			username:  "grace",
			blocker:   7,
			targetID:  42,
			wantTuple: [2]int64{7, 42},
		},
		{
			name:     "cannot block self",
			username: "ada",
			blocker:  7,
			targetID: 7,
			wantErr:  ErrCannotBlockSelf,
		},
		{
			name:     "empty username",
			username: "",
			blocker:  7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid username",
			username: "a",
			blocker:  7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:      "target not found",
			username:  "ghost",
			blocker:   7,
			targetErr: repositories.ErrUserNotFound,
			wantErr:   repositories.ErrUserNotFound,
		},
		{
			name:     "already blocked",
			username: "grace",
			blocker:  7,
			targetID: 42,
			blockErr: repositories.ErrAlreadyBlocked,
			wantErr:  repositories.ErrAlreadyBlocked,
		},
		{
			name:     "unexpected repository error",
			username: "grace",
			blocker:  7,
			targetID: 42,
			blockErr: genericErr,
			wantErr:  genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTuple [2]int64
			blockRepo := &testutil.FakeBlockRepository{
				BlockUserFunc: func(_ context.Context, blocker, blocked int64) error {
					gotTuple = [2]int64{blocker, blocked}
					return tt.blockErr
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			svc := NewBlockService(blockRepo, userRepo)
			err := svc.BlockUserByUsername(context.Background(), tt.blocker, tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("BlockUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BlockUserByUsername() error = %v; want nil", err)
			}
			if gotTuple != tt.wantTuple {
				t.Fatalf("BlockUser() called with %v; want %v", gotTuple, tt.wantTuple)
			}
		})
	}
}

func TestUnblockUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		username  string
		blocker   int64
		targetID  int64
		targetErr error
		unblock   error
		wantErr   error
		wantTuple [2]int64
	}{
		{
			name:      "success",
			username:  "grace",
			blocker:   7,
			targetID:  42,
			wantTuple: [2]int64{7, 42},
		},
		{
			name:     "empty username",
			username: "",
			blocker:  7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid username",
			username: "a",
			blocker:  7,
			wantErr:  ErrInvalidUsername,
		},
		{
			name:      "target not found",
			username:  "ghost",
			blocker:   7,
			targetErr: repositories.ErrUserNotFound,
			wantErr:   repositories.ErrUserNotFound,
		},
		{
			name:     "not blocked",
			username: "grace",
			blocker:  7,
			targetID: 42,
			unblock:  repositories.ErrNotBlocked,
			wantErr:  repositories.ErrNotBlocked,
		},
		{
			name:     "unexpected repository error",
			username: "grace",
			blocker:  7,
			targetID: 42,
			unblock:  genericErr,
			wantErr:  genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTuple [2]int64
			blockRepo := &testutil.FakeBlockRepository{
				UnblockUserFunc: func(_ context.Context, blocker, blocked int64) error {
					gotTuple = [2]int64{blocker, blocked}
					return tt.unblock
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			svc := NewBlockService(blockRepo, userRepo)
			err := svc.UnblockUserByUsername(context.Background(), tt.blocker, tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UnblockUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnblockUserByUsername() error = %v; want nil", err)
			}
			if gotTuple != tt.wantTuple {
				t.Fatalf("UnblockUser() called with %v; want %v", gotTuple, tt.wantTuple)
			}
		})
	}
}
