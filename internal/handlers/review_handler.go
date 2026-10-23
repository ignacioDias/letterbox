package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/middleware"
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
	GetReviewsFromMovieByID(w http.ResponseWriter, r *http.Request)
}

type reviewHandler struct {
	reviewService services.ReviewService
}

type ReviewReq struct {
	Score            int16  `json:"score"`
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"contains_spoilers"`
}

func NewReviewHandler(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository, movieRepo repositories.MovieRepository) ReviewHandler {
	return &reviewHandler{
		reviewService: services.NewReviewService(reviewRepo, userRepo, blockRepo, movieRepo),
	}
}

func (h *reviewHandler) GetReviewsFromUserByUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username := r.PathValue("username")
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	reviews, err := h.reviewService.GetReviewsFromUserByUsername(r.Context(), userID, username, limit, offset)
	if errors.Is(err, services.ErrInvalidUsername) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, repositories.ErrUserBlocked) {
		http.Error(w, "Cannot view this user's reviews", http.StatusForbidden)
		return
	}
	if err != nil {
		log.Printf("get reviews from user by username: %v", err)
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reviews)

}

func (h *reviewHandler) GetReviewsFromFollowingUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	reviews, err := h.reviewService.GetReviewsFromFollowingUsers(r.Context(), userID, limit, offset)
	if err != nil {
		log.Printf("get reviews from following users: %v", err)
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}
func (h *reviewHandler) GetReviewsFromMovieByID(w http.ResponseWriter, r *http.Request) {
	movieID, err := getMovieIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}
	reviews, err := h.reviewService.GetReviewsFromMovieByID(r.Context(), movieID, limit, offset)
	if err != nil {
		log.Printf("get reviews from movie by ID: %v", err)
		http.Error(w, "Failed to get reviews", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (h *reviewHandler) CreateReviewForMovieByID(w http.ResponseWriter, r *http.Request) {
	movieID, err := getMovieIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req ReviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	review := domains.NewReview(userID, movieID, req.Score, req.Content, req.ContainsSpoilers)
	review, err = h.reviewService.CreateReviewForMovieByID(r.Context(), review)
	if errors.Is(err, services.ErrInvalidScore) {
		http.Error(w, "Invalid score: 1-10", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrReviewAlreadyExists) {
		http.Error(w, "Review already exists", http.StatusConflict)
		return
	}
	if errors.Is(err, repositories.ErrMovieNotFound) {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("create review for movie by ID: %v", err)
		http.Error(w, "Failed to create review", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func (h *reviewHandler) UpdateReviewForMovieByID(w http.ResponseWriter, r *http.Request) {}
func (h *reviewHandler) DeleteReviewForMovieByID(w http.ResponseWriter, r *http.Request) {}
