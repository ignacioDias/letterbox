package services

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type trendingCall struct {
	mediaType  string
	timeWindow string
	options    map[string]string
}

type similarCall struct {
	movieID int
	options map[string]string
}

type recommendationCall struct {
	movieID int
	options map[string]string
}

type searchCall struct {
	query   string
	options map[string]string
}

type fakeTMDBMovieGetter struct {
	movie               *tmdb.MovieDetails
	err                 error
	trending            *tmdb.Trending
	trendingErr         error
	trendingCalls       []trendingCall
	similar             *tmdb.MovieSimilar
	similarErr          error
	similarCalls        []similarCall
	recommendations     *tmdb.MovieRecommendations
	recommendationsErr  error
	recommendationCalls []recommendationCall
	search              *tmdb.SearchMovies
	searchErr           error
	searchCalls         []searchCall
}

func (f *fakeTMDBMovieGetter) GetMovieDetails(int, map[string]string) (*tmdb.MovieDetails, error) {
	return f.movie, f.err
}

func (f *fakeTMDBMovieGetter) GetTrending(mediaType, timeWindow string, options map[string]string) (*tmdb.Trending, error) {
	f.trendingCalls = append(f.trendingCalls, trendingCall{mediaType: mediaType, timeWindow: timeWindow, options: options})
	return f.trending, f.trendingErr
}

func (f *fakeTMDBMovieGetter) GetMovieSimilar(movieID int, options map[string]string) (*tmdb.MovieSimilar, error) {
	f.similarCalls = append(f.similarCalls, similarCall{movieID: movieID, options: options})
	return f.similar, f.similarErr
}

func (f *fakeTMDBMovieGetter) GetMovieRecommendations(movieID int, options map[string]string) (*tmdb.MovieRecommendations, error) {
	f.recommendationCalls = append(f.recommendationCalls, recommendationCall{movieID: movieID, options: options})
	return f.recommendations, f.recommendationsErr
}

func (f *fakeTMDBMovieGetter) GetSearchMovies(query string, options map[string]string) (*tmdb.SearchMovies, error) {
	f.searchCalls = append(f.searchCalls, searchCall{query: query, options: options})
	return f.search, f.searchErr
}

func mustParseTrending(t *testing.T, body string) *tmdb.Trending {
	t.Helper()
	var trending tmdb.Trending
	if err := json.Unmarshal([]byte(body), &trending); err != nil {
		t.Fatalf("parse trending fixture: %v", err)
	}
	return &trending
}

func mustParseSimilar(t *testing.T, body string) *tmdb.MovieSimilar {
	t.Helper()
	var similar tmdb.MovieSimilar
	if err := json.Unmarshal([]byte(body), &similar); err != nil {
		t.Fatalf("parse similar fixture: %v", err)
	}
	return &similar
}

func mustParseRecommendations(t *testing.T, body string) *tmdb.MovieRecommendations {
	t.Helper()
	var recommendations tmdb.MovieRecommendations
	if err := json.Unmarshal([]byte(body), &recommendations); err != nil {
		t.Fatalf("parse recommendations fixture: %v", err)
	}
	return &recommendations
}

func mustParseSearch(t *testing.T, body string) *tmdb.SearchMovies {
	t.Helper()
	var search tmdb.SearchMovies
	if err := json.Unmarshal([]byte(body), &search); err != nil {
		t.Fatalf("parse search fixture: %v", err)
	}
	return &search
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
				GetMovieByIDFunc: func(_ context.Context, movieID int32) (*domains.Movie, error) {
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
		GetMovieByIDFunc: func(context.Context, int32) (*domains.Movie, error) {
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

func TestGetTrendingMovies(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	trending := mustParseTrending(t, `{
		"page": 2,
		"total_pages": 20,
		"total_results": 400,
		"results": [
			{"id": 10, "title": "Arrival", "poster_path": "/arrival.jpg"},
			{"id": 11, "title": "Dune", "poster_path": "/dune.jpg"}
		]
	}`)
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
		name        string
		page        string
		trending    *tmdb.Trending
		trendingErr error
		clientErr   error
		saveErr     error
		wantErr     error
		wantTMDB    bool
		wantPage    string
		wantSaved   []int32
		want        *domains.PaginatedMovies
	}{
		{
			name:      "returns trending movies with pagination metadata",
			page:      "2",
			trending:  trending,
			wantTMDB:  true,
			wantPage:  "2",
			wantSaved: []int32{10, 11},
			want:      want,
		},
		{
			name:      "omitted page defaults to page 1",
			page:      "",
			trending:  trending,
			wantTMDB:  true,
			wantPage:  "1",
			wantSaved: []int32{10, 11},
			want:      want,
		},
		{
			name:      "maximum page is accepted",
			page:      "500",
			trending:  trending,
			wantTMDB:  true,
			wantPage:  "500",
			wantSaved: []int32{10, 11},
			want:      want,
		},
		{name: "non-numeric page", page: "abc", wantErr: ErrInvalidPage},
		{name: "zero page", page: "0", wantErr: ErrInvalidPage},
		{name: "negative page", page: "-1", wantErr: ErrInvalidPage},
		{name: "page above maximum", page: "501", wantErr: ErrInvalidPage},
		{name: "fractional page", page: "1.5", wantErr: ErrInvalidPage},
		{
			name:      "tmdb client error",
			page:      "1",
			clientErr: genericErr,
			wantErr:   genericErr,
			wantTMDB:  false,
		},
		{
			name:        "tmdb error",
			page:        "1",
			trendingErr: genericErr,
			wantErr:     genericErr,
			wantTMDB:    true,
			wantPage:    "1",
		},
		{
			name:      "save error does not fail the request",
			page:      "1",
			trending:  trending,
			saveErr:   genericErr,
			wantTMDB:  true,
			wantPage:  "1",
			wantSaved: []int32{10, 11},
			want:      want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved []int32
			movieRepo := &testutil.FakeMovieRepository{
				SaveMovieFunc: func(_ context.Context, movie *domains.Movie) error {
					saved = append(saved, movie.TMDBID)
					return tt.saveErr
				},
			}
			client := &fakeTMDBMovieGetter{trending: tt.trending, trendingErr: tt.trendingErr}
			svc := &movieService{
				movieRepository: movieRepo,
				newTMDBClient: func(context.Context) (tmdbMovieGetter, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return client, nil
				},
			}

			got, err := svc.GetTrendingMovies(context.Background(), tt.page)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetTrendingMovies() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetTrendingMovies() error = %v; want nil", err)
			}

			if tt.wantTMDB {
				if len(client.trendingCalls) != 1 {
					t.Fatalf("TMDB GetTrending called %d times; want 1", len(client.trendingCalls))
				}
				call := client.trendingCalls[0]
				if call.mediaType != "movie" || call.timeWindow != "day" {
					t.Fatalf("TMDB GetTrending(%q, %q); want movie, day", call.mediaType, call.timeWindow)
				}
				if call.options["page"] != tt.wantPage {
					t.Fatalf("TMDB page option = %q; want %q", call.options["page"], tt.wantPage)
				}
			} else if len(client.trendingCalls) != 0 {
				t.Fatalf("TMDB GetTrending called %d times; want 0", len(client.trendingCalls))
			}

			if !slices.Equal(saved, tt.wantSaved) {
				t.Fatalf("saved movie IDs = %v; want %v", saved, tt.wantSaved)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("GetTrendingMovies() = %+v; want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetTrendingMovies() = nil; want result")
			}
			if got.Page != tt.want.Page || got.TotalPages != tt.want.TotalPages || got.TotalResults != tt.want.TotalResults {
				t.Fatalf("GetTrendingMovies() metadata = page %d, total pages %d, total results %d; want %d, %d, %d",
					got.Page, got.TotalPages, got.TotalResults, tt.want.Page, tt.want.TotalPages, tt.want.TotalResults)
			}
			if len(got.Results) != len(tt.want.Results) {
				t.Fatalf("GetTrendingMovies() returned %d movies; want %d", len(got.Results), len(tt.want.Results))
			}
			for i := range got.Results {
				if got.Results[i].TMDBID != tt.want.Results[i].TMDBID ||
					got.Results[i].Title != tt.want.Results[i].Title ||
					got.Results[i].PosterPath != tt.want.Results[i].PosterPath {
					t.Fatalf("GetTrendingMovies() result %d = %+v; want %+v", i, got.Results[i], tt.want.Results[i])
				}
			}
		})
	}
}

func TestGetSimilarMovies(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	similar := mustParseSimilar(t, `{
		"page": 3,
		"total_pages": 10,
		"total_results": 200,
		"results": [
			{"id": 20, "title": "Interstellar", "poster_path": "/interstellar.jpg"},
			{"id": 21, "title": "Contact", "poster_path": "/contact.jpg"}
		]
	}`)
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
		movieID    int32
		page       string
		similar    *tmdb.MovieSimilar
		similarErr error
		clientErr  error
		saveErr    error
		wantErr    error
		wantTMDB   bool
		wantPage   string
		wantSaved  []int32
		want       *domains.PaginatedMovies
	}{
		{
			name:      "returns similar movies with pagination metadata",
			movieID:   550,
			page:      "3",
			similar:   similar,
			wantTMDB:  true,
			wantPage:  "3",
			wantSaved: []int32{20, 21},
			want:      want,
		},
		{
			name:      "omitted page defaults to page 1",
			movieID:   550,
			page:      "",
			similar:   similar,
			wantTMDB:  true,
			wantPage:  "1",
			wantSaved: []int32{20, 21},
			want:      want,
		},
		{name: "non-numeric page", movieID: 550, page: "abc", wantErr: ErrInvalidPage},
		{name: "zero page", movieID: 550, page: "0", wantErr: ErrInvalidPage},
		{name: "page above maximum", movieID: 550, page: "501", wantErr: ErrInvalidPage},
		{
			name:      "tmdb client error",
			movieID:   550,
			page:      "1",
			clientErr: genericErr,
			wantErr:   genericErr,
		},
		{
			name:       "movie not found in tmdb",
			movieID:    550,
			page:       "1",
			similarErr: tmdb.Error{StatusCode: tmdbNotFoundStatusCode},
			wantErr:    repositories.ErrMovieNotFound,
			wantTMDB:   true,
			wantPage:   "1",
		},
		{
			name:       "tmdb error",
			movieID:    550,
			page:       "1",
			similarErr: genericErr,
			wantErr:    genericErr,
			wantTMDB:   true,
			wantPage:   "1",
		},
		{
			name:      "save error does not fail the request",
			movieID:   550,
			page:      "1",
			similar:   similar,
			saveErr:   genericErr,
			wantTMDB:  true,
			wantPage:  "1",
			wantSaved: []int32{20, 21},
			want:      want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved []int32
			movieRepo := &testutil.FakeMovieRepository{
				SaveMovieFunc: func(_ context.Context, movie *domains.Movie) error {
					saved = append(saved, movie.TMDBID)
					return tt.saveErr
				},
			}
			client := &fakeTMDBMovieGetter{similar: tt.similar, similarErr: tt.similarErr}
			svc := &movieService{
				movieRepository: movieRepo,
				newTMDBClient: func(context.Context) (tmdbMovieGetter, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return client, nil
				},
			}

			got, err := svc.GetSimilarMovies(context.Background(), tt.movieID, tt.page)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetSimilarMovies() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetSimilarMovies() error = %v; want nil", err)
			}

			if tt.wantTMDB {
				if len(client.similarCalls) != 1 {
					t.Fatalf("TMDB GetMovieSimilar called %d times; want 1", len(client.similarCalls))
				}
				if client.similarCalls[0].movieID != int(tt.movieID) {
					t.Fatalf("TMDB GetMovieSimilar(%d); want %d", client.similarCalls[0].movieID, tt.movieID)
				}
				if got := client.similarCalls[0].options["page"]; got != tt.wantPage {
					t.Fatalf("TMDB page option = %q; want %q", got, tt.wantPage)
				}
			} else if len(client.similarCalls) != 0 {
				t.Fatalf("TMDB GetMovieSimilar called %d times; want 0", len(client.similarCalls))
			}

			if !slices.Equal(saved, tt.wantSaved) {
				t.Fatalf("saved movie IDs = %v; want %v", saved, tt.wantSaved)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("GetSimilarMovies() = %+v; want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetSimilarMovies() = nil; want result")
			}
			if got.Page != tt.want.Page || got.TotalPages != tt.want.TotalPages || got.TotalResults != tt.want.TotalResults {
				t.Fatalf("GetSimilarMovies() metadata = page %d, total pages %d, total results %d; want %d, %d, %d",
					got.Page, got.TotalPages, got.TotalResults, tt.want.Page, tt.want.TotalPages, tt.want.TotalResults)
			}
			if len(got.Results) != len(tt.want.Results) {
				t.Fatalf("GetSimilarMovies() returned %d movies; want %d", len(got.Results), len(tt.want.Results))
			}
			for i := range got.Results {
				if got.Results[i].TMDBID != tt.want.Results[i].TMDBID ||
					got.Results[i].Title != tt.want.Results[i].Title ||
					got.Results[i].PosterPath != tt.want.Results[i].PosterPath {
					t.Fatalf("GetSimilarMovies() result %d = %+v; want %+v", i, got.Results[i], tt.want.Results[i])
				}
			}
		})
	}
}

func TestGetRecommendationsForMovieByID(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	recommendations := mustParseRecommendations(t, `{
		"page": 4,
		"total_pages": 8,
		"total_results": 160,
		"results": [
			{"id": 30, "title": "Moon", "poster_path": "/moon.jpg"},
			{"id": 31, "title": "Solaris", "poster_path": "/solaris.jpg"}
		]
	}`)
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
		name            string
		movieID         int32
		page            string
		recommendations *tmdb.MovieRecommendations
		tmdbErr         error
		clientErr       error
		saveErr         error
		wantErr         error
		wantTMDB        bool
		wantPage        string
		wantSaved       []int32
		want            *domains.PaginatedMovies
	}{
		{
			name:            "returns recommendations with pagination metadata",
			movieID:         550,
			page:            "4",
			recommendations: recommendations,
			wantTMDB:        true,
			wantPage:        "4",
			wantSaved:       []int32{30, 31},
			want:            want,
		},
		{
			name:            "omitted page defaults to page 1",
			movieID:         550,
			page:            "",
			recommendations: recommendations,
			wantTMDB:        true,
			wantPage:        "1",
			wantSaved:       []int32{30, 31},
			want:            want,
		},
		{
			name:            "maximum page is accepted",
			movieID:         550,
			page:            "500",
			recommendations: recommendations,
			wantTMDB:        true,
			wantPage:        "500",
			wantSaved:       []int32{30, 31},
			want:            want,
		},
		{
			name:    "empty results",
			movieID: 550,
			page:    "1",
			recommendations: mustParseRecommendations(t, `{
				"page": 1,
				"total_pages": 0,
				"total_results": 0,
				"results": []
			}`),
			wantTMDB: true,
			wantPage: "1",
			want: &domains.PaginatedMovies{
				Page:         1,
				TotalPages:   0,
				TotalResults: 0,
				Results:      []*domains.Movie{},
			},
		},
		{name: "non-numeric page", movieID: 550, page: "abc", wantErr: ErrInvalidPage},
		{name: "zero page", movieID: 550, page: "0", wantErr: ErrInvalidPage},
		{name: "page above maximum", movieID: 550, page: "501", wantErr: ErrInvalidPage},
		{
			name:      "tmdb client error",
			movieID:   550,
			page:      "1",
			clientErr: genericErr,
			wantErr:   genericErr,
		},
		{
			name:     "movie not found in tmdb",
			movieID:  550,
			page:     "1",
			tmdbErr:  tmdb.Error{StatusCode: tmdbNotFoundStatusCode},
			wantErr:  repositories.ErrMovieNotFound,
			wantTMDB: true,
			wantPage: "1",
		},
		{
			name:     "tmdb error",
			movieID:  550,
			page:     "1",
			tmdbErr:  genericErr,
			wantErr:  genericErr,
			wantTMDB: true,
			wantPage: "1",
		},
		{
			name:            "save error does not fail the request",
			movieID:         550,
			page:            "1",
			recommendations: recommendations,
			saveErr:         genericErr,
			wantTMDB:        true,
			wantPage:        "1",
			wantSaved:       []int32{30, 31},
			want:            want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved []int32
			movieRepo := &testutil.FakeMovieRepository{
				SaveMovieFunc: func(_ context.Context, movie *domains.Movie) error {
					saved = append(saved, movie.TMDBID)
					return tt.saveErr
				},
			}
			client := &fakeTMDBMovieGetter{recommendations: tt.recommendations, recommendationsErr: tt.tmdbErr}
			svc := &movieService{
				movieRepository: movieRepo,
				newTMDBClient: func(context.Context) (tmdbMovieGetter, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return client, nil
				},
			}

			got, err := svc.GetRecommendationsForMovieByID(context.Background(), tt.movieID, tt.page)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetRecommendationsForMovieByID() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetRecommendationsForMovieByID() error = %v; want nil", err)
			}

			if tt.wantTMDB {
				if len(client.recommendationCalls) != 1 {
					t.Fatalf("TMDB GetMovieRecommendations called %d times; want 1", len(client.recommendationCalls))
				}
				if client.recommendationCalls[0].movieID != int(tt.movieID) {
					t.Fatalf("TMDB GetMovieRecommendations(%d); want %d", client.recommendationCalls[0].movieID, tt.movieID)
				}
				if got := client.recommendationCalls[0].options["page"]; got != tt.wantPage {
					t.Fatalf("TMDB page option = %q; want %q", got, tt.wantPage)
				}
			} else if len(client.recommendationCalls) != 0 {
				t.Fatalf("TMDB GetMovieRecommendations called %d times; want 0", len(client.recommendationCalls))
			}

			if !slices.Equal(saved, tt.wantSaved) {
				t.Fatalf("saved movie IDs = %v; want %v", saved, tt.wantSaved)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("GetRecommendationsForMovieByID() = %+v; want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetRecommendationsForMovieByID() = nil; want result")
			}
			if got.Page != tt.want.Page || got.TotalPages != tt.want.TotalPages || got.TotalResults != tt.want.TotalResults {
				t.Fatalf("GetRecommendationsForMovieByID() metadata = page %d, total pages %d, total results %d; want %d, %d, %d",
					got.Page, got.TotalPages, got.TotalResults, tt.want.Page, tt.want.TotalPages, tt.want.TotalResults)
			}
			if len(got.Results) != len(tt.want.Results) {
				t.Fatalf("GetRecommendationsForMovieByID() returned %d movies; want %d", len(got.Results), len(tt.want.Results))
			}
			for i := range got.Results {
				if got.Results[i].TMDBID != tt.want.Results[i].TMDBID ||
					got.Results[i].Title != tt.want.Results[i].Title ||
					got.Results[i].PosterPath != tt.want.Results[i].PosterPath {
					t.Fatalf("GetRecommendationsForMovieByID() result %d = %+v; want %+v", i, got.Results[i], tt.want.Results[i])
				}
			}
		})
	}
}

func TestSearchMovies(t *testing.T) {
	genericErr := errors.New("tmdb unavailable")
	search := mustParseSearch(t, `{
		"page": 2,
		"total_pages": 3,
		"total_results": 45,
		"results": [
			{"id": 20, "title": "Arrival", "poster_path": "/arrival.jpg"},
			{"id": 21, "title": "Contact", "poster_path": "/contact.jpg"}
		]
	}`)
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
		name      string
		search    string
		page      string
		result    *tmdb.SearchMovies
		tmdbErr   error
		clientErr error
		saveErr   error
		wantErr   error
		wantTMDB  bool
		wantQuery string
		wantPage  string
		wantSaved []int32
		want      *domains.PaginatedMovies
	}{
		{
			name:      "returns search results with pagination metadata",
			search:    "arrival",
			page:      "2",
			result:    search,
			wantTMDB:  true,
			wantQuery: "arrival",
			wantPage:  "2",
			wantSaved: []int32{20, 21},
			want:      want,
		},
		{
			name:      "trims query and defaults omitted page to page 1",
			search:    "  arrival  ",
			page:      "",
			result:    search,
			wantTMDB:  true,
			wantQuery: "arrival",
			wantPage:  "1",
			wantSaved: []int32{20, 21},
			want:      want,
		},
		{
			name:    "empty query",
			search:  "",
			page:    "1",
			wantErr: ErrInvalidSearchQuery,
		},
		{
			name:    "whitespace-only query",
			search:  "   ",
			page:    "1",
			wantErr: ErrInvalidSearchQuery,
		},
		{name: "non-numeric page", search: "arrival", page: "abc", wantErr: ErrInvalidPage},
		{name: "zero page", search: "arrival", page: "0", wantErr: ErrInvalidPage},
		{name: "page above maximum", search: "arrival", page: "501", wantErr: ErrInvalidPage},
		{
			name:      "empty results",
			search:    "arrival",
			page:      "1",
			result:    mustParseSearch(t, `{"page": 1, "total_pages": 0, "total_results": 0, "results": []}`),
			wantTMDB:  true,
			wantQuery: "arrival",
			wantPage:  "1",
			want: &domains.PaginatedMovies{
				Page:         1,
				TotalPages:   0,
				TotalResults: 0,
				Results:      []*domains.Movie{},
			},
		},
		{
			name:      "tmdb client error",
			search:    "arrival",
			page:      "1",
			clientErr: genericErr,
			wantErr:   genericErr,
		},
		{
			name:      "tmdb error",
			search:    "arrival",
			page:      "1",
			tmdbErr:   genericErr,
			wantErr:   genericErr,
			wantTMDB:  true,
			wantQuery: "arrival",
			wantPage:  "1",
		},
		{
			name:      "save error does not fail the request",
			search:    "arrival",
			page:      "1",
			result:    search,
			saveErr:   genericErr,
			wantTMDB:  true,
			wantQuery: "arrival",
			wantPage:  "1",
			wantSaved: []int32{20, 21},
			want:      want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved []int32
			movieRepo := &testutil.FakeMovieRepository{
				SaveMovieFunc: func(_ context.Context, movie *domains.Movie) error {
					saved = append(saved, movie.TMDBID)
					return tt.saveErr
				},
			}
			client := &fakeTMDBMovieGetter{search: tt.result, searchErr: tt.tmdbErr}
			svc := &movieService{
				movieRepository: movieRepo,
				newTMDBClient: func(context.Context) (tmdbMovieGetter, error) {
					if tt.clientErr != nil {
						return nil, tt.clientErr
					}
					return client, nil
				},
			}

			got, err := svc.SearchMovies(context.Background(), tt.search, tt.page)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SearchMovies() error = %v; want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("SearchMovies() error = %v; want nil", err)
			}

			if tt.wantTMDB {
				if len(client.searchCalls) != 1 {
					t.Fatalf("TMDB GetSearchMovies called %d times; want 1", len(client.searchCalls))
				}
				if client.searchCalls[0].query != tt.wantQuery {
					t.Fatalf("TMDB GetSearchMovies(%q); want %q", client.searchCalls[0].query, tt.wantQuery)
				}
				if got := client.searchCalls[0].options["page"]; got != tt.wantPage {
					t.Fatalf("TMDB page option = %q; want %q", got, tt.wantPage)
				}
			} else if len(client.searchCalls) != 0 {
				t.Fatalf("TMDB GetSearchMovies called %d times; want 0", len(client.searchCalls))
			}

			if !slices.Equal(saved, tt.wantSaved) {
				t.Fatalf("saved movie IDs = %v; want %v", saved, tt.wantSaved)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("SearchMovies() = %+v; want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("SearchMovies() = nil; want result")
			}
			if got.Page != tt.want.Page || got.TotalPages != tt.want.TotalPages || got.TotalResults != tt.want.TotalResults {
				t.Fatalf("SearchMovies() metadata = page %d, total pages %d, total results %d; want %d, %d, %d",
					got.Page, got.TotalPages, got.TotalResults, tt.want.Page, tt.want.TotalPages, tt.want.TotalResults)
			}
			if len(got.Results) != len(tt.want.Results) {
				t.Fatalf("SearchMovies() returned %d movies; want %d", len(got.Results), len(tt.want.Results))
			}
			for i := range got.Results {
				if got.Results[i].TMDBID != tt.want.Results[i].TMDBID ||
					got.Results[i].Title != tt.want.Results[i].Title ||
					got.Results[i].PosterPath != tt.want.Results[i].PosterPath {
					t.Fatalf("SearchMovies() result %d = %+v; want %+v", i, got.Results[i], tt.want.Results[i])
				}
			}
		})
	}
}
