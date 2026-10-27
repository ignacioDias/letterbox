package services

import (
	"encoding/json"
	"letterbox/internal/domains"
	"strconv"
)

var cacheKeyVersion = "v1"

func getKeyForMovieByID(movieID int32) string {
	return "tmdb:" + cacheKeyVersion + ":movieID:" + strconv.Itoa(int(movieID))
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
