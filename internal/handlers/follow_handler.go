package handlers

import (
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
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
	GetFollowersCountByUsername(w http.ResponseWriter, r *http.Request)
	GetFollowingCountByUsername(w http.ResponseWriter, r *http.Request)
}

type followHandler struct {
	followServices services.FollowService
}

var followLookupErrors = []errorMapping{
	invalidUsernameError,
	userNotFoundError,
}

var followUserErrors = []errorMapping{
	invalidUsernameError,
	userNotFoundError,
	{err: repositories.ErrAlreadyFollowing, status: http.StatusConflict, message: "Already following this user"},
	{err: services.ErrCannotFollowSelf, status: http.StatusBadRequest, message: "Cannot follow yourself"},
	{err: repositories.ErrUserBlocked, status: http.StatusForbidden, message: "Cannot follow this user"},
}

var unfollowUserErrors = []errorMapping{
	invalidUsernameError,
	userNotFoundError,
	{err: services.ErrCannotUnfollowSelf, status: http.StatusBadRequest, message: "Cannot unfollow yourself"},
	{err: repositories.ErrNotFollowing, status: http.StatusConflict, message: "Not following this user"},
}

func NewFollowHandler(followRepository repositories.FollowRepository, userRepository repositories.UserRepository, blockRepository repositories.BlockRepository) FollowHandler {
	return &followHandler{
		followServices: services.NewFollowService(followRepository, userRepository, blockRepository),
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
	if err != nil {
		writeError(w, err, "get followers by username", "Error getting followers", followLookupErrors...)
		return
	}
	writeJSON(w, http.StatusOK, followers)
}

func (h *followHandler) GetFollowingByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	following, err := h.followServices.GetFollowingByUsername(r.Context(), username, limit, offset)
	if err != nil {
		writeError(w, err, "get following by username", "Error getting following", followLookupErrors...)
		return
	}
	writeJSON(w, http.StatusOK, following)
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
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	username := r.PathValue("username")
	err := h.followServices.FollowUserByUsername(r.Context(), userID, username)
	if err != nil {
		writeError(w, err, "follow user by username", "Error following user", followUserErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *followHandler) UnfollowUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	username := r.PathValue("username")
	err := h.followServices.UnfollowUserByUsername(r.Context(), userID, username)
	if err != nil {
		writeError(w, err, "unfollow user by username", "Error unfollowing user", unfollowUserErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *followHandler) GetFollowersCountByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	count, err := h.followServices.GetFollowersCountByUsername(r.Context(), username)
	if err != nil {
		writeError(w, err, "get followers count by username", "Error getting followers count", followLookupErrors...)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *followHandler) GetFollowingCountByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	count, err := h.followServices.GetFollowingCountByUsername(r.Context(), username)
	if err != nil {
		writeError(w, err, "get following count by username", "Error getting following count", followLookupErrors...)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}
