package repositories

import "github.com/jmoiron/sqlx"

type MovieRepository interface{}

type movieRepository struct {
	db *sqlx.DB
}

func NewMovieRepository(db *sqlx.DB) MovieRepository {
	return &movieRepository{}
}
