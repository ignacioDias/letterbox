package services

import (
	"encoding/json"
	"letterbox/internal/domains"
	"net/url"
	"strconv"
	"strings"
)

var cacheKeyVersion = "v1"

func getKeyForMovieByID(movieID int32) string {
	return "tmdb:" + cacheKeyVersion + ":movieID:" + strconv.Itoa(int(movieID))
}

func getKeyForTrendingMovies(page string) string {
	return "tmdb:" + cacheKeyVersion + ":trending:movie:day:page:" + page
}

func getKeyForRecommendations(movieID int32, page string) string {
	return "tmdb:" + cacheKeyVersion + ":recommendations:movieID:" + strconv.Itoa(int(movieID)) + ":page:" + page
}

func getKeyForSimilarMovies(movieID int32, page string) string {
	return "tmdb:" + cacheKeyVersion + ":similar:movieID:" + strconv.Itoa(int(movieID)) + ":page:" + page
}

func getKeyForSearchMovies(search, page string) string {
	return "tmdb:" + cacheKeyVersion + ":search:movie:query:" + url.QueryEscape(strings.ToLower(search)) + ":page:" + page
}

func generateMovieFromCacheResult(result string) (*domains.Movie, error) {
	var movie domains.Movie
	if err := json.Unmarshal([]byte(result), &movie); err != nil {
		return nil, err
	}
	return &movie, nil
}

func generateCacheValueFromMovie(movie *domains.Movie) (string, error) {
	movieJSON, err := json.Marshal(movie)
	if err != nil {
		return "", err
	}
	return string(movieJSON), nil
}

func generatePaginatedMoviesFromCacheResult(result string) (*domains.PaginatedMovies, error) {
	var movies domains.PaginatedMovies
	if err := json.Unmarshal([]byte(result), &movies); err != nil {
		return nil, err
	}
	return &movies, nil
}

func generateCacheValueFromPaginatedMovies(movies *domains.PaginatedMovies) (string, error) {
	moviesJSON, err := json.Marshal(movies)
	if err != nil {
		return "", err
	}
	return string(moviesJSON), nil
}
