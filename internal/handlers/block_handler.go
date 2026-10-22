package handlers

import (
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"net/http"
)

type BlockHandler interface {
	BlockUserByUsername(w http.ResponseWriter, r *http.Request)
	UnblockUserByUsername(w http.ResponseWriter, r *http.Request)
}

type blockHandler struct {
	blockServices services.BlockService
}

var blockUserErrors = []errorMapping{
	invalidUsernameError,
	userNotFoundError,
	{err: services.ErrCannotBlockSelf, status: http.StatusBadRequest, message: "Cannot block yourself"},
	{err: repositories.ErrAlreadyBlocked, status: http.StatusConflict, message: "Already blocking this user"},
}

var unblockUserErrors = []errorMapping{
	invalidUsernameError,
	userNotFoundError,
	{err: repositories.ErrNotBlocked, status: http.StatusConflict, message: "Not blocking this user"},
}

func NewBlockHandler(blockRepository repositories.BlockRepository, userRepository repositories.UserRepository) BlockHandler {
	return &blockHandler{
		blockServices: services.NewBlockService(blockRepository, userRepository),
	}
}

func (h *blockHandler) BlockUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	username := r.PathValue("username")
	err := h.blockServices.BlockUserByUsername(r.Context(), userID, username)
	if err != nil {
		writeError(w, err, "block user by username", "Error blocking user", blockUserErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *blockHandler) UnblockUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	username := r.PathValue("username")
	err := h.blockServices.UnblockUserByUsername(r.Context(), userID, username)
	if err != nil {
		writeError(w, err, "unblock user by username", "Error unblocking user", unblockUserErrors...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
