package repositories

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

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
