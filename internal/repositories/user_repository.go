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
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
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

func (r *userRepository) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	userID, err := r.getIDByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	query := `SELECT u.id, u.username, u.profile_picture_url, u.country
	FROM users u
	JOIN follows f ON u.id = f.follower_id
	WHERE f.followed_id = $1
	ORDER BY u.username
	LIMIT $2 OFFSET $3`
	return r.getPublicUsers(ctx, query, userID, limit, offset)
}

func (r *userRepository) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	userID, err := r.getIDByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	query := `SELECT u.id, u.username, u.profile_picture_url, u.country
	FROM users u
	JOIN follows f ON u.id = f.followed_id
	WHERE f.follower_id = $1
	ORDER BY u.username
	LIMIT $2 OFFSET $3`
	return r.getPublicUsers(ctx, query, userID, limit, offset)
}

func (r *userRepository) getIDByUsername(ctx context.Context, username string) (int64, error) {
	var userID int64
	if err := r.db.GetContext(ctx, &userID, `SELECT id FROM users WHERE username = $1`, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (r *userRepository) getPublicUsers(ctx context.Context, query string, args ...any) ([]*domains.PublicUser, error) {
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domains.PublicUser, 0)
	for rows.Next() {
		var user domains.PublicUser
		if err := rows.StructScan(&user); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
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
