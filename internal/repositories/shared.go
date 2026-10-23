package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrSessionNotFound       = errors.New("session not found")
	ErrAlreadyFollowing      = errors.New("already following this user")
	ErrNotFollowing          = errors.New("not following this user")
	ErrAlreadyBlocked        = errors.New("already blocked this user")
	ErrNotBlocked            = errors.New("not blocking this user")
	ErrUserBlocked           = errors.New("user is blocked")
	ErrReviewAlreadyExists   = errors.New("review already exists")
	ErrMovieNotFound         = errors.New("movie not found")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
