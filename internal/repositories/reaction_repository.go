package repositories

import (
	"context"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type ReactionRepository interface {
	CreateReactionForReview(ctx context.Context, reaction *domains.Reaction) (*domains.Reaction, error)
	RemoveReactionFromReview(ctx context.Context, reviewID, userID int64) error
}

type reactionRepository struct {
	db *sqlx.DB
}

var ErrReactionNotFound = errors.New("Reaction not found")

func NewReactionRepository(db *sqlx.DB) ReactionRepository {
	return &reactionRepository{
		db: db,
	}
}

func (r *reactionRepository) CreateReactionForReview(ctx context.Context, reaction *domains.Reaction) (*domains.Reaction, error) {
	query := `
		INSERT INTO review_reactions (user_id, review_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, review_id) DO UPDATE SET value = EXCLUDED.value
		RETURNING created_at
	`
	err := r.db.QueryRowxContext(ctx, query, reaction.UserID, reaction.TargetID, reaction.Value).Scan(&reaction.CreatedAt)
	if err != nil {
		return nil, mapPgError(err, "23503", ErrReviewNotFound)
	}
	return reaction, nil
}

func (r *reactionRepository) RemoveReactionFromReview(ctx context.Context, reviewID, userID int64) error {
	var reviewExists bool
	existsQuery := `SELECT EXISTS (SELECT 1 FROM reviews WHERE id = $1)`
	if err := r.db.GetContext(ctx, &reviewExists, existsQuery, reviewID); err != nil {
		return err
	}
	if !reviewExists {
		return ErrReviewNotFound
	}
	query := `DELETE FROM review_reactions WHERE review_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, reviewID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrReactionNotFound
	}
	return nil
}
