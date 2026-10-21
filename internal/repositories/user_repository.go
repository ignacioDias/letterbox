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
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
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
