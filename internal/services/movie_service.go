package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type MovieService interface {
	GetMovieByID(ctx context.Context, movieID int32) (*domains.Movie, error)
	GetTrendingMovies(ctx context.Context, page string) (*domains.TrendingMovies, error)
}

type tmdbMovieGetter interface {
	GetMovieDetails(id int, urlOptions map[string]string) (*tmdb.MovieDetails, error)
	GetTrending(mediaType, timeWindow string, urlOptions map[string]string) (*tmdb.Trending, error)
}

const tmdbNotFoundStatusCode = 34

const maxTMDBPage = 500

var ErrInvalidPage = errors.New("invalid page number")

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
func (s *movieService) GetMovieByID(ctx context.Context, movieID int32) (*domains.Movie, error) {
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

func (s *movieService) getMovieFromTMDB(ctx context.Context, movieID int32) (*domains.Movie, error) {
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
func (s *movieService) GetTrendingMovies(ctx context.Context, page string) (*domains.TrendingMovies, error) {
	if page == "" {
		page = "1"
	}
	if ok := isValidPage(page); !ok {
		return nil, ErrInvalidPage
	}
	tmdbClient, err := s.newTMDBClient(ctx)
	if err != nil {
		return nil, err
	}
	options := make(map[string]string)
	options["page"] = page

	trending, err := tmdbClient.GetTrending("movie", "day", options)

	if err != nil {
		return nil, err
	}

	results := s.convertTMDBTrendingToDomainMovies(ctx, trending)

	return &domains.TrendingMovies{
		Page:         trending.Page,
		TotalPages:   trending.TotalPages,
		TotalResults: trending.TotalResults,
		Results:      results,
	}, nil
}

func isValidPage(page string) bool {
	n, err := strconv.Atoi(page)
	if err != nil {
		return false
	}
	return n >= 1 && n <= maxTMDBPage
}

func (s *movieService) convertTMDBTrendingToDomainMovies(ctx context.Context, trending *tmdb.Trending) []*domains.Movie {
	if trending == nil || len(trending.Results) == 0 {
		return []*domains.Movie{}
	}
	var domainMovies []*domains.Movie
	domainMovies = make([]*domains.Movie, 0, len(trending.Results))
	for _, result := range trending.Results {
		domainMovie := domains.NewMovie(int32(result.ID), result.Title, result.PosterPath, time.Now())
		if err := s.movieRepository.SaveMovie(ctx, domainMovie); err != nil {
			log.Printf("Error saving movie to repository: %v", err)
		}
		domainMovies = append(domainMovies, domainMovie)
	}
	return domainMovies
}
func convertTMDBMovieToDomainMovie(tmdbMovie *tmdb.MovieDetails) *domains.Movie {
	return &domains.Movie{
		TMDBID:     int32(tmdbMovie.ID),
		Title:      tmdbMovie.Title,
		PosterPath: tmdbMovie.PosterPath,
		CachedAt:   time.Now(),
	}
}
