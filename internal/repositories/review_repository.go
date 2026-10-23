package repositories

import (
	"context"
	"errors"
	"letterbox/internal/domains"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type ReviewRepository interface {
	GetReviewsFromUserByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error)
}

type reviewRepository struct {
	db *sqlx.DB
}

func NewReviewRepository(db *sqlx.DB) ReviewRepository {
	return &reviewRepository{
		db: db,
	}
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

func (r *reviewRepository) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	query := `
		INSERT INTO reviews (user_id, movie_id, score, content, contains_spoilers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, review.UserID, review.MovieID, review.Score, review.Content, review.ContainsSpoilers).Scan(&review.ID, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		return nil, mapReviewInsertError(err)
	}
	return review, nil
}

func mapReviewInsertError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return ErrReviewAlreadyExists
	}
	return err
}
