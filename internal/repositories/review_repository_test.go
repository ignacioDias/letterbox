package repositories

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapReviewInsertError(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "unique violation",
			err:  &pgconn.PgError{Code: "23505"},
			want: ErrReviewAlreadyExists,
		},
		{
			name: "wrapped unique violation",
			err:  fmt.Errorf("insert review: %w", &pgconn.PgError{Code: "23505"}),
			want: ErrReviewAlreadyExists,
		},
		{
			name: "foreign key violation",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "reviews_user_id_fkey"},
			want: nil,
		},
		{
			name: "other database error",
			err:  genericErr,
			want: nil,
		},
		{
			name: "nil",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapReviewInsertError(tt.err)
			if tt.want == nil {
				if !errors.Is(got, tt.err) {
					t.Fatalf("mapReviewInsertError() = %v; want original error %v", got, tt.err)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapReviewInsertError() = %v; want %v", got, tt.want)
			}
		})
	}
}
