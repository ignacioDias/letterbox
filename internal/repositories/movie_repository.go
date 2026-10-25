package repositories

import (
	"context"
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type MovieRepository interface {
	SaveMovie(ctx context.Context, movie *domains.Movie) error
	GetMovieByID(ctx context.Context, movieID int64) (*domains.Movie, error)
}

type movieRepository struct {
	db *sqlx.DB
}

func NewMovieRepository(db *sqlx.DB) MovieRepository {
	return &movieRepository{
		db: db,
	}
}

func (r *movieRepository) SaveMovie(ctx context.Context, movie *domains.Movie) error {
	query := `INSERT INTO movies (tmdb_id, title, poster_path) VALUES ($1, $2, $3) ON CONFLICT (tmdb_id) DO UPDATE SET title = EXCLUDED.title, poster_path = EXCLUDED.poster_path RETURNING cached_at`
	return r.db.QueryRowxContext(ctx, query, movie.TMDBID, movie.Title, movie.PosterPath).Scan(&movie.CachedAt)
}

func (r *movieRepository) GetMovieByID(ctx context.Context, movieID int64) (*domains.Movie, error) {
	query := `SELECT tmdb_id, title, poster_path, cached_at FROM movies WHERE tmdb_id = $1`
	var movie domains.Movie
	if err := r.db.GetContext(ctx, &movie, query, movieID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieNotFound
		}
		return nil, err
	}
	return &movie, nil
}
