package middleware

import (
	"context"
	"errors"
	"letterbox/internal/repositories"
	"log"
	"net/http"
)

const SessionCookieName = "session_id"

type contextKey string

const userIDKey contextKey = "userID"

type AuthenticationMiddleware interface {
	Authenticate(next http.HandlerFunc) http.HandlerFunc
}

type authenticationMiddleware struct {
	sessionRepo repositories.SessionRepository
}

func NewAuthenticationMiddleware(sessionRepo repositories.SessionRepository) AuthenticationMiddleware {
	return &authenticationMiddleware{
		sessionRepo: sessionRepo,
	}
}

func (m *authenticationMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized: missing session cookie", http.StatusUnauthorized)
			return
		}
		session, err := m.sessionRepo.GetSessionByID(r.Context(), cookie.Value)
		if errors.Is(err, repositories.ErrSessionNotFound) {
			http.Error(w, "Unauthorized: invalid session", http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("authentication: failed to get session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if session == nil {
			log.Print("authentication: repository returned a nil session without an error")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
		next(w, r.WithContext(ctx))
	}
}

func GetUserID(r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value(userIDKey).(int64)
	return userID, ok
}
