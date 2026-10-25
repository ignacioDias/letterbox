package repositories

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapPgError(t *testing.T) {
	genericErr := errors.New("database unavailable")
	want := ErrReviewNotFound

	tests := []struct {
		name string
		err  error
		code string
		want error
	}{
		{name: "matching code", err: &pgconn.PgError{Code: "23503"}, code: "23503", want: want},
		{name: "wrapped matching code", err: fmt.Errorf("insert reaction: %w", &pgconn.PgError{Code: "23503"}), code: "23503", want: want},
		{name: "different code", err: &pgconn.PgError{Code: "23505"}, code: "23503", want: nil},
		{name: "other error", err: genericErr, code: "23503", want: nil},
		{name: "nil", err: nil, code: "23503", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapPgError(tt.err, tt.code, want)
			if tt.want == nil {
				if !errors.Is(got, tt.err) {
					t.Fatalf("mapPgError() = %v; want original error %v", got, tt.err)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapPgError() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestMapForeignKeyError(t *testing.T) {
	genericErr := errors.New("database unavailable")
	want := ErrCommentNotFound

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "matching target constraint",
			err:  &pgconn.PgError{Code: "23503", TableName: "comment_reactions", ConstraintName: "comment_reactions_comment_id_fkey"},
			want: want,
		},
		{
			name: "wrapped matching target constraint",
			err:  fmt.Errorf("insert reaction: %w", &pgconn.PgError{Code: "23503", TableName: "comment_reactions", ConstraintName: "comment_reactions_comment_id_fkey"}),
			want: want,
		},
		{
			name: "user constraint",
			err:  &pgconn.PgError{Code: "23503", TableName: "comment_reactions", ConstraintName: "comment_reactions_user_id_fkey"},
		},
		{
			name: "different table",
			err:  &pgconn.PgError{Code: "23503", TableName: "review_reactions", ConstraintName: "review_reactions_review_id_fkey"},
		},
		{
			name: "different code",
			err:  &pgconn.PgError{Code: "23505", TableName: "comment_reactions", ConstraintName: "comment_reactions_comment_id_fkey"},
		},
		{name: "other error", err: genericErr},
		{name: "nil", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapForeignKeyError(tt.err, "comment_reactions", "comment_id", want)
			if tt.want == nil {
				if !errors.Is(got, tt.err) {
					t.Fatalf("mapForeignKeyError() = %v; want original error %v", got, tt.err)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapForeignKeyError() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "foreign key violation", err: &pgconn.PgError{Code: "23503"}},
		{name: "wrapped unique violation", err: fmt.Errorf("insert follow: %w", &pgconn.PgError{Code: "23505"}), want: true},
		{name: "other error", err: errors.New("boom")},
		{name: "nil", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("isUniqueViolation(%v) = %v; want %v", tt.err, got, tt.want)
			}
		})
	}
}
