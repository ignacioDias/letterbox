package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"net/http"
)

type AuthHandler interface {
	RegisterUser(w http.ResponseWriter, r *http.Request)
	LoginUser(w http.ResponseWriter, r *http.Request)
	LogoutUser(w http.ResponseWriter, r *http.Request)
	GetCurrentUser(w http.ResponseWriter, r *http.Request)
}

type authHandler struct {
	authServices services.AuthService
}

type RegisterUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Country  string `json:"country"`
}

func NewAuthHandler(authRepo repositories.AuthRepository) AuthHandler {
	return &authHandler{
		authServices: services.NewAuthService(authRepo),
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (a *authHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	// Implement the logic for user login here
}

func (a *authHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	// Implement the logic for user logout here
}

func (a *authHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get the current authenticated user here
}
