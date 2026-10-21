package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
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
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	user, err := a.authServices.RegisterUser(r.Context(), registerReq.Username, registerReq.Email, registerReq.Password, registerReq.Country)
	if errors.Is(err, services.ErrInvalidUsername) || errors.Is(err, services.ErrInvalidEmail) || errors.Is(err, services.ErrInvalidPassword) {
		http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUsernameAlreadyExists) || errors.Is(err, repositories.ErrEmailAlreadyExists) {
		http.Error(w, "Error: "+err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Error creating the user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("register user: encode response: %v", err)
	}
}

func (a *authHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("login user: encode response: %v", err)
	}
}

func (a *authHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {

}
