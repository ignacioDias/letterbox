package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
	"unicode/utf8"
)

type ReviewService interface {
	GetReviewsFromUserByUsername(ctx context.Context, userID int64, username string, limit, offset int) ([]*domains.Review, error)
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	UpdateReviewForMovieByID(ctx context.Context, review *domains.Review) error
	DeleteReviewForMovieByID(ctx context.Context, userID int64, movieID int32) error
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
}

type reviewService struct {
	reviewRepository repositories.ReviewRepository
	userRepository   repositories.UserRepository
	blockRepository  repositories.BlockRepository
	movieRepository  repositories.MovieRepository
}

var ErrInvalidScore = errors.New("invalid score: must be between 1 and 10")
var ErrReviewContentTooLong = errors.New("review content is too long: must be less than 10000 characters")

func NewReviewService(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository, movieRepo repositories.MovieRepository) ReviewService {
	return &reviewService{
		reviewRepository: reviewRepo,
		userRepository:   userRepo,
		blockRepository:  blockRepo,
		movieRepository:  movieRepo,
	}
}

func (s *reviewService) GetReviewsFromUserByUsername(ctx context.Context, userID int64, username string, limit, offset int) ([]*domains.Review, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	user, err := s.userRepository.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	reviewerID := user.ID
	blocked, err := s.blockRepository.IsBlockedBetween(ctx, userID, reviewerID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, repositories.ErrUserBlocked
	}
	return s.reviewRepository.GetReviewsFromUserByUserID(ctx, reviewerID, limit, offset)
}

func (s *reviewService) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	if review.Score < 1 || review.Score > 10 {
		return nil, ErrInvalidScore
	}
	if utf8.RuneCountInString(review.Content) > 10000 {
		return nil, ErrReviewContentTooLong
	}
	return s.reviewRepository.CreateReviewForMovieByID(ctx, review)
}

func (s *reviewService) UpdateReviewForMovieByID(ctx context.Context, review *domains.Review) error {
	// return s.reviewRepository.UpdateReviewForMovieByID(ctx, review)
	return nil
}

func (s *reviewService) DeleteReviewForMovieByID(ctx context.Context, userID int64, movieID int32) error {
	// return s.reviewRepository.DeleteReviewForMovieByID(ctx, userID, movieID)
	return nil
}

func (s *reviewService) GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	return s.reviewRepository.GetReviewsFromFollowingUsers(ctx, userID, limit, offset)
}

func (s *reviewService) GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
	return s.reviewRepository.GetReviewsFromMovieByID(ctx, movieID, limit, offset)
}
