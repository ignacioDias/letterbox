package repositories

import (
	"context"
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type ReviewRepository interface {
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error)
	GetReviewsFromUserByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error)
	GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error)
	GetReviewsCountForMovieByID(ctx context.Context, movieID int32) (int, error)
	UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error)
	DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error
}

type reviewRepository struct {
	db *sqlx.DB
}

func NewReviewRepository(db *sqlx.DB) ReviewRepository {
	return &reviewRepository{
		db: db,
	}
}

func (r *reviewRepository) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	query := `
		INSERT INTO reviews (user_id, movie_id, score, content, contains_spoilers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, review.UserID, review.MovieID, review.Score, review.Content, review.ContainsSpoilers).Scan(&review.ID, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		return nil, mapPgError(err, "23505", ErrReviewAlreadyExists)
	}
	return review, nil
}

func (r *reviewRepository) GetReviewByID(ctx context.Context, reviewID int64) (*domains.Review, error) {
	review := &domains.Review{}
	query := `
		SELECT id, user_id, movie_id, score, COALESCE(content, '') AS content, contains_spoilers, created_at, updated_at
		FROM reviews
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, review, query, reviewID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}

func (r *reviewRepository) GetReviewsFromUserByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	reviews := []*domains.Review{}
	query := `
		SELECT r.id, r.user_id, r.movie_id, r.score, COALESCE(r.content, '') AS content, r.contains_spoilers, r.created_at, r.updated_at
		FROM reviews r
		WHERE r.user_id = $1
		ORDER BY r.updated_at DESC, r.id DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &reviews, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	reviews := []*domains.Review{}
	query := `
		SELECT r.id, r.user_id, r.movie_id, r.score, COALESCE(r.content, '') AS content, r.contains_spoilers, r.created_at, r.updated_at
		FROM reviews r
		JOIN follows f ON r.user_id = f.followed_user_id
		WHERE f.follower_user_id = $1
		ORDER BY r.updated_at DESC, r.id DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &reviews, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
	reviews := []*domains.Review{}
	query := `
		SELECT r.id, r.user_id, r.movie_id, r.score, COALESCE(r.content, '') AS content, r.contains_spoilers, r.created_at, r.updated_at
		FROM reviews r
		WHERE r.movie_id = $1
		ORDER BY r.updated_at DESC, r.id DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &reviews, query, movieID, limit, offset)
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) GetAverageRatingForMovieByID(ctx context.Context, movieID int32) (float64, error) {
	var averageRating float64
	query := `
		SELECT COALESCE(AVG(r.score), 0)::float8
		FROM movies m
		LEFT JOIN reviews r ON r.movie_id = m.tmdb_id
		WHERE m.tmdb_id = $1
		GROUP BY m.tmdb_id
	`
	err := r.db.GetContext(ctx, &averageRating, query, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrMovieNotFound
		}
		return 0, err
	}
	return averageRating, nil
}

func (r *reviewRepository) GetReviewsCountForMovieByID(ctx context.Context, movieID int32) (int, error) {
	var count int
	query := `
		SELECT COUNT(r.id)
		FROM movies m
		LEFT JOIN reviews r ON r.movie_id = m.tmdb_id
		WHERE m.tmdb_id = $1
		GROUP BY m.tmdb_id
	`
	err := r.db.GetContext(ctx, &count, query, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrMovieNotFound
		}
		return 0, err
	}
	return count, nil
}

func (r *reviewRepository) UpdateReviewByID(ctx context.Context, userID int64, reviewID int64, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
	review := &domains.Review{}
	query := `
		UPDATE reviews
		SET score = COALESCE($1, score),
		    content = COALESCE($2, content),
		    contains_spoilers = COALESCE($3, contains_spoilers),
		    updated_at = NOW()
		WHERE user_id = $4 AND id = $5
		RETURNING id, user_id, movie_id, score, COALESCE(content, '') AS content, contains_spoilers, created_at, updated_at
	`
	err := r.db.GetContext(ctx, review, query, score, content, containsSpoilers, userID, reviewID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}

func (r *reviewRepository) DeleteReviewByID(ctx context.Context, userID int64, reviewID int64) error {
	query := `
		DELETE FROM reviews
		WHERE user_id = $1 AND id = $2
	`
	result, err := r.db.ExecContext(ctx, query, userID, reviewID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrReviewNotFound
	}
	return nil
}
