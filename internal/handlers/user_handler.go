package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type UserHandler interface {
	GetCurrentUser(w http.ResponseWriter, r *http.Request)
}

type userHandler struct {
	userServices services.UserService
}

func NewUserHandler(userRepository repositories.UserRepository) UserHandler {
	return &userHandler{
		userServices: services.NewUserService(userRepository),
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
