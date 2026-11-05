package handlers

import (
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"net/http"
)

type GoogleOAuth2Handler interface {
	GoogleOAuth2Login(w http.ResponseWriter, r *http.Request)
	GoogleOAuth2Callback(w http.ResponseWriter, r *http.Request)
}

type googleOAuth2Handler struct {
	googleAuthService services.GoogleAuthService
}

func NewGoogleOAuth2Handler(userIdentityRepo repositories.UserIdentityRepository, userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) GoogleOAuth2Handler {
	return &googleOAuth2Handler{
		googleAuthService: services.NewGoogleAuthService(userIdentityRepo, userRepo, sessionRepo),
	}
}

func (h *googleOAuth2Handler) GoogleOAuth2Login(w http.ResponseWriter, r *http.Request) {

}

func (h *googleOAuth2Handler) GoogleOAuth2Callback(w http.ResponseWriter, r *http.Request) {}
