package services

import (
	"context"
	"errors"
	"letterbox/internal/cache"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

var reviewCacheGenerationCounter atomic.Uint64

const reviewCacheTTLSeconds = 3600

type ReviewService interface {
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error)
	GetReviewsFromUserByUsername(ctx context.Context, userID int64, username string, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error)
	GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error)
	GetReviewsCountForMovieByID(ctx context.Context, movieID int32) (int, error)
	UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error)
	DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error
}

type reviewService struct {
	reviewRepository repositories.ReviewRepository
	userRepository   repositories.UserRepository
	blockRepository  repositories.BlockRepository
	cache            cache.Cache
	cacheEnabled     bool
}

var ErrInvalidScore = errors.New("invalid score: must be between 1 and 10")
var ErrReviewContentTooLong = errors.New("review content is too long: must be less than 10000 characters")

func NewReviewService(reviewRepo repositories.ReviewRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository, caches ...cache.Cache) ReviewService {
	reviewCache := cache.Cache(noopCache{})
	cacheEnabled := false
	if len(caches) > 0 && caches[0] != nil {
		reviewCache = caches[0]
		cacheEnabled = true
	}
	return &reviewService{
		reviewRepository: reviewRepo,
		userRepository:   userRepo,
		blockRepository:  blockRepo,
		cache:            reviewCache,
		cacheEnabled:     cacheEnabled,
	}
}

func (s *reviewService) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	if review.Score < 1 || review.Score > 10 {
		return nil, ErrInvalidScore
	}
	if utf8.RuneCountInString(review.Content) > 10000 {
		return nil, ErrReviewContentTooLong
	}
	created, err := s.reviewRepository.CreateReviewForMovieByID(ctx, review)
	if err != nil {
		return nil, err
	}
	s.invalidateReviewCache(ctx, created.UserID, created.MovieID, created.ID)
	return created, nil
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
	generation := s.getCacheGeneration(ctx, getKeyForUserReviewsGeneration(reviewerID))
	key := getKeyForUserReviews(reviewerID, limit, offset, generation)
	reviews, found, err := s.getReviewsFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return reviews, nil
	}
	reviews, err = s.reviewRepository.GetReviewsFromUserByUserID(ctx, reviewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cacheReviews(ctx, key, reviews)
	return reviews, nil
}

func (s *reviewService) GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	generation := s.getCacheGeneration(ctx, getKeyForFollowingReviewsGeneration())
	key := getKeyForFollowingReviews(userID, limit, offset, generation)
	reviews, found, err := s.getReviewsFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return reviews, nil
	}
	reviews, err = s.reviewRepository.GetReviewsFromFollowingUsers(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cacheReviews(ctx, key, reviews)
	return reviews, nil
}

func (s *reviewService) GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
	generation := s.getCacheGeneration(ctx, getKeyForMovieReviewsGeneration(movieID))
	key := getKeyForMovieReviews(movieID, limit, offset, generation)
	reviews, found, err := s.getReviewsFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return reviews, nil
	}
	reviews, err = s.reviewRepository.GetReviewsFromMovieByID(ctx, movieID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cacheReviews(ctx, key, reviews)
	return reviews, nil
}

func (s *reviewService) GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error) {
	key := getKeyForReviewByID(reviewID)
	review, found, err := s.getReviewFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return review, nil
	}
	review, err = s.reviewRepository.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	s.cacheReview(ctx, key, review)
	return review, nil
}

func (s *reviewService) GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error) {
	key := getKeyForMovieReviewAverage(movieID)
	rating, found, err := s.getFloatFromCache(ctx, key)
	if err != nil {
		return 0, err
	}
	if found {
		return rating, nil
	}
	rating, err = s.reviewRepository.GetAverageRatingForMovieByID(ctx, movieID)
	if err != nil {
		return 0, err
	}
	s.cacheFloat(ctx, key, rating)
	return rating, nil
}

func (s *reviewService) GetReviewsCountForMovieByID(ctx context.Context, movieID int32) (int, error) {
	key := getKeyForMovieReviewCount(movieID)
	count, found, err := s.getIntFromCache(ctx, key)
	if err != nil {
		return 0, err
	}
	if found {
		return count, nil
	}
	count, err = s.reviewRepository.GetReviewsCountForMovieByID(ctx, movieID)
	if err != nil {
		return 0, err
	}
	s.cacheInt(ctx, key, count)
	return count, nil
}

func (s *reviewService) UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
	if score != nil && (*score < 1 || *score > 10) {
		return nil, ErrInvalidScore
	}
	if content != nil && utf8.RuneCountInString(*content) > 10000 {
		return nil, ErrReviewContentTooLong
	}
	review, err := s.reviewRepository.UpdateReviewByID(ctx, userID, reviewID, containsSpoilers, content, score)
	if err != nil {
		return nil, err
	}
	s.invalidateReviewCache(ctx, review.UserID, review.MovieID, review.ID)
	return review, nil
}

func (s *reviewService) DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error {
	var movieID int32
	if s.cacheEnabled {
		review, err := s.reviewRepository.GetReviewByID(ctx, reviewID)
		if err != nil {
			return err
		}
		movieID = review.MovieID
	}
	if err := s.reviewRepository.DeleteReviewByID(ctx, userID, reviewID); err != nil {
		return err
	}
	s.invalidateReviewCache(ctx, userID, movieID, reviewID)
	return nil
}

func (s *reviewService) getCacheGeneration(ctx context.Context, key string) string {
	generation, err := s.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrKeyNotFound) {
			log.Printf("Error reading review cache generation: %v", err)
		}
		return "0"
	}
	if generation == "" {
		return "0"
	}
	return generation
}

func newReviewCacheGeneration() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(reviewCacheGenerationCounter.Add(1), 10)
}

func (s *reviewService) resetCacheGeneration(ctx context.Context, key string) {
	if err := s.cache.Set(ctx, key, newReviewCacheGeneration(), reviewCacheTTLSeconds); err != nil {
		log.Printf("Error resetting review cache generation: %v", err)
	}
}

func (s *reviewService) getReviewsFromCache(ctx context.Context, key string) ([]*domains.Review, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		log.Printf("Error reading reviews from cache: %v", err)
		return nil, false, nil
	}
	reviews, err := generateReviewsFromCacheResult(result)
	if err == nil {
		return reviews, true, nil
	}
	log.Printf("Error decoding cached reviews: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached reviews: %v", err)
	}
	return nil, false, nil
}

func (s *reviewService) cacheReviews(ctx context.Context, key string, reviews []*domains.Review) {
	value, err := generateCacheValueFromReviews(reviews)
	if err != nil {
		log.Printf("Error encoding reviews for cache: %v", err)
		return
	}
	if err := s.cache.Set(ctx, key, value, reviewCacheTTLSeconds); err != nil {
		log.Printf("Error saving reviews to cache: %v", err)
	}
}

func (s *reviewService) getReviewFromCache(ctx context.Context, key string) (*domains.Review, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		log.Printf("Error reading review from cache: %v", err)
		return nil, false, nil
	}
	review, err := generateReviewFromCacheResult(result)
	if err == nil {
		return review, true, nil
	}
	log.Printf("Error decoding cached review: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached review: %v", err)
	}
	return nil, false, nil
}

func (s *reviewService) cacheReview(ctx context.Context, key string, review *domains.Review) {
	value, err := generateCacheValueFromReview(review)
	if err != nil {
		log.Printf("Error encoding review for cache: %v", err)
		return
	}
	if err := s.cache.Set(ctx, key, value, reviewCacheTTLSeconds); err != nil {
		log.Printf("Error saving review to cache: %v", err)
	}
}

func (s *reviewService) getFloatFromCache(ctx context.Context, key string) (float64, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return 0, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, false, ctxErr
		}
		log.Printf("Error reading review aggregate from cache: %v", err)
		return 0, false, nil
	}
	value, err := strconv.ParseFloat(result, 64)
	if err == nil {
		return value, true, nil
	}
	log.Printf("Error decoding cached review aggregate: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached review aggregate: %v", err)
	}
	return 0, false, nil
}

func (s *reviewService) cacheFloat(ctx context.Context, key string, value float64) {
	if err := s.cache.Set(ctx, key, strconv.FormatFloat(value, 'g', -1, 64), reviewCacheTTLSeconds); err != nil {
		log.Printf("Error saving review aggregate to cache: %v", err)
	}
}

func (s *reviewService) getIntFromCache(ctx context.Context, key string) (int, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return 0, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, false, ctxErr
		}
		log.Printf("Error reading review count from cache: %v", err)
		return 0, false, nil
	}
	value, err := strconv.Atoi(result)
	if err == nil {
		return value, true, nil
	}
	log.Printf("Error decoding cached review count: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached review count: %v", err)
	}
	return 0, false, nil
}

func (s *reviewService) cacheInt(ctx context.Context, key string, value int) {
	if err := s.cache.Set(ctx, key, strconv.Itoa(value), reviewCacheTTLSeconds); err != nil {
		log.Printf("Error saving review count to cache: %v", err)
	}
}

func (s *reviewService) invalidateReviewCache(ctx context.Context, userID int64, movieID int32, reviewID int64) {
	if !s.cacheEnabled {
		return
	}
	s.resetCacheGeneration(ctx, getKeyForUserReviewsGeneration(userID))
	s.resetCacheGeneration(ctx, getKeyForMovieReviewsGeneration(movieID))
	s.resetCacheGeneration(ctx, getKeyForFollowingReviewsGeneration())

	keys := []string{
		getKeyForMovieReviewAverage(movieID),
		getKeyForMovieReviewCount(movieID),
		getKeyForReviewByID(reviewID),
	}
	if _, err := s.cache.Delete(ctx, keys); err != nil {
		log.Printf("Error deleting review cache: %v", err)
	}
}
