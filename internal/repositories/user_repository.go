package repositories

import (
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domains.User) (int64, error)
	GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
	DeleteUserByID(ctx context.Context, userID int64) error
	UpdateUser(ctx context.Context, user *domains.User) error
}

type userRepository struct {
	db *sqlx.DB
}

var (
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrSessionNotFound       = errors.New("session not found")
)

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *domains.User) (int64, error) {
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

func (r *userRepository) GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (*domains.User, error) {
	query := `SELECT id, username, email, password_hash, profile_picture_url, country FROM users WHERE email = $1 OR username = $2`
	return r.getUser(ctx, query, emailOrUsername, emailOrUsername)
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*domains.User, error) {
	query := `SELECT id, username, email, profile_picture_url, country FROM users WHERE username = $1`
	return r.getUser(ctx, query, username)
}

func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*domains.User, error) {
	query := `SELECT id, username, email, profile_picture_url, country FROM users WHERE id = $1`
	return r.getUser(ctx, query, userID)
}

func (r *userRepository) getUser(ctx context.Context, query string, args ...any) (*domains.User, error) {
	var user domains.User
	if err := r.db.GetContext(ctx, &user, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) DeleteUserByID(ctx context.Context, userID int64) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domains.User) error {
	query := `UPDATE users SET username = $1, email = $2, profile_picture_url = $3, country = $4, updated_at = NOW() WHERE id = $5`
	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.ProfilePictureURL, user.Country, user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return ErrUsernameAlreadyExists
			case "users_email_key":
				return ErrEmailAlreadyExists
			}
		}
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
