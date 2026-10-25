package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"net/http"
	"os"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type MovieService interface {
	GetMovieByID(ctx context.Context, movieID int64) (*domains.Movie, error)
}

type tmdbMovieGetter interface {
	GetMovieDetails(id int, urlOptions map[string]string) (*tmdb.MovieDetails, error)
}

const tmdbNotFoundStatusCode = 34

type movieService struct {
	movieRepository repositories.MovieRepository
	newTMDBClient   func(ctx context.Context) (tmdbMovieGetter, error)
}

type contextTransport struct {
	ctx  context.Context
	base http.RoundTripper
}

func NewMovieService(movieRepository repositories.MovieRepository) MovieService {
	apiKey := os.Getenv("TMDB_API_KEY")
	return &movieService{
		movieRepository: movieRepository,
		newTMDBClient: func(ctx context.Context) (tmdbMovieGetter, error) {
			client, err := tmdb.Init(apiKey)
			if err != nil {
				return nil, err
			}
			client.SetClientConfig(http.Client{
				Transport: contextTransport{ctx: ctx, base: http.DefaultTransport},
			})
			return client, nil
		},
	}
}

func (t contextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req.WithContext(t.ctx))
}
func (s *movieService) GetMovieByID(ctx context.Context, movieID int64) (*domains.Movie, error) {
	movie, err := s.movieRepository.GetMovieByID(ctx, movieID)
	if err == nil {
		return movie, nil
	}
	if !errors.Is(err, repositories.ErrMovieNotFound) {
		return nil, err
	}
	movie, err = s.getMovieFromTMDB(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if err := s.movieRepository.SaveMovie(ctx, movie); err != nil {
		return nil, err
	}
	return movie, nil
}

func (s *movieService) getMovieFromTMDB(ctx context.Context, movieID int64) (*domains.Movie, error) {
	client, err := s.newTMDBClient(ctx)
	if err != nil {
		return nil, err
	}

	movie, err := client.GetMovieDetails(int(movieID), nil)
	if err != nil {
		var tmdbErr tmdb.Error
		if errors.As(err, &tmdbErr) && tmdbErr.StatusCode == tmdbNotFoundStatusCode {
			return nil, repositories.ErrMovieNotFound
		}
		return nil, err
	}
	return convertTMDBMovieToDomainMovie(movie), nil
}
func convertTMDBMovieToDomainMovie(tmdbMovie *tmdb.MovieDetails) *domains.Movie {
	return &domains.Movie{
		TMDBID:     int64(tmdbMovie.ID),
		Title:      tmdbMovie.Title,
		PosterPath: tmdbMovie.PosterPath,
		CachedAt:   time.Now(),
	}
}
