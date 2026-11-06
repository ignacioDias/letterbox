package services

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	googleProvider         = "google"
	googleUserNameAttempts = 5
	googleUserInfoTimeout  = 10 * time.Second
)

var (
	ErrInvalidOAuthState = errors.New("invalid oauth state")
	ErrMissingOAuthCode  = errors.New("missing oauth authorization code")
	ErrEmailNotVerified  = errors.New("google account email is not verified")
	ErrInvalidOAuthUser  = errors.New("invalid google user data")
)

type GoogleAuthService interface {
	GetGoogleOAuth2LoginURL(state, verifier string) string
	GoogleOAuth2Callback(ctx context.Context, code, state, expectedState, verifier string) (*domains.User, error)
	CreateSession(ctx context.Context, userID int64) (*domains.Session, error)
}

type googleAuthService struct {
	userIdentityRepository repositories.UserIdentityRepository
	userRepository         repositories.UserRepository
	sessionRepository      repositories.SessionRepository
	googleOAuthConfig      *oauth2.Config
}

type GoogleUser struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	FirstName     string `json:"given_name"`
	LastName      string `json:"family_name"`
	AvatarURL     string `json:"picture"`
}

var googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

const googleOAuthCallbackPath = "/api/v1/auth/oauth2/google/callback"

func NewGoogleAuthService(userIdentityRepo repositories.UserIdentityRepository, userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) GoogleAuthService {
	return &googleAuthService{
		userIdentityRepository: userIdentityRepo,
		userRepository:         userRepo,
		sessionRepository:      sessionRepo,
		googleOAuthConfig: &oauth2.Config{
			RedirectURL:  googleOAuthRedirectURL(),                // Where Google sends the user back
			ClientID:     os.Getenv("CLIENT_ID_GOOGLE_OAUTH"),     // Get from Google Cloud Console
			ClientSecret: os.Getenv("CLIENT_SECRET_GOOGLE_OAUTH"), // Get from Google Cloud Console
			Scopes: []string{ // Permissions we're asking for
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint, // Google's OAuth2 endpoints
		},
	}
}

func googleOAuthRedirectURL() string {
	if redirectURL := os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"); redirectURL != "" {
		return redirectURL
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port + googleOAuthCallbackPath
}

func (s *googleAuthService) CreateSession(ctx context.Context, userID int64) (*domains.Session, error) {
	return createSessionForUser(ctx, s.sessionRepository, userID)
}

func (s *googleAuthService) GetGoogleOAuth2LoginURL(state, verifier string) string {
	return s.googleOAuthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (s *googleAuthService) GoogleOAuth2Callback(ctx context.Context, code, state, expectedState, verifier string) (*domains.User, error) {
	// Validate the state cookie to prevent CSRF. The PKCE verifier cookie must
	// also be present; without it the authorization code cannot be redeemed.
	if expectedState == "" || verifier == "" || subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) != 1 {
		return nil, ErrInvalidOAuthState
	}
	if code == "" {
		return nil, ErrMissingOAuthCode
	}

	token, err := s.googleOAuthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchange google oauth code: %w", err)
	}

	googleUser, err := s.getGoogleUserData(ctx, token)
	if err != nil {
		return nil, err
	}
	if googleUser.ID == "" {
		return nil, fmt.Errorf("%w: missing provider user id", ErrInvalidOAuthUser)
	}
	if googleUser.Email == "" {
		return nil, fmt.Errorf("%w: missing email", ErrInvalidOAuthUser)
	}
	if err := validateEmail(googleUser.Email); err != nil {
		return nil, err
	}
	if !googleUser.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return s.findOrCreateUser(ctx, googleUser)
}

func (s *googleAuthService) findOrCreateUser(ctx context.Context, googleUser *GoogleUser) (*domains.User, error) {
	identity, err := s.userIdentityRepository.GetByProviderAndProviderUserID(ctx, googleProvider, googleUser.ID)
	if err == nil {
		return s.userRepository.GetUserByID(ctx, identity.UserID)
	}
	if !errors.Is(err, repositories.ErrUserIdentityNotFound) {
		return nil, err
	}

	// Link a verified Google account to an existing local account with the same email.
	_, err = s.userRepository.GetUserByEmailOrUsername(ctx, googleUser.Email)
	if err == nil {
		return s.linkExistingUser(ctx, googleUser)
	}
	if !errors.Is(err, repositories.ErrUserNotFound) {
		return nil, err
	}

	// No account yet: create the user and identity together, retrying on username collisions.
	passwordHash, err := generateUnusablePasswordHash()
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < googleUserNameAttempts; attempt++ {
		user := domains.NewUser(generateUsernameFromEmail(googleUser.Email), googleUser.Email, passwordHash, googleUser.AvatarURL, "")
		identity := domains.NewUserIdentity(0, googleProvider, googleUser.ID, googleUser.Email)

		err = s.userIdentityRepository.CreateUserWithIdentity(ctx, user, identity)
		if err == nil {
			return user, nil
		}
		switch {
		case errors.Is(err, repositories.ErrUsernameAlreadyExists):
			continue
		case errors.Is(err, repositories.ErrEmailAlreadyExists):
			// Another request created the account between our lookup and insert.
			return s.linkExistingUser(ctx, googleUser)
		case errors.Is(err, repositories.ErrUserIdentityExists):
			return s.userByGoogleID(ctx, googleUser.ID)
		default:
			return nil, err
		}
	}
	return nil, repositories.ErrUsernameAlreadyExists
}

func (s *googleAuthService) linkExistingUser(ctx context.Context, googleUser *GoogleUser) (*domains.User, error) {
	user, err := s.userRepository.GetUserByEmailOrUsername(ctx, googleUser.Email)
	if err != nil {
		return nil, err
	}
	identity := domains.NewUserIdentity(user.ID, googleProvider, googleUser.ID, googleUser.Email)
	if _, err := s.userIdentityRepository.CreateIdentity(ctx, identity); err != nil {
		if errors.Is(err, repositories.ErrUserIdentityExists) {
			return s.userByGoogleID(ctx, googleUser.ID)
		}
		return nil, err
	}
	return user, nil
}

func (s *googleAuthService) userByGoogleID(ctx context.Context, providerUserID string) (*domains.User, error) {
	identity, err := s.userIdentityRepository.GetByProviderAndProviderUserID(ctx, googleProvider, providerUserID)
	if err != nil {
		return nil, err
	}
	return s.userRepository.GetUserByID(ctx, identity.UserID)
}

// getGoogleUserData fetches user profile information from Google's UserInfo API
// using an authenticated client, so the access token is never placed in the URL.
func (s *googleAuthService) getGoogleUserData(ctx context.Context, token *oauth2.Token) (*GoogleUser, error) {
	client := s.googleOAuthConfig.Client(ctx, token)
	client.Timeout = googleUserInfoTimeout

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create google user info request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch google user info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google user info API returned status %d", response.StatusCode)
	}

	var googleUser GoogleUser
	if err := json.NewDecoder(response.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("decode google user info: %w", err)
	}
	return &googleUser, nil
}
