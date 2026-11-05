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

func getKeyForFollowGeneration(kind, username string) string {
	return "follow:" + cacheKeyVersion + ":" + kind + ":generation:user:" + username
}

func getKeyForFollowList(kind, username string, limit, offset int, generation string) string {
	return "follow:" + cacheKeyVersion + ":" + kind + ":user:" + username + ":generation:" + generation + ":limit:" + strconv.Itoa(limit) + ":offset:" + strconv.Itoa(offset)
}

func getKeyForFollowCount(kind, username string) string {
	return "follow:" + cacheKeyVersion + ":" + kind + ":count:user:" + username
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

func generatePublicUsersFromCacheResult(result string) ([]*domains.PublicUser, error) {
	var users []*domains.PublicUser
	if err := json.Unmarshal([]byte(result), &users); err != nil {
		return nil, err
	}
	return users, nil
}

func generateCacheValueFromPublicUsers(users []*domains.PublicUser) (string, error) {
	usersJSON, err := json.Marshal(users)
	if err != nil {
		return "", err
	}
	return string(usersJSON), nil
}
