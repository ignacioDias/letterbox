package repositories

import (
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *domains.User) (int64, error)
	GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (*domains.User, error)
	CreateSession(ctx context.Context, session *domains.Session) error
	GetSessionByID(ctx context.Context, sessionID string) (*domains.Session, error)
	DeleteExpiredSessions(ctx context.Context) error
}

type authRepository struct {
	db *sqlx.DB
}

var (
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrSessionNotFound       = errors.New("session not found")
)

func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &authRepository{
		db: db,
	}
}

func (r *authRepository) CreateUser(ctx context.Context, user *domains.User) (int64, error) {
	query := `INSERT INTO users (username, email, password_hash, profile_picture_url, country) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.ProfilePictureURL, user.Country).Scan(&user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return 0, ErrUsernameAlreadyExists
			case "users_email_key":
				return 0, ErrEmailAlreadyExists
			}
		}
		return 0, err
	}
	return user.ID, nil
}

func (r *authRepository) GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (*domains.User, error) {
	query := `SELECT id, username, email, password_hash, profile_picture_url, country FROM users WHERE email = $1 OR username = $2`
	var user domains.User
	if err := r.db.GetContext(ctx, &user, query, emailOrUsername, emailOrUsername); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CreateSession(ctx context.Context, session *domains.Session) error {
	query := `INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, session.ID, session.UserID, session.CreatedAt, session.ExpiresAt)
	return err
}

func (r *authRepository) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}

func (r *authRepository) GetSessionByID(ctx context.Context, sessionID string) (*domains.Session, error) {
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
