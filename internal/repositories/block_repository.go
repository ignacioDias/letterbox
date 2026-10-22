package repositories

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type BlockRepository interface {
	BlockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error
	UnblockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error
	IsBlockedBetween(ctx context.Context, userID1, userID2 int64) (bool, error)
}

type blockRepository struct {
	db *sqlx.DB
}

func NewBlockRepository(db *sqlx.DB) BlockRepository {
	return &blockRepository{
		db: db,
	}
}

func (r *blockRepository) BlockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM follows
	WHERE (follower_id = $1 AND followed_id = $2)
	   OR (follower_id = $2 AND followed_id = $1)`
	if _, err := tx.ExecContext(ctx, query, userIDBlocker, userIDBlocked); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)`, userIDBlocker, userIDBlocked); err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyBlocked
		}
		return err
	}
	return tx.Commit()
}

func (r *blockRepository) UnblockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error {
	query := `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`
	result, err := r.db.ExecContext(ctx, query, userIDBlocker, userIDBlocked)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotBlocked
	}
	return nil
}

func (r *blockRepository) IsBlockedBetween(ctx context.Context, userID1, userID2 int64) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM blocks
		WHERE (blocker_id = $1 AND blocked_id = $2)
		   OR (blocker_id = $2 AND blocked_id = $1)
	)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, userID1, userID2); err != nil {
		return false, err
	}
	return exists, nil
}
