package repositories

import (
	"context"
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *domains.Session) error
	GetSessionByID(ctx context.Context, sessionID string) (*domains.Session, error)
	DeleteSessionByID(ctx context.Context, sessionID string) error
	DeleteOtherSessions(ctx context.Context, userID int64, keepSessionID string) error
	DeleteExpiredSessions(ctx context.Context) error
}

type sessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) CreateSession(ctx context.Context, session *domains.Session) error {
	query := `INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, session.ID, session.UserID, session.CreatedAt, session.ExpiresAt)
	return err
}

func (r *sessionRepository) DeleteOtherSessions(ctx context.Context, userID int64, keepSessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1 AND id <> $2`, userID, keepSessionID)
	return err
}

func (r *sessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}

func (r *sessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*domains.Session, error) {
	query := `SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = $1 AND expires_at > NOW()`
	var session domains.Session
	if err := r.db.GetContext(ctx, &session, query, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) DeleteSessionByID(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}
