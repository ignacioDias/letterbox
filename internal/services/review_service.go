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
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error)
	GetReviewsFromUserByUsername(ctx context.Context, userID int64, username string, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error)
	GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error)
	UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error)
	DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error
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

func (s *reviewService) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	if review.Score < 1 || review.Score > 10 {
		return nil, ErrInvalidScore
	}
	if utf8.RuneCountInString(review.Content) > 10000 {
		return nil, ErrReviewContentTooLong
	}
	return s.reviewRepository.CreateReviewForMovieByID(ctx, review)
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

func (s *reviewService) GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	return s.reviewRepository.GetReviewsFromFollowingUsers(ctx, userID, limit, offset)
}

func (s *reviewService) GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
	return s.reviewRepository.GetReviewsFromMovieByID(ctx, movieID, limit, offset)
}

func (s *reviewService) GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error) {
	return s.reviewRepository.GetReviewByID(ctx, reviewID)
}

func (s *reviewService) GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error) {
	return s.reviewRepository.GetAverageRatingForMovieByID(ctx, movieID)
}

func (s *reviewService) UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
	if score != nil && (*score < 1 || *score > 10) {
		return nil, ErrInvalidScore
	}
	if content != nil && utf8.RuneCountInString(*content) > 10000 {
		return nil, ErrReviewContentTooLong
	}
	return s.reviewRepository.UpdateReviewByID(ctx, userID, reviewID, containsSpoilers, content, score)
}

func (s *reviewService) DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error {
	return s.reviewRepository.DeleteReviewByID(ctx, userID, reviewID)
}
