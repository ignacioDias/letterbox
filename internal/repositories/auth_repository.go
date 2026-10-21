package repositories

import (
	"errors"
	"letterbox/internal/domains"

	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *domains.User) (int64, error)
}

type authRepository struct {
	db *sqlx.DB
}

var (
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
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
