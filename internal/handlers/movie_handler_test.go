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
	getMovieByIDFunc func(ctx context.Context, movieID int64) (*domains.Movie, error)
}

func (f *fakeMovieService) GetMovieByID(ctx context.Context, movieID int64) (*domains.Movie, error) {
	return f.getMovieByIDFunc(ctx, movieID)
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
		{name: "movie not found", movieID: "10", movieErr: repositories.ErrMovieNotFound, wantStatus: http.StatusNotFound, wantCalled: true},
		{name: "service error", movieID: "10", movieErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotID int64
			movieService := &fakeMovieService{
				getMovieByIDFunc: func(_ context.Context, movieID int64) (*domains.Movie, error) {
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
