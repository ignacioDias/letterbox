package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
)

type fakeMovieService struct {
	getMovieByIDFunc      func(ctx context.Context, movieID int32) (*domains.Movie, error)
	getTrendingMoviesFunc func(ctx context.Context, page string) (*domains.TrendingMovies, error)
}

func (f *fakeMovieService) GetMovieByID(ctx context.Context, movieID int32) (*domains.Movie, error) {
	return f.getMovieByIDFunc(ctx, movieID)
}

func (f *fakeMovieService) GetTrendingMovies(ctx context.Context, page string) (*domains.TrendingMovies, error) {
	return f.getTrendingMoviesFunc(ctx, page)
}

func TestGetMovieByIDHandler(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	wantMovie := &domains.Movie{TMDBID: 10, Title: "Arrival", PosterPath: "/arrival.jpg"}

	tests := []struct {
		name       string
		movieID    string
		movie      *domains.Movie
		movieErr   error
		wantStatus int
		wantCalled bool
	}{
		{name: "success", movieID: "10", movie: wantMovie, wantStatus: http.StatusOK, wantCalled: true},
		{name: "invalid movie ID", movieID: "abc", wantStatus: http.StatusBadRequest},
		{name: "zero movie ID", movieID: "0", wantStatus: http.StatusBadRequest},
		{name: "negative movie ID", movieID: "-1", wantStatus: http.StatusBadRequest},
		{name: "movie ID above int32", movieID: "2147483648", wantStatus: http.StatusBadRequest},
		{name: "movie not found", movieID: "10", movieErr: repositories.ErrMovieNotFound, wantStatus: http.StatusNotFound, wantCalled: true},
		{name: "service error", movieID: "10", movieErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotID int32
			movieService := &fakeMovieService{
				getMovieByIDFunc: func(_ context.Context, movieID int32) (*domains.Movie, error) {
					called = true
					gotID = movieID
					return tt.movie, tt.movieErr
				},
			}
			handler := NewMovieHandler(movieService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+tt.movieID, nil)
			req.SetPathValue("movie_id", tt.movieID)
			rec := httptest.NewRecorder()

			handler.GetMovieByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v; want %v", called, tt.wantCalled)
			}
			if tt.wantCalled && gotID != 10 {
				t.Fatalf("service called with movieID %d; want 10", gotID)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.Movie
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.TMDBID != wantMovie.TMDBID || got.Title != wantMovie.Title || got.PosterPath != wantMovie.PosterPath {
				t.Fatalf("unexpected response: %+v", got)
			}
		})
	}
}

func TestGetTrendingMoviesHandler(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	want := &domains.TrendingMovies{
		Page:         2,
		TotalPages:   20,
		TotalResults: 400,
		Results: []*domains.Movie{
			{TMDBID: 10, Title: "Arrival", PosterPath: "/arrival.jpg"},
			{TMDBID: 11, Title: "Dune", PosterPath: "/dune.jpg"},
		},
	}

	tests := []struct {
		name       string
		query      string
		trending   *domains.TrendingMovies
		serviceErr error
		wantStatus int
		wantCalled bool
		wantPage   string
	}{
		{name: "success", query: "?page=2", trending: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: "2"},
		{name: "omitted page", trending: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: ""},
		{name: "service error", query: "?page=2", serviceErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true, wantPage: "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotPage string
			movieService := &fakeMovieService{
				getTrendingMoviesFunc: func(_ context.Context, page string) (*domains.TrendingMovies, error) {
					called = true
					gotPage = page
					return tt.trending, tt.serviceErr
				},
			}
			handler := NewMovieHandler(movieService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/trending"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.GetTrendingMovies(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v; want %v", called, tt.wantCalled)
			}
			if gotPage != tt.wantPage {
				t.Fatalf("service called with page %q; want %q", gotPage, tt.wantPage)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.TrendingMovies
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Page != want.Page || got.TotalPages != want.TotalPages || got.TotalResults != want.TotalResults {
				t.Fatalf("unexpected metadata: %+v", got)
			}
			if len(got.Results) != len(want.Results) {
				t.Fatalf("returned %d movies; want %d", len(got.Results), len(want.Results))
			}
			for i := range got.Results {
				if got.Results[i].TMDBID != want.Results[i].TMDBID || got.Results[i].Title != want.Results[i].Title {
					t.Fatalf("unexpected result %d: %+v", i, got.Results[i])
				}
			}
		})
	}
}
