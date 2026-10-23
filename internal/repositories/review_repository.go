package repositories

import (
	"context"
	"database/sql"
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
	DeleteReviewForMovie(ctx context.Context, userID int64, movieID int32) error
	UpdateReviewForMovie(ctx context.Context, userID int64, movieID int32, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error)
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

func (r *reviewRepository) DeleteReviewForMovie(ctx context.Context, userID int64, movieID int32) error {
	query := `
		DELETE FROM reviews
		WHERE user_id = $1 AND movie_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, userID, movieID)
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

func (r *reviewRepository) UpdateReviewForMovie(ctx context.Context, userID int64, movieID int32, containsSpoilers *bool, content *string, score *int16) (*domains.Review, error) {
	review := &domains.Review{}
	query := `
		UPDATE reviews
		SET score = COALESCE($1, score),
		    content = COALESCE($2, content),
		    contains_spoilers = COALESCE($3, contains_spoilers),
		    updated_at = NOW()
		WHERE user_id = $4 AND movie_id = $5
		RETURNING id, user_id, movie_id, score, COALESCE(content, '') AS content, contains_spoilers, created_at, updated_at
	`
	err := r.db.GetContext(ctx, review, query, score, content, containsSpoilers, userID, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}
