package repositories

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type BlockRepository interface {
	BlockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error
	UnblockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error
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
	query := `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)`
	if _, err := r.db.ExecContext(ctx, query, userIDBlocker, userIDBlocked); err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyBlocked
		}
		return err
	}
	return nil
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
