package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	"golang.org/x/oauth2"
)

func newGoogleTestServer(t *testing.T, userInfo map[string]any) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
			t.Errorf("Authorization header = %q; want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	previousURL := googleUserInfoURL
	googleUserInfoURL = server.URL + "/userinfo"
	t.Cleanup(func() { googleUserInfoURL = previousURL })

	return server
}

func newGoogleServiceForTest(server *httptest.Server, identityRepo repositories.UserIdentityRepository, userRepo repositories.UserRepository) *googleAuthService {
	return &googleAuthService{
		userIdentityRepository: identityRepo,
		userRepository:         userRepo,
		sessionRepository:      &testutil.FakeSessionRepository{},
		googleOAuthConfig: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost/callback",
			Scopes:       []string{"openid"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  server.URL + "/auth",
				TokenURL: server.URL + "/token",
			},
		},
	}
}

func googleUserInfo(verified bool) map[string]any {
	return map[string]any{
		"sub":            "google-user-id",
		"email":          "ada@example.com",
		"email_verified": verified,
		"given_name":     "Ada",
		"family_name":    "Lovelace",
		"picture":        "https://example.com/ada.jpg",
	}
}

func TestGoogleOAuth2CallbackCreatesUserAndIdentity(t *testing.T) {
	server := newGoogleTestServer(t, googleUserInfo(true))

	var createdUser *domains.User
	var createdIdentity *domains.UserIdentity
	identityRepo := &testutil.FakeUserIdentityRepository{
		GetByProviderAndProviderUserIDFunc: func(context.Context, string, string) (*domains.UserIdentity, error) {
			return nil, repositories.ErrUserIdentityNotFound
		},
		CreateUserWithIdentityFunc: func(_ context.Context, user *domains.User, identity *domains.UserIdentity) error {
			user.ID = 42
			identity.UserID = 42
			createdUser = user
			createdIdentity = identity
			return nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByEmailOrUsernameFunc: func(context.Context, string) (*domains.User, error) {
			return nil, repositories.ErrUserNotFound
		},
	}
	svc := newGoogleServiceForTest(server, identityRepo, userRepo)

	verifier := oauth2.GenerateVerifier()
	user, err := svc.GoogleOAuth2Callback(context.Background(), "code", "state", "state", verifier)
	if err != nil {
		t.Fatalf("GoogleOAuth2Callback() error = %v", err)
	}

	if user.ID != 42 || user.Email != "ada@example.com" || user.ProfilePictureURL != "https://example.com/ada.jpg" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if err := validateUsername(user.Username); err != nil {
		t.Fatalf("generated username %q is invalid: %v", user.Username, err)
	}
	if user.PasswordHash == "" {
		t.Fatal("expected a non-empty password hash for an OAuth-only account")
	}
	if checkPassword(user.PasswordHash, "") == nil {
		t.Fatal("OAuth-only account should not accept an empty password")
	}
	if createdUser == nil || createdIdentity == nil {
		t.Fatal("expected the user and identity to be created")
	}
	if createdIdentity.Provider != googleProvider || createdIdentity.ProviderUserID != "google-user-id" {
		t.Fatalf("unexpected identity: %+v", createdIdentity)
	}
	if createdIdentity.UserID != 42 {
		t.Fatalf("identity user id = %d; want 42", createdIdentity.UserID)
	}
}

func TestGoogleOAuth2CallbackUsesExistingIdentity(t *testing.T) {
	server := newGoogleTestServer(t, googleUserInfo(true))
	expected := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com"}

	identityRepo := &testutil.FakeUserIdentityRepository{
		GetByProviderAndProviderUserIDFunc: func(context.Context, string, string) (*domains.UserIdentity, error) {
			return &domains.UserIdentity{UserID: 7}, nil
		},
		CreateUserWithIdentityFunc: func(context.Context, *domains.User, *domains.UserIdentity) error {
			t.Fatal("should not create a user when the identity already exists")
			return nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByIDFunc: func(_ context.Context, userID int64) (*domains.User, error) {
			if userID != 7 {
				t.Fatalf("GetUserByID(%d); want 7", userID)
			}
			return expected, nil
		},
	}
	svc := newGoogleServiceForTest(server, identityRepo, userRepo)

	user, err := svc.GoogleOAuth2Callback(context.Background(), "code", "state", "state", oauth2.GenerateVerifier())
	if err != nil {
		t.Fatalf("GoogleOAuth2Callback() error = %v", err)
	}
	if user != expected {
		t.Fatalf("user = %+v; want %+v", user, expected)
	}
}

func TestGoogleOAuth2CallbackLinksExistingAccountByEmail(t *testing.T) {
	server := newGoogleTestServer(t, googleUserInfo(true))
	existing := &domains.User{ID: 5, Username: "ada", Email: "ada@example.com"}

	var linked *domains.UserIdentity
	identityRepo := &testutil.FakeUserIdentityRepository{
		GetByProviderAndProviderUserIDFunc: func(context.Context, string, string) (*domains.UserIdentity, error) {
			return nil, repositories.ErrUserIdentityNotFound
		},
		CreateIdentityFunc: func(_ context.Context, identity *domains.UserIdentity) (*domains.UserIdentity, error) {
			linked = identity
			return identity, nil
		},
		CreateUserWithIdentityFunc: func(context.Context, *domains.User, *domains.UserIdentity) error {
			t.Fatal("should link the existing account instead of creating a new user")
			return nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByEmailOrUsernameFunc: func(context.Context, string) (*domains.User, error) {
			return existing, nil
		},
	}
	svc := newGoogleServiceForTest(server, identityRepo, userRepo)

	user, err := svc.GoogleOAuth2Callback(context.Background(), "code", "state", "state", oauth2.GenerateVerifier())
	if err != nil {
		t.Fatalf("GoogleOAuth2Callback() error = %v", err)
	}
	if user != existing {
		t.Fatalf("user = %+v; want %+v", user, existing)
	}
	if linked == nil || linked.UserID != 5 || linked.ProviderUserID != "google-user-id" {
		t.Fatalf("unexpected linked identity: %+v", linked)
	}
}

func TestGoogleOAuth2CallbackRejectsUnverifiedEmail(t *testing.T) {
	server := newGoogleTestServer(t, googleUserInfo(false))
	svc := newGoogleServiceForTest(server, &testutil.FakeUserIdentityRepository{}, &testutil.FakeUserRepository{})

	_, err := svc.GoogleOAuth2Callback(context.Background(), "code", "state", "state", oauth2.GenerateVerifier())
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("error = %v; want %v", err, ErrEmailNotVerified)
	}
}

func TestGoogleOAuth2CallbackValidatesStateAndVerifier(t *testing.T) {
	tests := []struct {
		name          string
		state         string
		expectedState string
		verifier      string
		wantErr       error
	}{
		{name: "mismatched state", state: "other", expectedState: "state", verifier: "verifier", wantErr: ErrInvalidOAuthState},
		{name: "missing state", state: "", expectedState: "", verifier: "verifier", wantErr: ErrInvalidOAuthState},
		{name: "missing verifier", state: "state", expectedState: "state", verifier: "", wantErr: ErrInvalidOAuthState},
		{name: "missing code", state: "state", expectedState: "state", verifier: "verifier", wantErr: ErrMissingOAuthCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &googleAuthService{googleOAuthConfig: &oauth2.Config{}}
			_, err := svc.GoogleOAuth2Callback(context.Background(), "", tt.state, tt.expectedState, tt.verifier)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoogleOAuth2CallbackRetriesUsernameCollisions(t *testing.T) {
	server := newGoogleTestServer(t, googleUserInfo(true))
	attempts := 0
	identityRepo := &testutil.FakeUserIdentityRepository{
		GetByProviderAndProviderUserIDFunc: func(context.Context, string, string) (*domains.UserIdentity, error) {
			return nil, repositories.ErrUserIdentityNotFound
		},
		CreateUserWithIdentityFunc: func(_ context.Context, user *domains.User, _ *domains.UserIdentity) error {
			attempts++
			if attempts < 3 {
				return repositories.ErrUsernameAlreadyExists
			}
			user.ID = 99
			return nil
		},
	}
	userRepo := &testutil.FakeUserRepository{
		GetUserByEmailOrUsernameFunc: func(context.Context, string) (*domains.User, error) {
			return nil, repositories.ErrUserNotFound
		},
	}
	svc := newGoogleServiceForTest(server, identityRepo, userRepo)

	user, err := svc.GoogleOAuth2Callback(context.Background(), "code", "state", "state", oauth2.GenerateVerifier())
	if err != nil {
		t.Fatalf("GoogleOAuth2Callback() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d; want 3", attempts)
	}
	if user.ID != 99 {
		t.Fatalf("user ID = %d; want 99", user.ID)
	}
}

func TestGenerateUsernameFromEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "plain", email: "ada@example.com"},
		{name: "dots", email: "ada.lovelace@example.com"},
		{name: "plus", email: "ada+test@example.com"},
		{name: "long", email: "a-very-long-local-part-that-exceeds-limits@example.com"},
		{name: "empty local part", email: "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateUsername(generateUsernameFromEmail(tt.email)); err != nil {
				t.Fatalf("generateUsernameFromEmail(%q) = %q is invalid: %v", tt.email, generateUsernameFromEmail(tt.email), err)
			}
		})
	}
}
