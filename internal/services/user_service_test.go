package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/cache"
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

func TestSearchUsers(t *testing.T) {
	genericErr := errors.New("database unavailable")
	users := []domains.PublicUser{{ID: 7, Username: "ada"}}

	tests := []struct {
		name       string
		search     string
		limit      int
		offset     int
		users      []domains.PublicUser
		repoErr    error
		wantErr    error
		wantCalled bool
		wantSearch string
		wantLimit  int
		wantOffset int
	}{
		{name: "success trims search", search: "  ada  ", limit: 10, offset: 5, users: users, wantCalled: true, wantSearch: "ada", wantLimit: 10, wantOffset: 5},
		{name: "empty search", search: "", limit: 10, wantErr: ErrInvalidSearchQuery},
		{name: "whitespace only search", search: "   ", limit: 10, wantErr: ErrInvalidSearchQuery},
		{name: "repository error", search: "ada", limit: 10, repoErr: genericErr, wantErr: genericErr, wantCalled: true, wantSearch: "ada", wantLimit: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSearch string
			var gotLimit, gotOffset int
			userRepo := &testutil.FakeUserRepository{
				SearchUsersFunc: func(_ context.Context, search string, limit int, offset int) ([]domains.PublicUser, error) {
					gotSearch = search
					gotLimit = limit
					gotOffset = offset
					return tt.users, tt.repoErr
				},
			}
			svc := NewUserService(userRepo, &testutil.FakeSessionRepository{})

			got, err := svc.SearchUsers(context.Background(), tt.search, tt.limit, tt.offset)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SearchUsers() error = %v; want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("SearchUsers() error = %v; want nil", err)
				}
				if len(got) != len(tt.users) {
					t.Fatalf("SearchUsers() returned %d users; want %d", len(got), len(tt.users))
				}
				for i := range got {
					if got[i] != tt.users[i] {
						t.Fatalf("SearchUsers() user %d = %+v; want %+v", i, got[i], tt.users[i])
					}
				}
			}

			if !tt.wantCalled {
				return
			}
			if gotSearch != tt.wantSearch || gotLimit != tt.wantLimit || gotOffset != tt.wantOffset {
				t.Fatalf("repository called with search %q, limit %d, offset %d; want %q, %d, %d",
					gotSearch, gotLimit, gotOffset, tt.wantSearch, tt.wantLimit, tt.wantOffset)
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
			name:    "invalid country",
			userID:  7,
			country: "ARG",
			wantErr: ErrInvalidCountry,
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

func TestGetUserByIDCacheHit(t *testing.T) {
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com"}
	value, err := generateCacheValueFromUser(user)
	if err != nil {
		t.Fatalf("encode cached user: %v", err)
	}
	repoCalled := false
	fakeCache := &testutil.FakeCache{
		GetFunc: func(_ context.Context, key string) (string, error) {
			if key != getKeyForUserByID(7) {
				t.Fatalf("cache.Get(%q); want %q", key, getKeyForUserByID(7))
			}
			return value, nil
		},
		SetFunc: func(context.Context, string, string, int) error {
			return nil
		},
		DeleteFunc: func(context.Context, []string) (int64, error) {
			return 0, nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
			repoCalled = true
			return nil, errors.New("repository should not be called")
		},
	}
	svc := NewUserService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	got, err := svc.GetUserByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if repoCalled {
		t.Fatal("GetUserByID called repository on cache hit")
	}
	if got.Username != "ada" {
		t.Fatalf("GetUserByID returned %+v", got)
	}
}

func TestGetUserByUsernameCacheMissCachesResult(t *testing.T) {
	user := &domains.User{ID: 7, Username: "ada"}
	wantKey := getKeyForUserByUsername("ada")
	var setKey string
	var setTTL int
	fakeCache := &testutil.FakeCache{
		GetFunc: func(context.Context, string) (string, error) {
			return "", cache.ErrKeyNotFound
		},
		SetFunc: func(_ context.Context, key string, _ string, expiringTimeInSeconds int) error {
			setKey = key
			setTTL = expiringTimeInSeconds
			return nil
		},
		DeleteFunc: func(context.Context, []string) (int64, error) {
			return 0, nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByUsernameFunc: func(_ context.Context, username string) (*domains.User, error) {
			if username != "ada" {
				t.Fatalf("GetUserByUsername(%q); want ada", username)
			}
			return user, nil
		},
	}
	svc := NewUserService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	got, err := svc.GetUserByUsername(context.Background(), "ada")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.Username != "ada" {
		t.Fatalf("GetUserByUsername returned %+v", got)
	}
	if setKey != wantKey {
		t.Fatalf("cache.Set key = %q; want %q", setKey, wantKey)
	}
	if setTTL != userCacheTTLSeconds {
		t.Fatalf("cache.Set TTL = %d; want %d", setTTL, userCacheTTLSeconds)
	}
}

func TestSearchUsersCacheHit(t *testing.T) {
	users := []domains.PublicUser{{ID: 7, Username: "ada"}}
	value, err := generateCacheValueFromPublicUserList(users)
	if err != nil {
		t.Fatalf("encode cached users: %v", err)
	}
	generation := "generation-1"
	generationKey := getKeyForUserSearchGeneration()
	listKey := getKeyForUserSearch("ada", 20, 5, generation)
	repoCalled := false
	fakeCache := &testutil.FakeCache{
		GetFunc: func(_ context.Context, key string) (string, error) {
			switch key {
			case generationKey:
				return generation, nil
			case listKey:
				return value, nil
			default:
				return "", cache.ErrKeyNotFound
			}
		},
		SetFunc: func(context.Context, string, string, int) error {
			return nil
		},
		DeleteFunc: func(context.Context, []string) (int64, error) {
			return 0, nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		SearchUsersFunc: func(context.Context, string, int, int) ([]domains.PublicUser, error) {
			repoCalled = true
			return nil, errors.New("repository should not be called")
		},
	}
	svc := NewUserService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	got, err := svc.SearchUsers(context.Background(), "ada", 20, 5)
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if repoCalled {
		t.Fatal("SearchUsers called repository on cache hit")
	}
	if len(got) != 1 || got[0].Username != "ada" {
		t.Fatalf("SearchUsers returned %+v", got)
	}
}

func TestUpdateUserByIDInvalidatesCache(t *testing.T) {
	var deletedKeys []string
	searchGenerationReset := false
	fakeCache := &testutil.FakeCache{
		GetFunc: func(context.Context, string) (string, error) {
			return "", cache.ErrKeyNotFound
		},
		SetFunc: func(_ context.Context, key string, _ string, expiringTimeInSeconds int) error {
			if expiringTimeInSeconds != userCacheTTLSeconds {
				t.Fatalf("cache.Set TTL = %d; want %d", expiringTimeInSeconds, userCacheTTLSeconds)
			}
			if key == getKeyForUserSearchGeneration() {
				searchGenerationReset = true
			}
			return nil
		},
		DeleteFunc: func(_ context.Context, keys []string) (int64, error) {
			deletedKeys = append(deletedKeys, keys...)
			return int64(len(keys)), nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
			return &domains.User{ID: 7, Username: "ada", Email: "ada@example.com"}, nil
		},
		UpdateUserFunc: func(context.Context, *domains.User) error {
			return nil
		},
	}
	svc := NewUserService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	if err := svc.UpdateUserByID(context.Background(), 7, "ada_l", "", "", ""); err != nil {
		t.Fatalf("UpdateUserByID: %v", err)
	}

	wantDeleted := map[string]bool{
		getKeyForUserByID(7):             false,
		getKeyForUserByUsername("ada"):   false,
		getKeyForUserByUsername("ada_l"): false,
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
	if !searchGenerationReset {
		t.Fatal("search cache generation was not reset")
	}
}

func TestDeleteUserByIDInvalidatesCache(t *testing.T) {
	var deletedKeys []string
	searchGenerationReset := false
	fakeCache := &testutil.FakeCache{
		GetFunc: func(context.Context, string) (string, error) {
			return "", cache.ErrKeyNotFound
		},
		SetFunc: func(_ context.Context, key string, _ string, _ int) error {
			if key == getKeyForUserSearchGeneration() {
				searchGenerationReset = true
			}
			return nil
		},
		DeleteFunc: func(_ context.Context, keys []string) (int64, error) {
			deletedKeys = append(deletedKeys, keys...)
			return int64(len(keys)), nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
			return &domains.User{ID: 7, Username: "ada"}, nil
		},
		DeleteUserByIDFunc: func(context.Context, int64) error {
			return nil
		},
	}
	svc := NewUserService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	if err := svc.DeleteUserByID(context.Background(), 7); err != nil {
		t.Fatalf("DeleteUserByID: %v", err)
	}

	foundIDKey := false
	foundUsernameKey := false
	for _, key := range deletedKeys {
		if key == getKeyForUserByID(7) {
			foundIDKey = true
		}
		if key == getKeyForUserByUsername("ada") {
			foundUsernameKey = true
		}
	}
	if !foundIDKey || !foundUsernameKey {
		t.Fatalf("cache.Delete keys = %v; want id and username keys", deletedKeys)
	}
	if !searchGenerationReset {
		t.Fatal("search cache generation was not reset")
	}
}

func TestRegisterUserResetsSearchCacheGeneration(t *testing.T) {
	searchGenerationReset := false
	fakeCache := &testutil.FakeCache{
		GetFunc: func(context.Context, string) (string, error) {
			return "", cache.ErrKeyNotFound
		},
		SetFunc: func(_ context.Context, key string, _ string, _ int) error {
			if key == getKeyForUserSearchGeneration() {
				searchGenerationReset = true
			}
			return nil
		},
		DeleteFunc: func(context.Context, []string) (int64, error) {
			return 0, nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		CreateUserFunc: func(_ context.Context, user *domains.User) (int64, error) {
			return 7, nil
		},
	}
	svc := NewAuthService(userRepo, &testutil.FakeSessionRepository{}, fakeCache)

	if _, err := svc.RegisterUser(context.Background(), "ada", "ada@example.com", "secret123", "", "AR"); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	if !searchGenerationReset {
		t.Fatal("search cache generation was not reset")
	}
}
