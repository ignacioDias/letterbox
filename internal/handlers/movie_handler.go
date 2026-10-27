package handlers

import (
	"context"
	"errors"
	"letterbox/internal/cache"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type MovieHandler interface {
	GetMovieByID(w http.ResponseWriter, r *http.Request)
	GetTrendingMovies(w http.ResponseWriter, r *http.Request)
	GetRecommendationsForMovieByID(w http.ResponseWriter, r *http.Request)
	GetSimilarMovies(w http.ResponseWriter, r *http.Request)
	SearchMovies(w http.ResponseWriter, r *http.Request)
}

type movieHandler struct {
	movieService services.MovieService
}

func NewMovieHandler(movieRepo repositories.MovieRepository, cache cache.Cache) MovieHandler {
	return &movieHandler{
		movieService: services.NewMovieService(movieRepo, cache),
	}
}

func (h *movieHandler) GetMovieByID(w http.ResponseWriter, r *http.Request) {
	movieID, err := getInt32PathValue(r, "movie_id")
	if err != nil {
		log.Printf("Error getting movie ID: %v", err)
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := h.movieService.GetMovieByID(r.Context(), movieID)
	if err != nil {
		log.Printf("Error getting movie: %v", err)
		if errors.Is(err, repositories.ErrMovieNotFound) {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error getting the movie", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, movie)
}

type paginatedMoviesFetcher func(ctx context.Context, page string) (*domains.PaginatedMovies, error)

type paginatedMoviesByIDFetcher func(ctx context.Context, movieID int32, page string) (*domains.PaginatedMovies, error)

func (h *movieHandler) GetTrendingMovies(w http.ResponseWriter, r *http.Request) {
	handlePaginatedMovies(w, r, h.movieService.GetTrendingMovies, "trending movies")
}

func (h *movieHandler) GetSimilarMovies(w http.ResponseWriter, r *http.Request) {
	handlePaginatedMoviesByID(w, r, h.movieService.GetSimilarMovies, "similar movies")
}

func (h *movieHandler) GetRecommendationsForMovieByID(w http.ResponseWriter, r *http.Request) {
	handlePaginatedMoviesByID(w, r, h.movieService.GetRecommendationsForMovieByID, "recommended movies")
}

func (h *movieHandler) SearchMovies(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := r.URL.Query().Get("page")
	movies, err := h.movieService.SearchMovies(r.Context(), search, page)
	if writePaginatedMoviesError(w, err, "search results") {
		return
	}
	writeJSON(w, http.StatusOK, movies)
}

func handlePaginatedMovies(w http.ResponseWriter, r *http.Request, fetch paginatedMoviesFetcher, label string) {
	movies, err := fetch(r.Context(), r.URL.Query().Get("page"))
	if writePaginatedMoviesError(w, err, label) {
		return
	}
	writeJSON(w, http.StatusOK, movies)
}

func handlePaginatedMoviesByID(w http.ResponseWriter, r *http.Request, fetch paginatedMoviesByIDFetcher, label string) {
	movieID, err := getInt32PathValue(r, "movie_id")
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}
	movies, err := fetch(r.Context(), movieID, r.URL.Query().Get("page"))
	if writePaginatedMoviesError(w, err, label) {
		return
	}
	writeJSON(w, http.StatusOK, movies)
}

func writePaginatedMoviesError(w http.ResponseWriter, err error, label string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, services.ErrInvalidPage):
		http.Error(w, "Invalid page parameter", http.StatusBadRequest)
	case errors.Is(err, services.ErrInvalidSearchQuery):
		http.Error(w, "Invalid search query", http.StatusBadRequest)
	case errors.Is(err, repositories.ErrMovieNotFound):
		http.Error(w, "Movie not found", http.StatusNotFound)
	default:
		log.Printf("Error getting %s: %v", label, err)
		http.Error(w, "Error getting "+label, http.StatusInternalServerError)
	}
	return true
}
