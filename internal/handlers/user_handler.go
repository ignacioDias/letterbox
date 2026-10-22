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

type UserHandler interface {
	GetCurrentUser(w http.ResponseWriter, r *http.Request)
	GetUserByUsername(w http.ResponseWriter, r *http.Request)
	DeleteCurrentUser(w http.ResponseWriter, r *http.Request)
	UpdateCurrentUser(w http.ResponseWriter, r *http.Request)
	UpdateCurrentUserPassword(w http.ResponseWriter, r *http.Request)
}

type userHandler struct {
	userServices services.UserService
	secureCookie bool
}

type UpdateUserRequest struct {
	Username       string `json:"username"`
	Email          string `json:"email"`
	Country        string `json:"country"`
	ProfilePicture string `json:"profilePicture"`
}

type UpdateUserPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func NewUserHandler(userRepository repositories.UserRepository, sessionRepository repositories.SessionRepository) UserHandler {
	return &userHandler{
		userServices: services.NewUserService(userRepository, sessionRepository),
		secureCookie: os.Getenv("APP_ENV") == "production",
	}
}

func (h *userHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	user, err := h.userServices.GetUserByID(r.Context(), userID)
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Error getting user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("get current user: encode response: %v", err)
	}
}

func (h *userHandler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	user, err := h.userServices.GetUserByUsername(r.Context(), username)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get user by username: %v", err)
		http.Error(w, "Error getting user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("get user by username: encode response: %v", err)
	}
}

func (h *userHandler) DeleteCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	err := h.userServices.DeleteUserByID(r.Context(), userID)
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("delete current user: %v", err)
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}
	clearSessionCookie(w, h.secureCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *userHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	var updateReq UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if err := h.userServices.UpdateUserByID(r.Context(), userID, updateReq.Username, updateReq.Email, updateReq.Country, updateReq.ProfilePicture); err != nil {
		if errors.Is(err, services.ErrInvalidUsername) || errors.Is(err, services.ErrInvalidEmail) {
			http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, repositories.ErrUsernameAlreadyExists) || errors.Is(err, repositories.ErrEmailAlreadyExists) {
			http.Error(w, "Error: "+err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, repositories.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("update current user: %v", err)
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *userHandler) UpdateCurrentUserPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var updatePasswordReq UpdateUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&updatePasswordReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	err = h.userServices.UpdateUserPasswordByID(r.Context(), userID, cookie.Value, updatePasswordReq.CurrentPassword, updatePasswordReq.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrInvalidNewPassword) {
			http.Error(w, "Error: invalid new password format", http.StatusBadRequest)
			return
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			http.Error(w, "Error with the current password", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, repositories.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("update current user password: %v", err)
		http.Error(w, "Error updating user password", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
