package handlers

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type ReactionHandler interface {
	LikeReviewByID(w http.ResponseWriter, r *http.Request)
	DislikeReviewByID(w http.ResponseWriter, r *http.Request)
	RemoveReactionFromReviewByID(w http.ResponseWriter, r *http.Request)

	LikeCommentByID(w http.ResponseWriter, r *http.Request)
	DislikeCommentByID(w http.ResponseWriter, r *http.Request)
	RemoveReactionFromCommentByID(w http.ResponseWriter, r *http.Request)
}

type reactionHandler struct {
	reactionService services.ReactionService
}

type reactionTarget struct {
	pathParam   string
	invalidMsg  string
	notFoundMsg string
	notFoundErr error
	create      func(ctx context.Context, targetID, userID int64, value int16) (*domains.Reaction, error)
	remove      func(ctx context.Context, targetID, userID int64) error
}

func NewReactionHandler(reactionRepository repositories.ReactionRepository) ReactionHandler {
	return &reactionHandler{
		reactionService: services.NewReactionService(reactionRepository),
	}
}

func (h *reactionHandler) reviewTarget() reactionTarget {
	return reactionTarget{
		pathParam:   "review_id",
		invalidMsg:  "Invalid review ID",
		notFoundMsg: "Review not found",
		notFoundErr: repositories.ErrReviewNotFound,
		create:      h.reactionService.CreateReactionForReviewByID,
		remove:      h.reactionService.RemoveReactionFromReviewByID,
	}
}

func (h *reactionHandler) commentTarget() reactionTarget {
	return reactionTarget{
		pathParam:   "comment_id",
		invalidMsg:  "Invalid comment ID",
		notFoundMsg: "Comment not found",
		notFoundErr: repositories.ErrCommentNotFound,
		create:      h.reactionService.CreateReactionForCommentByID,
		remove:      h.reactionService.RemoveReactionFromCommentByID,
	}
}

func (h *reactionHandler) LikeReviewByID(w http.ResponseWriter, r *http.Request) {
	h.createReaction(w, r, h.reviewTarget(), 1)
}

func (h *reactionHandler) DislikeReviewByID(w http.ResponseWriter, r *http.Request) {
	h.createReaction(w, r, h.reviewTarget(), -1)
}

func (h *reactionHandler) RemoveReactionFromReviewByID(w http.ResponseWriter, r *http.Request) {
	h.removeReaction(w, r, h.reviewTarget())
}

func (h *reactionHandler) LikeCommentByID(w http.ResponseWriter, r *http.Request) {
	h.createReaction(w, r, h.commentTarget(), 1)
}

func (h *reactionHandler) DislikeCommentByID(w http.ResponseWriter, r *http.Request) {
	h.createReaction(w, r, h.commentTarget(), -1)
}

func (h *reactionHandler) RemoveReactionFromCommentByID(w http.ResponseWriter, r *http.Request) {
	h.removeReaction(w, r, h.commentTarget())
}

func (h *reactionHandler) createReaction(w http.ResponseWriter, r *http.Request, target reactionTarget, value int16) {
	targetID, err := getInt64PathValue(r, target.pathParam)
	if err != nil {
		http.Error(w, target.invalidMsg, http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	reaction, err := target.create(r.Context(), targetID, userID, value)
	if errors.Is(err, target.notFoundErr) {
		http.Error(w, target.notFoundMsg, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error creating reaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reaction)
}

func (h *reactionHandler) removeReaction(w http.ResponseWriter, r *http.Request, target reactionTarget) {
	targetID, err := getInt64PathValue(r, target.pathParam)
	if err != nil {
		http.Error(w, target.invalidMsg, http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err = target.remove(r.Context(), targetID, userID)
	if errors.Is(err, target.notFoundErr) {
		http.Error(w, target.notFoundMsg, http.StatusNotFound)
		return
	}
	if errors.Is(err, repositories.ErrReactionNotFound) {
		http.Error(w, "Reaction not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error removing reaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
