package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"letterbox/internal/middleware"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	authBodyLimit     = 1 << 10  // 1 KiB
	passwordBodyLimit = 1 << 10  // 1 KiB
	userBodyLimit     = 4 << 10  // 4 KiB
	reviewBodyLimit   = 64 << 10 // 64 KiB
	defaultListLimit  = 20
	maxListLimit      = 100
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return false
		}
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return false
	}
	return true
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

func getPathInt(r *http.Request, name string, bits int) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, bits)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}

func getInt32PathValue(r *http.Request, name string) (int32, error) {
	id, err := getPathInt(r, name, 32)
	return int32(id), err
}

func getInt64PathValue(r *http.Request, name string) (int64, error) {
	return getPathInt(r, name, 64)
}
