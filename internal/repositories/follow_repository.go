package repositories

import (
	"context"
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type FollowRepository interface {
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	UnfollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error
	FollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error
}

type followRepository struct {
	db *sqlx.DB
}

func NewFollowRepository(db *sqlx.DB) FollowRepository {
	return &followRepository{
		db: db,
	}
}

func (r *followRepository) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
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

func (r *followRepository) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
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

func (r *followRepository) getIDByUsername(ctx context.Context, username string) (int64, error) {
	var userID int64
	if err := r.db.GetContext(ctx, &userID, `SELECT id FROM users WHERE username = $1`, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (r *followRepository) getPublicUsers(ctx context.Context, query string, args ...any) ([]*domains.PublicUser, error) {
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

func (r *followRepository) FollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO follows (follower_id, followed_id) VALUES ($1, $2)`, userIDFollower, userIDFollowing)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyFollowing
		}
		return err
	}
	return nil
}

func (r *followRepository) UnfollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM follows WHERE follower_id = $1 AND followed_id = $2`, userIDFollower, userIDFollowing)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFollowing
	}
	return nil
}
