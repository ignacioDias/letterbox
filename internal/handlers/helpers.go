package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/middleware"
	"log"
	"net/http"
	"strconv"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}

func requireSessionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}
	return cookie.Value, true
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func parsePagination(r *http.Request) (int, int, error) {
	limit := defaultListLimit
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxListLimit {
			return 0, 0, errors.New("invalid limit")
		}
		limit = value
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, errors.New("invalid offset")
		}
		offset = value
	}
	return limit, offset, nil
}

func getMovieIDFromRequest(r *http.Request) (int32, error) {
	parsedMovieID, err := strconv.ParseInt(r.PathValue("movie_id"), 10, 32)
	if err != nil {
		return 0, err
	}
	movieID := int32(parsedMovieID)
	if movieID <= 0 {
		return 0, errors.New("invalid movie ID")
	}
	return movieID, nil
}
