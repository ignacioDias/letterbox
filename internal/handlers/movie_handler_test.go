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
	"letterbox/internal/services"
)

type fakeMovieService struct {
	getMovieByIDFunc                   func(ctx context.Context, movieID int32) (*domains.Movie, error)
	getTrendingMoviesFunc              func(ctx context.Context, page string) (*domains.PaginatedMovies, error)
	getSimilarMoviesFunc               func(ctx context.Context, movieID int32, page string) (*domains.PaginatedMovies, error)
	getRecommendationsForMovieByIDFunc func(ctx context.Context, movieID int32, page string) (*domains.PaginatedMovies, error)
	searchMoviesFunc                   func(ctx context.Context, search string, page string) (*domains.PaginatedMovies, error)
}

func (f *fakeMovieService) GetMovieByID(ctx context.Context, movieID int32) (*domains.Movie, error) {
	return f.getMovieByIDFunc(ctx, movieID)
}

func (f *fakeMovieService) GetTrendingMovies(ctx context.Context, page string) (*domains.PaginatedMovies, error) {
	return f.getTrendingMoviesFunc(ctx, page)
}

func (f *fakeMovieService) GetSimilarMovies(ctx context.Context, movieID int32, page string) (*domains.PaginatedMovies, error) {
	return f.getSimilarMoviesFunc(ctx, movieID, page)
}

func (f *fakeMovieService) GetRecommendationsForMovieByID(ctx context.Context, movieID int32, page string) (*domains.PaginatedMovies, error) {
	return f.getRecommendationsForMovieByIDFunc(ctx, movieID, page)
}

func (f *fakeMovieService) SearchMovies(ctx context.Context, search string, page string) (*domains.PaginatedMovies, error) {
	return f.searchMoviesFunc(ctx, search, page)
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
	want := &domains.PaginatedMovies{
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
		trending   *domains.PaginatedMovies
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
				getTrendingMoviesFunc: func(_ context.Context, page string) (*domains.PaginatedMovies, error) {
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
			var got domains.PaginatedMovies
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

func TestGetSimilarMoviesHandler(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	want := &domains.PaginatedMovies{
		Page:         3,
		TotalPages:   10,
		TotalResults: 200,
		Results: []*domains.Movie{
			{TMDBID: 20, Title: "Interstellar", PosterPath: "/interstellar.jpg"},
			{TMDBID: 21, Title: "Contact", PosterPath: "/contact.jpg"},
		},
	}

	tests := []struct {
		name       string
		movieID    string
		query      string
		similar    *domains.PaginatedMovies
		serviceErr error
		wantStatus int
		wantCalled bool
		wantPage   string
	}{
		{name: "success", movieID: "550", query: "?page=3", similar: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: "3"},
		{name: "omitted page", movieID: "550", similar: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: ""},
		{name: "invalid movie ID", movieID: "abc", query: "?page=1", wantStatus: http.StatusBadRequest},
		{name: "zero movie ID", movieID: "0", wantStatus: http.StatusBadRequest},
		{name: "invalid page", movieID: "550", query: "?page=abc", serviceErr: services.ErrInvalidPage, wantStatus: http.StatusBadRequest, wantCalled: true, wantPage: "abc"},
		{name: "movie not found", movieID: "550", query: "?page=1", serviceErr: repositories.ErrMovieNotFound, wantStatus: http.StatusNotFound, wantCalled: true, wantPage: "1"},
		{name: "service error", movieID: "550", query: "?page=1", serviceErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true, wantPage: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotMovieID int32
			var gotPage string
			movieService := &fakeMovieService{
				getSimilarMoviesFunc: func(_ context.Context, movieID int32, page string) (*domains.PaginatedMovies, error) {
					called = true
					gotMovieID = movieID
					gotPage = page
					return tt.similar, tt.serviceErr
				},
			}
			handler := NewMovieHandler(movieService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+tt.movieID+"/similar"+tt.query, nil)
			req.SetPathValue("movie_id", tt.movieID)
			rec := httptest.NewRecorder()

			handler.GetSimilarMovies(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v; want %v", called, tt.wantCalled)
			}
			if tt.wantCalled {
				if gotMovieID != 550 {
					t.Fatalf("service called with movieID %d; want 550", gotMovieID)
				}
				if gotPage != tt.wantPage {
					t.Fatalf("service called with page %q; want %q", gotPage, tt.wantPage)
				}
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.PaginatedMovies
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

func TestGetRecommendationsForMovieByIDHandler(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	want := &domains.PaginatedMovies{
		Page:         4,
		TotalPages:   8,
		TotalResults: 160,
		Results: []*domains.Movie{
			{TMDBID: 30, Title: "Moon", PosterPath: "/moon.jpg"},
			{TMDBID: 31, Title: "Solaris", PosterPath: "/solaris.jpg"},
		},
	}

	tests := []struct {
		name       string
		movieID    string
		query      string
		result     *domains.PaginatedMovies
		serviceErr error
		wantStatus int
		wantCalled bool
		wantPage   string
	}{
		{name: "success", movieID: "550", query: "?page=4", result: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: "4"},
		{name: "omitted page", movieID: "550", result: want, wantStatus: http.StatusOK, wantCalled: true, wantPage: ""},
		{name: "invalid movie ID", movieID: "abc", query: "?page=1", wantStatus: http.StatusBadRequest},
		{name: "zero movie ID", movieID: "0", wantStatus: http.StatusBadRequest},
		{name: "invalid page", movieID: "550", query: "?page=abc", serviceErr: services.ErrInvalidPage, wantStatus: http.StatusBadRequest, wantCalled: true, wantPage: "abc"},
		{name: "movie not found", movieID: "550", query: "?page=1", serviceErr: repositories.ErrMovieNotFound, wantStatus: http.StatusNotFound, wantCalled: true, wantPage: "1"},
		{name: "service error", movieID: "550", query: "?page=1", serviceErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true, wantPage: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotMovieID int32
			var gotPage string
			movieService := &fakeMovieService{
				getRecommendationsForMovieByIDFunc: func(_ context.Context, movieID int32, page string) (*domains.PaginatedMovies, error) {
					called = true
					gotMovieID = movieID
					gotPage = page
					return tt.result, tt.serviceErr
				},
			}
			handler := NewMovieHandler(movieService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+tt.movieID+"/recommendations"+tt.query, nil)
			req.SetPathValue("movie_id", tt.movieID)
			rec := httptest.NewRecorder()

			handler.GetRecommendationsForMovieByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v; want %v", called, tt.wantCalled)
			}
			if tt.wantCalled {
				if gotMovieID != 550 {
					t.Fatalf("service called with movieID %d; want 550", gotMovieID)
				}
				if gotPage != tt.wantPage {
					t.Fatalf("service called with page %q; want %q", gotPage, tt.wantPage)
				}
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.PaginatedMovies
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

func TestSearchMoviesHandler(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	want := &domains.PaginatedMovies{
		Page:         2,
		TotalPages:   3,
		TotalResults: 45,
		Results: []*domains.Movie{
			{TMDBID: 20, Title: "Arrival", PosterPath: "/arrival.jpg"},
			{TMDBID: 21, Title: "Contact", PosterPath: "/contact.jpg"},
		},
	}

	tests := []struct {
		name       string
		query      string
		result     *domains.PaginatedMovies
		serviceErr error
		wantStatus int
		wantCalled bool
		wantSearch string
		wantPage   string
	}{
		{name: "success", query: "?search=arrival&page=2", result: want, wantStatus: http.StatusOK, wantCalled: true, wantSearch: "arrival", wantPage: "2"},
		{name: "omitted page", query: "?search=arrival", result: want, wantStatus: http.StatusOK, wantCalled: true, wantSearch: "arrival", wantPage: ""},
		{name: "missing search", query: "", serviceErr: services.ErrInvalidSearchQuery, wantStatus: http.StatusBadRequest, wantCalled: true},
		{name: "invalid page", query: "?search=arrival&page=abc", serviceErr: services.ErrInvalidPage, wantStatus: http.StatusBadRequest, wantCalled: true, wantSearch: "arrival", wantPage: "abc"},
		{name: "service error", query: "?search=arrival", serviceErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true, wantSearch: "arrival"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotSearch string
			var gotPage string
			movieService := &fakeMovieService{
				searchMoviesFunc: func(_ context.Context, search string, page string) (*domains.PaginatedMovies, error) {
					called = true
					gotSearch = search
					gotPage = page
					return tt.result, tt.serviceErr
				},
			}
			handler := NewMovieHandler(movieService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/search"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.SearchMovies(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v; want %v", called, tt.wantCalled)
			}
			if tt.wantCalled {
				if gotSearch != tt.wantSearch {
					t.Fatalf("service called with search %q; want %q", gotSearch, tt.wantSearch)
				}
				if gotPage != tt.wantPage {
					t.Fatalf("service called with page %q; want %q", gotPage, tt.wantPage)
				}
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.PaginatedMovies
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
