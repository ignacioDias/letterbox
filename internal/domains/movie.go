package domains

import "time"

type Movie struct {
	TMDBID     int64     `json:"tmdbId" db:"tmdb_id"`
	Title      string    `json:"title" db:"title"`
	PosterPath string    `json:"posterPath" db:"poster_path"`
	CachedAt   time.Time `json:"cachedAt" db:"cached_at"`
}

func NewMovie(tmdbID int64, title, posterPath string, cachedAt time.Time) *Movie {
	return &Movie{
		TMDBID:     tmdbID,
		Title:      title,
		PosterPath: posterPath,
		CachedAt:   cachedAt,
	}
}
