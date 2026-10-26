package domains

import "time"

type Movie struct {
	TMDBID     int32     `json:"tmdbId" db:"tmdb_id"`
	Title      string    `json:"title" db:"title"`
	PosterPath string    `json:"posterPath" db:"poster_path"`
	CachedAt   time.Time `json:"cachedAt" db:"cached_at"`
}

type PaginatedMovies struct {
	Page         int64    `json:"page"`
	TotalPages   int64    `json:"totalPages"`
	TotalResults int64    `json:"totalResults"`
	Results      []*Movie `json:"results"`
}

func NewMovie(tmdbID int32, title, posterPath string, cachedAt time.Time) *Movie {
	return &Movie{
		TMDBID:     tmdbID,
		Title:      title,
		PosterPath: posterPath,
		CachedAt:   cachedAt,
	}
}

func NewPaginatedMovies(page, totalPages, totalResults int64, results []*Movie) *PaginatedMovies {
	return &PaginatedMovies{
		Page:         page,
		TotalPages:   totalPages,
		TotalResults: totalResults,
		Results:      results,
	}
}
