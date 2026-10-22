package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	"golang.org/x/crypto/bcrypt"
)

func TestGetUserByUsername(t *testing.T) {
	genericErr := errors.New("database unavailable")
	user := &domains.User{ID: 7, Username: "ada"}

	tests := []struct {
		name      string
		username  string
		user      *domains.User
		repoErr   error
		wantErr   error
		wantInput string
	}{
		{name: "success", username: "  ada  ", user: user, wantInput: "ada"},
		{name: "invalid username", username: "a", wantErr: ErrInvalidUsername},
		{name: "not found", username: "ghost", repoErr: repositories.ErrUserNotFound, wantErr: repositories.ErrUserNotFound, wantInput: "ghost"},
		{name: "repository error", username: "ada", repoErr: genericErr, wantErr: genericErr, wantInput: "ada"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotInput string
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
					gotInput = username
					return tt.user, tt.repoErr
				},
			}
			svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})

			got, err := svc.GetUserByUsername(context.Background(), tt.username)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetUserByUsername() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUserByUsername() error = %v; want nil", err)
			}
			if got != tt.user {
				t.Fatalf("GetUserByUsername() = %+v; want %+v", got, tt.user)
			}
			if gotInput != tt.wantInput {
				t.Fatalf("repository called with %q; want %q", gotInput, tt.wantInput)
			}
		})
	}
}

func TestUpdateUserByID(t *testing.T) {
	current := &domains.User{
		ID:                7,
		Username:          "ada",
		Email:             "ada@example.com",
		PasswordHash:      "existing-hash",
		ProfilePictureURL: "pic.png",
		Country:           "AR",
	}
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name            string
		userID          int64
		username        string
		email           string
		country         string
		profilePicture  string
		getErr          error
		updateErr       error
		wantErr         error
		wantUser        *domains.User
		wantUpdateCalls int
	}{
		{
			name:            "keeps omitted fields",
			userID:          7,
			country:         "UY",
			wantUser:        &domains.User{ID: 7, Username: "ada", Email: "ada@example.com", PasswordHash: "existing-hash", ProfilePictureURL: "pic.png", Country: "UY"},
			wantUpdateCalls: 1,
		},
		{
			name:            "updates all fields",
			userID:          7,
			username:        "ada_l",
			email:           "ada@lovelace.com",
			country:         "GB",
			profilePicture:  "new.png",
			wantUser:        &domains.User{ID: 7, Username: "ada_l", Email: "ada@lovelace.com", PasswordHash: "existing-hash", ProfilePictureURL: "new.png", Country: "GB"},
			wantUpdateCalls: 1,
		},
		{
			name:     "invalid username",
			userID:   7,
			username: "a",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:    "invalid email",
			userID:  7,
			email:   "nope",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "user not found",
			userID:  7,
			getErr:  repositories.ErrUserNotFound,
			wantErr: repositories.ErrUserNotFound,
		},
		{
			name:            "update fails",
			userID:          7,
			country:         "UY",
			updateErr:       genericErr,
			wantErr:         genericErr,
			wantUpdateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updated *domains.User
			updateCalls := 0
			userRepo := &testutil.FakeUserRepository{
				GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return current, nil
				},
				UpdateUserFunc: func(_ context.Context, user *domains.User) error {
					updateCalls++
					updated = user
					return tt.updateErr
				},
			}
			svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})

			err := svc.UpdateUserByID(context.Background(), tt.userID, tt.username, tt.email, tt.country, tt.profilePicture)

			if updateCalls != tt.wantUpdateCalls {
				t.Fatalf("UpdateUser called %d times; want %d", updateCalls, tt.wantUpdateCalls)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateUserByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateUserByID() error = %v; want nil", err)
			}
			if updated == nil || *updated != *tt.wantUser {
				t.Fatalf("UpdateUser() called with %+v; want %+v", updated, tt.wantUser)
			}
		})
	}
}

func TestUpdateUserPasswordByID(t *testing.T) {
	genericErr := errors.New("database unavailable")
	currentHash, err := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	tests := []struct {
		name            string
		currentPassword string
		newPassword     string
		hash            string
		hashErr         error
		updateErr       error
		deleteErr       error
		wantErr         error
		wantUpdateCalls int
		wantDeleteCalls int
		wantNewPassword string
	}{
		{
			name:            "success",
			currentPassword: "oldpassword",
			newPassword:     "newpassword",
			hash:            string(currentHash),
			wantUpdateCalls: 1,
			wantDeleteCalls: 1,
			wantNewPassword: "newpassword",
		},
		{
			name:            "invalid current password format",
			currentPassword: "short",
			newPassword:     "newpassword",
			wantErr:         ErrInvalidPassword,
		},
		{
			name:            "invalid new password format",
			currentPassword: "oldpassword",
			newPassword:     "short",
			hash:            string(currentHash),
			wantErr:         ErrInvalidNewPassword,
		},
		{
			name:            "wrong current password",
			currentPassword: "wrongpassword",
			newPassword:     "newpassword",
			hash:            string(currentHash),
			wantErr:         ErrInvalidPassword,
		},
		{
			name:            "hashed password lookup fails",
			currentPassword: "oldpassword",
			newPassword:     "newpassword",
			hashErr:         genericErr,
			wantErr:         genericErr,
		},
		{
			name:            "update fails",
			currentPassword: "oldpassword",
			newPassword:     "newpassword",
			hash:            string(currentHash),
			updateErr:       genericErr,
			wantErr:         genericErr,
			wantUpdateCalls: 1,
		},
		{
			name:            "deleting other sessions fails",
			currentPassword: "oldpassword",
			newPassword:     "newpassword",
			hash:            string(currentHash),
			deleteErr:       genericErr,
			wantErr:         genericErr,
			wantUpdateCalls: 1,
			wantDeleteCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			deleteCalls := 0
			var newHash string
			userRepo := &testutil.FakeUserRepository{
				GetUserHashedPasswordFunc: func(context.Context, int64) (string, error) {
					return tt.hash, tt.hashErr
				},
				UpdatePasswordByIDFunc: func(_ context.Context, userID int64, hash string) error {
					updateCalls++
					newHash = hash
					return tt.updateErr
				},
			}
			sessionRepo := &testutil.FakeSessionRepository{
				DeleteOtherSessionsFunc: func(_ context.Context, userID int64, keepSessionID string) error {
					deleteCalls++
					if userID != 7 || keepSessionID != "session-1" {
						t.Fatalf("DeleteOtherSessions(%d, %q); want 7, session-1", userID, keepSessionID)
					}
					return tt.deleteErr
				},
			}
			svc := NewUserService(userRepo, sessionRepo)

			err := svc.UpdateUserPasswordByID(context.Background(), 7, "session-1", tt.currentPassword, tt.newPassword)

			if updateCalls != tt.wantUpdateCalls {
				t.Fatalf("UpdatePasswordByID called %d times; want %d", updateCalls, tt.wantUpdateCalls)
			}
			if deleteCalls != tt.wantDeleteCalls {
				t.Fatalf("DeleteOtherSessions called %d times; want %d", deleteCalls, tt.wantDeleteCalls)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateUserPasswordByID() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateUserPasswordByID() error = %v; want nil", err)
			}
			if err := checkPassword(newHash, tt.wantNewPassword); err != nil {
				t.Fatalf("stored hash does not match new password: %v", err)
			}
		})
	}
}

func TestUserPassThroughOperations(t *testing.T) {
	genericErr := errors.New("database unavailable")
	user := &domains.User{ID: 7, Username: "ada"}

	t.Run("get by id", func(t *testing.T) {
		userRepo := &testutil.FakeUserRepository{
			GetUserByIDFunc: func(_ context.Context, userID int64) (*domains.User, error) {
				if userID != 7 {
					t.Fatalf("GetUserByID(%d); want 7", userID)
				}
				return user, nil
			},
		}
		svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})
		got, err := svc.GetUserByID(context.Background(), 7)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if got != user {
			t.Fatalf("GetUserByID() = %+v; want %+v", got, user)
		}
	})

	t.Run("get by id error", func(t *testing.T) {
		userRepo := &testutil.FakeUserRepository{
			GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
				return nil, genericErr
			},
		}
		svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})
		if _, err := svc.GetUserByID(context.Background(), 7); !errors.Is(err, genericErr) {
			t.Fatalf("GetUserByID() error = %v; want %v", err, genericErr)
		}
	})

	t.Run("delete by id", func(t *testing.T) {
		userRepo := &testutil.FakeUserRepository{
			DeleteUserByIDFunc: func(_ context.Context, userID int64) error {
				if userID != 7 {
					t.Fatalf("DeleteUserByID(%d); want 7", userID)
				}
				return nil
			},
		}
		svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})
		if err := svc.DeleteUserByID(context.Background(), 7); err != nil {
			t.Fatalf("DeleteUserByID: %v", err)
		}
	})

	t.Run("delete by id error", func(t *testing.T) {
		userRepo := &testutil.FakeUserRepository{
			DeleteUserByIDFunc: func(context.Context, int64) error {
				return genericErr
			},
		}
		svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})
		if err := svc.DeleteUserByID(context.Background(), 7); !errors.Is(err, genericErr) {
			t.Fatalf("DeleteUserByID() error = %v; want %v", err, genericErr)
		}
	})
}
