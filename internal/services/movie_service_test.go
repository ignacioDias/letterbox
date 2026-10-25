package services

import (
	"context"
	"errors"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type fakeTMDBMovieGetter struct {
	movie *tmdb.MovieDetails
	err   error
}

func (f *fakeTMDBMovieGetter) GetMovieDetails(int, map[string]string) (*tmdb.MovieDetails, error) {
	return f.movie, f.err
}

func TestGetMovieByID(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	cachedMovie := &domains.Movie{TMDBID: 10, Title: "Cached"}
	tmdbMovie := &tmdb.MovieDetails{ID: 10, Title: "Arrival", PosterPath: "/arrival.jpg"}

	tests := []struct {
		name      string
		repoMovie *domains.Movie
		repoErr   error
		tmdbMovie *tmdb.MovieDetails
		tmdbErr   error
		clientErr error
		saveErr   error
		wantErr   error
		wantMovie *domains.Movie
		wantTMDB  bool
		wantSave  bool
	}{
		{
			name:      "returns cached movie",
			repoMovie: cachedMovie,
			wantMovie: cachedMovie,
		},
		{
			name:      "fetches and caches from tmdb",
			repoErr:   repositories.ErrMovieNotFound,
			tmdbMovie: tmdbMovie,
			wantMovie: &domains.Movie{TMDBID: 10, Title: "Arrival", PosterPath: "/arrival.jpg"},
			wantTMDB:  true,
			wantSave:  true,
		},
		{
			name:    "repository error",
			repoErr: genericErr,
			wantErr: genericErr,
		},
		{
			name:     "tmdb not found",
			repoErr:  repositories.ErrMovieNotFound,
			tmdbErr:  tmdb.Error{StatusCode: tmdbNotFoundStatusCode},
			wantErr:  repositories.ErrMovieNotFound,
			wantTMDB: true,
		},
		{
			name:     "tmdb error",
			repoErr:  repositories.ErrMovieNotFound,
			tmdbErr:  genericErr,
			wantErr:  genericErr,
			wantTMDB: true,
		},
		{
			name:      "tmdb client error",
			repoErr:   repositories.ErrMovieNotFound,
			clientErr: genericErr,
			wantErr:   genericErr,
			wantTMDB:  true,
		},
		{
			name:      "save error",
			repoErr:   repositories.ErrMovieNotFound,
			tmdbMovie: tmdbMovie,
			saveErr:   genericErr,
			wantErr:   genericErr,
			wantTMDB:  true,
			wantSave:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmdbCalls := 0
			saveCalls := 0
			var savedMovie *domains.Movie

			movieRepo := &testutil.FakeMovieRepository{
				GetMovieByIDFunc: func(_ context.Context, movieID int64) (*domains.Movie, error) {
					if movieID != 10 {
						t.Fatalf("GetMovieByID(%d); want 10", movieID)
					}
					return tt.repoMovie, tt.repoErr
				},
				SaveMovieFunc: func(_ context.Context, movie *domains.Movie) error {
					saveCalls++
					savedMovie = movie
					return tt.saveErr
				},
			}
			client := &fakeTMDBMovieGetter{movie: tt.tmdbMovie, err: tt.tmdbErr}
			svc := &movieService{
				movieRepository: movieRepo,
				newTMDBClient: func(context.Context) (tmdbMovieGetter, error) {
					tmdbCalls++
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return client, nil
				},
			}

			movie, err := svc.GetMovieByID(context.Background(), 10)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetMovieByID() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetMovieByID() error = %v; want nil", err)
			}
			if got := tmdbCalls > 0; got != tt.wantTMDB {
				t.Fatalf("TMDB client called = %v; want %v", got, tt.wantTMDB)
			}
			if got := saveCalls > 0; got != tt.wantSave {
				t.Fatalf("SaveMovie called = %v; want %v", got, tt.wantSave)
			}
			if tt.wantMovie == nil {
				if movie != nil {
					t.Fatalf("GetMovieByID() = %v; want nil", movie)
				}
				return
			}
			if movie == nil {
				t.Fatal("GetMovieByID() = nil; want movie")
			}
			if movie.TMDBID != tt.wantMovie.TMDBID || movie.Title != tt.wantMovie.Title || movie.PosterPath != tt.wantMovie.PosterPath {
				t.Fatalf("GetMovieByID() = %+v; want %+v", movie, tt.wantMovie)
			}
			if tt.wantSave {
				if savedMovie != movie {
					t.Fatalf("SaveMovie() called with %+v; want %+v", savedMovie, movie)
				}
			}
		})
	}
}

func TestGetMovieByIDPropagatesContextToTMDBClient(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request")

	movieRepo := &testutil.FakeMovieRepository{
		GetMovieByIDFunc: func(context.Context, int64) (*domains.Movie, error) {
			return nil, repositories.ErrMovieNotFound
		},
		SaveMovieFunc: func(context.Context, *domains.Movie) error {
			return nil
		},
	}
	var gotContext context.Context
	svc := &movieService{
		movieRepository: movieRepo,
		newTMDBClient: func(ctx context.Context) (tmdbMovieGetter, error) {
			gotContext = ctx
			return &fakeTMDBMovieGetter{movie: &tmdb.MovieDetails{ID: 10}}, nil
		},
	}

	if _, err := svc.GetMovieByID(ctx, 10); err != nil {
		t.Fatalf("GetMovieByID() error = %v; want nil", err)
	}
	if gotContext == nil || gotContext.Value(contextKey{}) != "request" {
		t.Fatal("GetMovieByID() did not propagate the request context to the TMDB client")
	}
}
