package handlers

import (
	"errors"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type BlockHandler interface {
	BlockUserByUsername(w http.ResponseWriter, r *http.Request)
	UnblockUserByUsername(w http.ResponseWriter, r *http.Request)
}

type blockHandler struct {
	blockServices services.BlockService
}

func NewBlockHandler(blockRepository repositories.BlockRepository, userRepository repositories.UserRepository) BlockHandler {
	return &blockHandler{
		blockServices: services.NewBlockService(blockRepository, userRepository),
	}
}

func (h *blockHandler) BlockUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username := r.PathValue("username")
	err := h.blockServices.BlockUserByUsername(r.Context(), userID, username)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrCannotBlockSelf) {
		http.Error(w, "Cannot block yourself", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrAlreadyBlocked) {
		http.Error(w, "Already blocking this user", http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("block user by username: %v", err)
		http.Error(w, "Error blocking user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *blockHandler) UnblockUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username := r.PathValue("username")
	err := h.blockServices.UnblockUserByUsername(r.Context(), userID, username)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, repositories.ErrNotBlocked) {
		http.Error(w, "Not blocking this user", http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("unblock user by username: %v", err)
		http.Error(w, "Error unblocking user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
