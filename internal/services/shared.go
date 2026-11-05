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

func getKeyForUserReviewsGeneration(userID int64) string {
	return "review:" + cacheKeyVersion + ":user:" + strconv.FormatInt(userID, 10) + ":generation"
}

func getKeyForUserReviews(userID int64, limit, offset int, generation string) string {
	return "review:" + cacheKeyVersion + ":user:" + strconv.FormatInt(userID, 10) + ":generation:" + generation + ":limit:" + strconv.Itoa(limit) + ":offset:" + strconv.Itoa(offset)
}

func getKeyForFollowingReviewsGeneration() string {
	return "review:" + cacheKeyVersion + ":following:generation"
}

func getKeyForFollowingReviews(userID int64, limit, offset int, generation string) string {
	return "review:" + cacheKeyVersion + ":following:user:" + strconv.FormatInt(userID, 10) + ":generation:" + generation + ":limit:" + strconv.Itoa(limit) + ":offset:" + strconv.Itoa(offset)
}

func getKeyForMovieReviewsGeneration(movieID int32) string {
	return "review:" + cacheKeyVersion + ":movie:" + strconv.FormatInt(int64(movieID), 10) + ":generation"
}

func getKeyForMovieReviews(movieID int32, limit, offset int, generation string) string {
	return "review:" + cacheKeyVersion + ":movie:" + strconv.FormatInt(int64(movieID), 10) + ":generation:" + generation + ":limit:" + strconv.Itoa(limit) + ":offset:" + strconv.Itoa(offset)
}

func getKeyForMovieReviewAverage(movieID int32) string {
	return "review:" + cacheKeyVersion + ":movie:" + strconv.FormatInt(int64(movieID), 10) + ":average"
}

func getKeyForMovieReviewCount(movieID int32) string {
	return "review:" + cacheKeyVersion + ":movie:" + strconv.FormatInt(int64(movieID), 10) + ":count"
}

func getKeyForReviewByID(reviewID int64) string {
	return "review:" + cacheKeyVersion + ":id:" + strconv.FormatInt(reviewID, 10)
}

func getKeyForUserByID(userID int64) string {
	return "user:" + cacheKeyVersion + ":id:" + strconv.FormatInt(userID, 10)
}

func getKeyForUserByUsername(username string) string {
	return "user:" + cacheKeyVersion + ":username:" + username
}

func getKeyForUserSearchGeneration() string {
	return "user:" + cacheKeyVersion + ":search:generation"
}

func getKeyForUserSearch(search string, limit, offset int, generation string) string {
	return "user:" + cacheKeyVersion + ":search:query:" + url.QueryEscape(strings.ToLower(search)) + ":generation:" + generation + ":limit:" + strconv.Itoa(limit) + ":offset:" + strconv.Itoa(offset)
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

func generateReviewsFromCacheResult(result string) ([]*domains.Review, error) {
	var reviews []*domains.Review
	if err := json.Unmarshal([]byte(result), &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

func generateCacheValueFromReviews(reviews []*domains.Review) (string, error) {
	reviewsJSON, err := json.Marshal(reviews)
	if err != nil {
		return "", err
	}
	return string(reviewsJSON), nil
}

func generateReviewFromCacheResult(result string) (*domains.Review, error) {
	var review domains.Review
	if err := json.Unmarshal([]byte(result), &review); err != nil {
		return nil, err
	}
	return &review, nil
}

func generateCacheValueFromReview(review *domains.Review) (string, error) {
	reviewJSON, err := json.Marshal(review)
	if err != nil {
		return "", err
	}
	return string(reviewJSON), nil
}

func generateUserFromCacheResult(result string) (*domains.User, error) {
	var user domains.User
	if err := json.Unmarshal([]byte(result), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func generateCacheValueFromUser(user *domains.User) (string, error) {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	return string(userJSON), nil
}

func generatePublicUserListFromCacheResult(result string) ([]domains.PublicUser, error) {
	var users []domains.PublicUser
	if err := json.Unmarshal([]byte(result), &users); err != nil {
		return nil, err
	}
	return users, nil
}

func generateCacheValueFromPublicUserList(users []domains.PublicUser) (string, error) {
	usersJSON, err := json.Marshal(users)
	if err != nil {
		return "", err
	}
	return string(usersJSON), nil
}
