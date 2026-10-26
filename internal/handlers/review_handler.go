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

type ReviewHandler interface {
	CreateReviewForMovieByID(w http.ResponseWriter, r *http.Request)
	GetReviewsFromUserByUsername(w http.ResponseWriter, r *http.Request)
	GetReviewsFromFollowingUsers(w http.ResponseWriter, r *http.Request)
	GetReviewsFromMovieByID(w http.ResponseWriter, r *http.Request)
	GetReviewByID(w http.ResponseWriter, r *http.Request)
	GetAverageRatingForMovieByID(w http.ResponseWriter, r *http.Request)
	GetReviewsCountForMovieByID(w http.ResponseWriter, r *http.Request)
	UpdateReviewByID(w http.ResponseWriter, r *http.Request)
	DeleteReviewByID(w http.ResponseWriter, r *http.Request)
}

type reviewHandler struct {
	reviewService services.ReviewService
}

type ReviewReq struct {
	Score            int16  `json:"score"`
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"containsSpoilers"`
}

type UpdateReviewReq struct {
	Score            *int16  `json:"score,omitempty"`
	Content          *string `json:"content,omitempty"`
	ContainsSpoilers *bool   `json:"containsSpoilers,omitempty"`
}

func NewReviewHandler(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository, movieRepo repositories.MovieRepository) ReviewHandler {
	return &reviewHandler{
		reviewService: services.NewReviewService(reviewRepo, userRepo, blockRepo, movieRepo),
	}
}

func (h *reviewHandler) CreateReviewForMovieByID(w http.ResponseWriter, r *http.Request) {
	movieID, ok := movieIDFromRequest(w, r)
	if !ok {
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req ReviewReq
	if !decodeJSON(w, r, &req, reviewBodyLimit) {
		return
	}
	review := domains.NewReview(userID, movieID, req.Score, req.Content, req.ContainsSpoilers)
	review, err := h.reviewService.CreateReviewForMovieByID(r.Context(), review)
	if errors.Is(err, services.ErrInvalidScore) {
		http.Error(w, "Invalid score: 1-10", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrReviewContentTooLong) {
		http.Error(w, "Review content is too long: must be less than 10000 characters", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrReviewAlreadyExists) {
		http.Error(w, "Review already exists", http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("create review for movie by ID: %v", err)
		http.Error(w, "Failed to create review", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, review)
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
	movieID, ok := movieIDFromRequest(w, r)
	if !ok {
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

func (h *reviewHandler) GetReviewByID(w http.ResponseWriter, r *http.Request) {
	reviewID, err := getInt64PathValue(r, "review_id")
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}
	review, err := h.reviewService.GetReviewByID(r.Context(), reviewID)
	if errors.Is(err, repositories.ErrReviewNotFound) {
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get review by ID: %v", err)
		http.Error(w, "Failed to get review", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func (h *reviewHandler) GetAverageRatingForMovieByID(w http.ResponseWriter, r *http.Request) {
	handleMovieReviewAggregate(w, r, "averageRating", h.reviewService.GetAverageRatingForMovieByID,
		"get average rating for movie by ID", "Failed to get average rating")
}

func (h *reviewHandler) GetReviewsCountForMovieByID(w http.ResponseWriter, r *http.Request) {
	handleMovieReviewAggregate(w, r, "count", h.reviewService.GetReviewsCountForMovieByID,
		"get reviews count for movie by ID", "Failed to get reviews count")
}

func movieIDFromRequest(w http.ResponseWriter, r *http.Request) (int32, bool) {
	movieID, err := getInt32PathValue(r, "movie_id")
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return 0, false
	}
	return movieID, true
}

func handleMovieReviewAggregate[T any](w http.ResponseWriter, r *http.Request, key string, fetch func(ctx context.Context, movieID int32) (T, error), logMessage string, errorMessage string) {
	movieID, ok := movieIDFromRequest(w, r)
	if !ok {
		return
	}
	value, err := fetch(r.Context(), movieID)
	if errors.Is(err, repositories.ErrMovieNotFound) {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("%s: %v", logMessage, err)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]T{key: value})
}

func (h *reviewHandler) UpdateReviewByID(w http.ResponseWriter, r *http.Request) {
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
	var req UpdateReviewReq
	if !decodeJSON(w, r, &req, reviewBodyLimit) {
		return
	}
	if req.Score == nil && req.Content == nil && req.ContainsSpoilers == nil {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}
	review, err := h.reviewService.UpdateReviewByID(r.Context(), userID, reviewID, req.ContainsSpoilers, req.Content, req.Score)
	if errors.Is(err, services.ErrInvalidScore) {
		http.Error(w, "Invalid score: 1-10", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrReviewContentTooLong) {
		http.Error(w, "Review content is too long: must be less than 10000 characters", http.StatusBadRequest)
		return
	}
	if errors.Is(err, repositories.ErrReviewNotFound) {
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("update review for movie by ID: %v", err)
		http.Error(w, "Failed to update review", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, review)
}
func (h *reviewHandler) DeleteReviewByID(w http.ResponseWriter, r *http.Request) {
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
	err = h.reviewService.DeleteReviewByID(r.Context(), userID, reviewID)
	if errors.Is(err, repositories.ErrReviewNotFound) {
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("delete review for movie by ID: %v", err)
		http.Error(w, "Failed to delete review", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
