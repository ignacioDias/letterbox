package repositories

import (
	"context"
	"errors"
	"fmt"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type ReactionRepository interface {
	CreateReactionForReview(ctx context.Context, reaction *domains.Reaction) (*domains.Reaction, error)
	RemoveReactionFromReview(ctx context.Context, reviewID, userID int64) error
	CreateReactionForComment(ctx context.Context, reaction *domains.Reaction) (*domains.Reaction, error)
	RemoveReactionFromComment(ctx context.Context, commentID, userID int64) error
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
	return r.createReaction(ctx, "review_reactions", "review_id", reaction, ErrReviewNotFound)
}

func (r *reactionRepository) CreateReactionForComment(ctx context.Context, reaction *domains.Reaction) (*domains.Reaction, error) {
	return r.createReaction(ctx, "comment_reactions", "comment_id", reaction, ErrCommentNotFound)
}

func (r *reactionRepository) createReaction(ctx context.Context, tableName, columnName string, reaction *domains.Reaction, errNotFound error) (*domains.Reaction, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, %s, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, %s) DO UPDATE SET value = EXCLUDED.value
		RETURNING created_at
	`, tableName, columnName, columnName)
	err := r.db.QueryRowxContext(ctx, query, reaction.UserID, reaction.TargetID, reaction.Value).Scan(&reaction.CreatedAt)
	if err != nil {
		return nil, mapForeignKeyError(err, tableName, columnName, errNotFound)
	}
	return reaction, nil
}

func (r *reactionRepository) RemoveReactionFromReview(ctx context.Context, reviewID, userID int64) error {
	return r.removeReaction(ctx, "review_id", "reviews", "review_reactions", reviewID, userID, ErrReviewNotFound)
}

func (r *reactionRepository) RemoveReactionFromComment(ctx context.Context, commentID, userID int64) error {
	return r.removeReaction(ctx, "comment_id", "comments", "comment_reactions", commentID, userID, ErrCommentNotFound)
}

func (r *reactionRepository) removeReaction(ctx context.Context, columnName, tableName, reactionTableName string, targetID, userID int64, errNotFound error) error {
	var targetExists bool
	existsQuery := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE id = $1)`, tableName)
	if err := r.db.GetContext(ctx, &targetExists, existsQuery, targetID); err != nil {
		return err
	}
	if !targetExists {
		return errNotFound
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND user_id = $2`, reactionTableName, columnName)
	result, err := r.db.ExecContext(ctx, query, targetID, userID)
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
