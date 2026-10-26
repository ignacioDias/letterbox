package handlers

import (
	"errors"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
)

type MovieHandler interface {
	GetMovieByID(w http.ResponseWriter, r *http.Request)
	GetTrendingMovies(w http.ResponseWriter, r *http.Request)
}

type movieHandler struct {
	movieService services.MovieService
}

func NewMovieHandler(movieService services.MovieService) MovieHandler {
	return &movieHandler{
		movieService: movieService,
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

func (h *movieHandler) GetTrendingMovies(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	trending, err := h.movieService.GetTrendingMovies(r.Context(), page)
	if errors.Is(err, services.ErrInvalidPage) {
		http.Error(w, "Invalid page parameter", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Error getting trending movies: %v", err)
		http.Error(w, "Error getting trending movies", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, trending)
}
