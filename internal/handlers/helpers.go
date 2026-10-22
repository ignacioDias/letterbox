package handlers

import (
	"encoding/json"
	"errors"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"log"
	"net/http"
	"time"
)

type errorMapping struct {
	err     error
	status  int
	message string
}

var (
	invalidUsernameError = errorMapping{err: services.ErrInvalidUsername, status: http.StatusBadRequest, message: "Invalid username"}
	userNotFoundError    = errorMapping{err: repositories.ErrUserNotFound, status: http.StatusNotFound, message: "User not found"}
)

func writeError(w http.ResponseWriter, err error, logContext, fallback string, mappings ...errorMapping) {
	for _, mapping := range mappings {
		if errors.Is(err, mapping.err) {
			message := mapping.message
			if message == "" {
				message = err.Error()
			}
			http.Error(w, message, mapping.status)
			return
		}
	}
	log.Printf("%s: %v", logContext, err)
	http.Error(w, fallback, http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return false
	}
	return true
}

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
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
