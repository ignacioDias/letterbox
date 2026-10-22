package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
	"strconv"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type FollowHandler interface {
	GetFollowersByUsername(w http.ResponseWriter, r *http.Request)
	GetFollowingByUsername(w http.ResponseWriter, r *http.Request)
	FollowUserByUsername(w http.ResponseWriter, r *http.Request)
	UnfollowUserByUsername(w http.ResponseWriter, r *http.Request)
}

type followHandler struct {
	followServices services.FollowService
}

func NewFollowHandler(followRepository repositories.FollowRepository) FollowHandler {
	return &followHandler{
		followServices: services.NewFollowService(followRepository),
	}
}

func (h *followHandler) GetFollowersByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	followers, err := h.followServices.GetFollowersByUsername(r.Context(), username, limit, offset)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get followers by username: %v", err)
		http.Error(w, "Error getting followers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(followers); err != nil {
		log.Printf("get followers by username: encode response: %v", err)
	}
}

func (h *followHandler) GetFollowingByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	following, err := h.followServices.GetFollowingByUsername(r.Context(), username, limit, offset)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get following by username: %v", err)
		http.Error(w, "Error getting following", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(following); err != nil {
		log.Printf("get following by username: encode response: %v", err)
	}
}

func parsePagination(r *http.Request) (int, int, error) {
	limit := defaultListLimit
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxListLimit {
			return 0, 0, errors.New("invalid limit")
		}
		limit = value
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, errors.New("invalid offset")
		}
		offset = value
	}
	return limit, offset, nil
}

func (h *followHandler) FollowUserByUsername(w http.ResponseWriter, r *http.Request) {
}

func (h *followHandler) UnfollowUserByUsername(w http.ResponseWriter, r *http.Request) {}
