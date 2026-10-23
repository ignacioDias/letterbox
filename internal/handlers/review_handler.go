package handlers

import (
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type ReviewHandler interface {
	GetReviewsFromUserByUsername(w http.ResponseWriter, r *http.Request)
	CreateReviewForMovieByID(w http.ResponseWriter, r *http.Request)
	UpdateReviewForMovieByID(w http.ResponseWriter, r *http.Request)
	DeleteReviewForMovieByID(w http.ResponseWriter, r *http.Request)
	GetReviewsFromFollowingUsers(w http.ResponseWriter, r *http.Request)
}

type reviewHandler struct {
	reviewService services.ReviewService
}

func NewReviewHandler(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, movieRepo repositories.MovieRepository) ReviewHandler {
	return &reviewHandler{
		reviewService: services.NewReviewService(reviewRepo, userRepo, movieRepo),
	}
}

func (h *reviewHandler) GetReviewsFromUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	reviews, err := h.reviewService.GetReviewsFromUserByUsername(r.Context(), username, limit, offset)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get reviews from user by username: %v", err)
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reviews)

}
func (h *reviewHandler) CreateReviewForMovieByID(w http.ResponseWriter, r *http.Request)     {}
func (h *reviewHandler) UpdateReviewForMovieByID(w http.ResponseWriter, r *http.Request)     {}
func (h *reviewHandler) DeleteReviewForMovieByID(w http.ResponseWriter, r *http.Request)     {}
func (h *reviewHandler) GetReviewsFromFollowingUsers(w http.ResponseWriter, r *http.Request) {}
