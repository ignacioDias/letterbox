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
	movieID, err := getInt64PathValue(r, "movie_id")
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
