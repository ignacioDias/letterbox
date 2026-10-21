package repositories

import (
	"letterbox/internal/domains"

	"context"

	"github.com/jmoiron/sqlx"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *domains.User) (int64, error)
}

type authRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &authRepository{
		db: db,
	}
}

func (r *authRepository) CreateUser(ctx context.Context, user *domains.User) (int64, error) {
	query := `INSERT INTO users (username, email, password_hash, profile_picture_url, country) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := r.db.DB.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.ProfilePictureURL, user.Country).Scan(&user.ID)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}
