package services

import (
	"context"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
)

type ReviewService interface {
	GetReviewsFromUserByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.Review, error)
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) error
	UpdateReviewForMovieByID(ctx context.Context, review *domains.Review) error
	DeleteReviewForMovieByID(ctx context.Context, userID int64, movieID int64) error
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
}

type reviewService struct {
	reviewRepository repositories.ReviewRepository
	userRepository   repositories.UserRepository
	movieRepository  repositories.MovieRepository
}

func NewReviewService(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, movieRepo repositories.MovieRepository) ReviewService {
	return &reviewService{
		reviewRepository: reviewRepo,
		userRepository:   userRepo,
		movieRepository:  movieRepo,
	}
}

func (s *reviewService) GetReviewsFromUserByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.Review, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	user, err := s.userRepository.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	userID := user.ID
	return s.reviewRepository.GetReviewsFromUserByUserID(ctx, userID, limit, offset)
}

func (s *reviewService) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) error {
	return nil
}

func (s *reviewService) UpdateReviewForMovieByID(ctx context.Context, review *domains.Review) error {
	// return s.reviewRepository.UpdateReviewForMovieByID(ctx, review)
	return nil
}

func (s *reviewService) DeleteReviewForMovieByID(ctx context.Context, userID int64, movieID int64) error {
	// return s.reviewRepository.DeleteReviewForMovieByID(ctx, userID, movieID)
	return nil
}

func (s *reviewService) GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	// return s.reviewRepository.GetReviewsFromFollowingUsers(ctx, userID, limit, offset)
	return nil, nil
}
