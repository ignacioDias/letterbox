package handlers

import (
	"errors"
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
}

type reactionHandler struct {
	reactionService services.ReactionService
}

func NewReactionHandler(reactionRepository repositories.ReactionRepository) ReactionHandler {
	return &reactionHandler{
		reactionService: services.NewReactionService(reactionRepository),
	}
}

func (h *reactionHandler) LikeReviewByID(w http.ResponseWriter, r *http.Request) {
	h.createReactionForReview(w, r, 1)
}

func (h *reactionHandler) DislikeReviewByID(w http.ResponseWriter, r *http.Request) {
	h.createReactionForReview(w, r, -1)
}

func (h *reactionHandler) createReactionForReview(w http.ResponseWriter, r *http.Request, value int16) {
	reviewID, err := getInt64PathValue(r, "review_id")
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	reaction, err := h.reactionService.CreateReactionForReviewByID(r.Context(), reviewID, userID, value)
	if errors.Is(err, repositories.ErrReviewNotFound) {
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error creating reaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reaction)
}
func (h *reactionHandler) RemoveReactionFromReviewByID(w http.ResponseWriter, r *http.Request) {
	reviewID, err := getInt64PathValue(r, "review_id")
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err = h.reactionService.RemoveReactionFromReviewByID(r.Context(), reviewID, userID)
	if errors.Is(err, repositories.ErrReviewNotFound) {
		http.Error(w, "Review not found", http.StatusNotFound)
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
