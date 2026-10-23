package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"net/http"
	"os"
)

type AuthHandler interface {
	RegisterUser(w http.ResponseWriter, r *http.Request)
	LoginUser(w http.ResponseWriter, r *http.Request)
	LogoutUser(w http.ResponseWriter, r *http.Request)
}

type authHandler struct {
	authServices services.AuthService
	secureCookie bool
}

type RegisterUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Country  string `json:"country"`
}

type LoginUserRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	Password        string `json:"password"`
}

func NewAuthHandler(userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) AuthHandler {
	return &authHandler{
		authServices: services.NewAuthService(userRepo, sessionRepo),
		secureCookie: os.Getenv("APP_ENV") == "production",
	}
}

func (a *authHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerReq RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user, err := a.authServices.RegisterUser(r.Context(), registerReq.Username, registerReq.Email, registerReq.Password, registerReq.Country)
	if errors.Is(err, services.ErrInvalidUsername) || errors.Is(err, services.ErrInvalidEmail) || errors.Is(err, services.ErrInvalidPassword) || errors.Is(err, services.ErrInvalidCountry) {
		http.Error(w, "Invalid user data", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUsernameAlreadyExists) || errors.Is(err, repositories.ErrEmailAlreadyExists) {
		http.Error(w, "Username or email already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Error creating the user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a *authHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user, err := a.authServices.LoginUser(r.Context(), loginReq.EmailOrUsername, loginReq.Password)
	if errors.Is(err, services.ErrInvalidCredentials) || errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Error logging in the user", http.StatusInternalServerError)
		return
	}
	session, err := a.authServices.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Error creating the session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})
	writeJSON(w, http.StatusOK, user)
}

func (a *authHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := requireSessionID(w, r)
	if !ok {
		return
	}
	if err := a.authServices.DeleteSession(r.Context(), sessionID); err != nil {
		http.Error(w, "Error logging out the user", http.StatusInternalServerError)
		return
	}
	clearSessionCookie(w, a.secureCookie)
	w.WriteHeader(http.StatusOK)
}
