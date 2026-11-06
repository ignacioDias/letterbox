package handlers

import (
	"crypto/rand"
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
)

const (
	oauthStateCookieName    = "oauthstate"
	oauthVerifierCookieName = "oauthcodeverifier"
	oauthCookieTTL          = 10 * time.Minute
)

type GoogleOAuth2Handler interface {
	GoogleOAuth2Login(w http.ResponseWriter, r *http.Request)
	GoogleOAuth2Callback(w http.ResponseWriter, r *http.Request)
}

type googleOAuth2Handler struct {
	googleAuthService services.GoogleAuthService
	secureCookie      bool
}

func NewGoogleOAuth2Handler(userIdentityRepo repositories.UserIdentityRepository, userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) GoogleOAuth2Handler {
	return &googleOAuth2Handler{
		googleAuthService: services.NewGoogleAuthService(userIdentityRepo, userRepo, sessionRepo),
		secureCookie:      os.Getenv("APP_ENV") == "production",
	}
}

func (h *googleOAuth2Handler) GoogleOAuth2Login(w http.ResponseWriter, r *http.Request) {
	// The state prevents CSRF and the verifier is required for PKCE.
	state := rand.Text()
	verifier := oauth2.GenerateVerifier()
	h.setOAuthCookie(w, oauthStateCookieName, state)
	h.setOAuthCookie(w, oauthVerifierCookieName, verifier)

	url := h.googleAuthService.GetGoogleOAuth2LoginURL(state, verifier)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *googleOAuth2Handler) GoogleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, stateErr := r.Cookie(oauthStateCookieName)
	verifierCookie, verifierErr := r.Cookie(oauthVerifierCookieName)
	// These are single-use values; clear them even if the callback fails.
	h.clearOAuthCookie(w, oauthStateCookieName)
	h.clearOAuthCookie(w, oauthVerifierCookieName)

	if stateErr != nil || verifierErr != nil {
		log.Printf("OAuth cookies not found for Google callback: state=%v verifier=%v", stateErr, verifierErr)
		http.Error(w, "OAuth state cookie not found", http.StatusBadRequest)
		return
	}
	if oauthError := r.FormValue("error"); oauthError != "" {
		log.Printf("Google OAuth authorization denied: %s", oauthError)
		http.Error(w, "Google authorization was not granted", http.StatusBadRequest)
		return
	}

	user, err := h.googleAuthService.GoogleOAuth2Callback(
		r.Context(),
		r.FormValue("code"),
		r.FormValue("state"),
		stateCookie.Value,
		verifierCookie.Value,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidOAuthState),
			errors.Is(err, services.ErrMissingOAuthCode),
			errors.Is(err, services.ErrEmailNotVerified),
			errors.Is(err, services.ErrInvalidOAuthUser),
			errors.Is(err, services.ErrInvalidEmail):
			http.Error(w, "Invalid Google OAuth callback", http.StatusBadRequest)
		case errors.Is(err, repositories.ErrEmailAlreadyExists),
			errors.Is(err, repositories.ErrUsernameAlreadyExists),
			errors.Is(err, repositories.ErrUserIdentityExists),
			errors.Is(err, repositories.ErrUserIdentityNotFound):
			http.Error(w, "Account already exists", http.StatusConflict)
		default:
			log.Printf("Error while making the google callback %v", err)
			http.Error(w, "Error processing Google OAuth callback", http.StatusInternalServerError)
		}
		return
	}

	session, err := h.googleAuthService.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Error creating the session", http.StatusInternalServerError)
		return
	}
	generateSession(session, w, h.secureCookie)
	writeJSON(w, http.StatusOK, user)
}

func (h *googleOAuth2Handler) setOAuthCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(oauthCookieTTL),
		MaxAge:   int(oauthCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *googleOAuth2Handler) clearOAuthCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
