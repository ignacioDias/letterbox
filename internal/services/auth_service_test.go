package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	"golang.org/x/crypto/bcrypt"
)

func TestRegisterUser(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name     string
		username string
		email    string
		password string
		repoErr  error
		wantErr  error
	}{
		{name: "success", username: "ada", email: "ada@example.com", password: "secret123"},
		{name: "trims input", username: "  ada  ", email: "  ada@example.com  ", password: "secret123"},
		{name: "invalid username", username: "a", email: "ada@example.com", password: "secret123", wantErr: ErrInvalidUsername},
		{name: "invalid email", username: "ada", email: "not-an-email", password: "secret123", wantErr: ErrInvalidEmail},
		{name: "invalid password", username: "ada", email: "ada@example.com", password: "short", wantErr: ErrInvalidPassword},
		{name: "username taken", username: "ada", email: "ada@example.com", password: "secret123", repoErr: repositories.ErrUsernameAlreadyExists, wantErr: repositories.ErrUsernameAlreadyExists},
		{name: "repository error", username: "ada", email: "ada@example.com", password: "secret123", repoErr: genericErr, wantErr: genericErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created *domains.User
			userRepo := &testutil.FakeUserRepository{
				CreateUserFunc: func(_ context.Context, user *domains.User) (int64, error) {
					created = user
					return 99, tt.repoErr
				},
			}
			svc := NewAuthService(userRepo, &testutil.FakeSessionRepository{})

			user, err := svc.RegisterUser(context.Background(), tt.username, tt.email, tt.password, "", "AR")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RegisterUser() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RegisterUser() error = %v; want nil", err)
			}
			if user == nil || created == nil {
				t.Fatal("expected the user to be created")
			}
			if user.ID != 99 {
				t.Fatalf("user ID = %d; want 99", user.ID)
			}
			if user.Username != "ada" || user.Email != "ada@example.com" || user.Country != "AR" {
				t.Fatalf("unexpected user: %+v", user)
			}
			if user.ProfilePictureURL != DefaultProfilePictureURL {
				t.Fatalf("ProfilePictureURL = %q; want %q", user.ProfilePictureURL, DefaultProfilePictureURL)
			}
			if user.PasswordHash == tt.password {
				t.Fatal("password was not hashed")
			}
			if err := checkPassword(user.PasswordHash, tt.password); err != nil {
				t.Fatalf("stored hash does not match password: %v", err)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	genericErr := errors.New("database unavailable")
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com", PasswordHash: string(passwordHash)}

	tests := []struct {
		name      string
		input     string
		password  string
		user      *domains.User
		repoErr   error
		wantUser  *domains.User
		wantErr   error
		wantInput string
	}{
		{
			name:      "username",
			input:     "  ada  ",
			password:  "secret123",
			user:      user,
			wantUser:  user,
			wantInput: "ada",
		},
		{
			name:      "email",
			input:     "ada@example.com",
			password:  "secret123",
			user:      user,
			wantUser:  user,
			wantInput: "ada@example.com",
		},
		{
			name:      "wrong password",
			input:     "ada",
			password:  "wrong-password",
			user:      user,
			wantErr:   ErrInvalidCredentials,
			wantInput: "ada",
		},
		{
			name:      "unknown user",
			input:     "ghost",
			password:  "secret123",
			repoErr:   repositories.ErrUserNotFound,
			wantErr:   ErrInvalidCredentials,
			wantInput: "ghost",
		},
		{
			name:      "repository error",
			input:     "ada",
			password:  "secret123",
			repoErr:   genericErr,
			wantErr:   genericErr,
			wantInput: "ada",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotInput string
			userRepo := &testutil.FakeUserRepository{
				GetUserByEmailOrUsernameFunc: func(_ context.Context, input string) (*domains.User, error) {
					gotInput = input
					return tt.user, tt.repoErr
				},
			}
			svc := NewAuthService(userRepo, &testutil.FakeSessionRepository{})

			got, err := svc.LoginUser(context.Background(), tt.input, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("LoginUser() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("LoginUser() error = %v; want nil", err)
			}
			if got != tt.wantUser {
				t.Fatalf("LoginUser() = %+v; want %+v", got, tt.wantUser)
			}
			if gotInput != tt.wantInput {
				t.Fatalf("repository called with %q; want %q", gotInput, tt.wantInput)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	var stored *domains.Session
	sessionRepo := &testutil.FakeSessionRepository{
		CreateSessionFunc: func(_ context.Context, session *domains.Session) error {
			stored = session
			return nil
		},
	}
	svc := NewAuthService(&testutil.FakeUserRepository{}, sessionRepo)

	before := time.Now()
	session, err := svc.CreateSession(context.Background(), 7)
	after := time.Now()

	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID is empty")
	}
	if session.UserID != 7 {
		t.Fatalf("session UserID = %d; want 7", session.UserID)
	}
	if stored != session {
		t.Fatal("session was not passed to the repository")
	}
	if session.ExpiresAt.Sub(session.CreatedAt) != time.Hour {
		t.Fatalf("session lifetime = %v; want 1h", session.ExpiresAt.Sub(session.CreatedAt))
	}
	if session.CreatedAt.Before(before) || session.CreatedAt.After(after) {
		t.Fatalf("CreatedAt = %v; want between %v and %v", session.CreatedAt, before, after)
	}
}

func TestDeleteSession(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "success"},
		{name: "repository error", repoErr: genericErr, wantErr: genericErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deleted string
			sessionRepo := &testutil.FakeSessionRepository{
				DeleteSessionByIDFunc: func(_ context.Context, sessionID string) error {
					deleted = sessionID
					return tt.repoErr
				},
			}
			svc := NewAuthService(&testutil.FakeUserRepository{}, sessionRepo)

			err := svc.DeleteSession(context.Background(), "session-1")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DeleteSession() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("DeleteSession() error = %v; want nil", err)
			}
			if deleted != "session-1" {
				t.Fatalf("repository called with %q; want session-1", deleted)
			}
		})
	}
}
