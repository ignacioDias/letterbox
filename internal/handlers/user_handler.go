package handlers

import (
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
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

var updateUserErrors = []errorMapping{
	{err: services.ErrInvalidUsername, status: http.StatusBadRequest},
	{err: services.ErrInvalidEmail, status: http.StatusBadRequest},
	{err: services.ErrInvalidCountry, status: http.StatusBadRequest},
	{err: repositories.ErrUsernameAlreadyExists, status: http.StatusConflict},
	{err: repositories.ErrEmailAlreadyExists, status: http.StatusConflict},
	userNotFoundError,
}

var updatePasswordErrors = []errorMapping{
	{err: services.ErrInvalidNewPassword, status: http.StatusBadRequest, message: "Invalid new password"},
	{err: services.ErrInvalidPassword, status: http.StatusUnauthorized, message: "Invalid current password"},
	userNotFoundError,
}

func NewUserHandler(userRepository repositories.UserRepository, sessionRepository repositories.SessionRepository) UserHandler {
	return &userHandler{
		userServices: services.NewUserService(userRepository, sessionRepository),
		secureCookie: os.Getenv("APP_ENV") == "production",
	}
}

func (h *userHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	user, err := h.userServices.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, err, "get current user", "Error getting user", userNotFoundError)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *userHandler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	user, err := h.userServices.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeError(w, err, "get user by username", "Error getting user", invalidUsernameError, userNotFoundError)
		return
	}
	writeJSON(w, http.StatusOK, domains.PublicUser{
		ID:                user.ID,
		Username:          user.Username,
		ProfilePictureURL: user.ProfilePictureURL,
		Country:           user.Country,
	})
}

func (h *userHandler) DeleteCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if err := h.userServices.DeleteUserByID(r.Context(), userID); err != nil {
		writeError(w, err, "delete current user", "Error deleting user", userNotFoundError)
		return
	}
	clearSessionCookie(w, h.secureCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *userHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var updateReq UpdateUserRequest
	if !decodeJSON(w, r, &updateReq) {
		return
	}
	if err := h.userServices.UpdateUserByID(r.Context(), userID, updateReq.Username, updateReq.Email, updateReq.Country, updateReq.ProfilePicture); err != nil {
		writeError(w, err, "update current user", "Error updating user", updateUserErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *userHandler) UpdateCurrentUserPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	sessionID, ok := requireSessionID(w, r)
	if !ok {
		return
	}
	var updatePasswordReq UpdateUserPasswordRequest
	if !decodeJSON(w, r, &updatePasswordReq) {
		return
	}
	err := h.userServices.UpdateUserPasswordByID(r.Context(), userID, sessionID, updatePasswordReq.CurrentPassword, updatePasswordReq.NewPassword)
	if err != nil {
		writeError(w, err, "update current user password", "Error updating user password", updatePasswordErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
