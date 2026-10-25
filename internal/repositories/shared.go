package repositories

import (
	"errors"
	"strings"

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
	ErrReviewNotFound        = errors.New("review not found")
	ErrCommentNotFound       = errors.New("comment not found")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapPgError(err error, code string, mapped error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == code {
		return mapped
	}
	return err
}

func mapForeignKeyError(err error, table, column string, mapped error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" &&
		pgErr.TableName == table &&
		strings.HasSuffix(pgErr.ConstraintName, "_"+column+"_fkey") {
		return mapped
	}
	return err
}
