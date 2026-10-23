package repositories

import (
	"context"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type ReviewRepository interface {
	GetReviewsFromUserByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromFollowingUsers(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByID(ctx context.Context, movieID int64, limit, offset int) ([]*domains.Review, error)
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

func (r *reviewRepository) GetReviewsFromMovieByID(ctx context.Context, movieID int64, limit, offset int) ([]*domains.Review, error) {
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
