package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"letterbox/internal/domains"
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	"golang.org/x/crypto/bcrypt"
)

func TestRegisterUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		body       string
		repoErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"username":"ada","email":"ada@example.com","password":"secret123","country":"AR"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid payload",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid username",
			body:       `{"username":"a","email":"ada@example.com","password":"secret123"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "duplicate username",
			body:       `{"username":"ada","email":"ada@example.com","password":"secret123"}`,
			repoErr:    repositories.ErrUsernameAlreadyExists,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "repository error",
			body:       `{"username":"ada","email":"ada@example.com","password":"secret123"}`,
			repoErr:    genericErr,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created *domains.User
			userRepo := &testutil.FakeUserRepository{
				CreateUserFunc: func(_ context.Context, user *domains.User) (int64, error) {
					created = user
					return 99, tt.repoErr
				},
			}
			handler := NewAuthHandler(userRepo, &testutil.FakeSessionRepository{})

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			handler.RegisterUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				return
			}
			if created == nil || created.PasswordHash == "secret123" {
				t.Fatal("expected the password to be hashed before storage")
			}
			var got domains.User
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.ID != 99 || got.Username != "ada" || got.Country != "AR" {
				t.Fatalf("unexpected response: %+v", got)
			}
			if strings.Contains(rec.Body.String(), "password") {
				t.Fatalf("response leaked a password field: %s", rec.Body.String())
			}
		})
	}
}

func TestLoginUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com", PasswordHash: string(passwordHash)}

	tests := []struct {
		name        string
		body        string
		repoErr     error
		sessionErr  error
		wantStatus  int
		wantCookie  bool
		wantCreated bool
	}{
		{
			name:        "success",
			body:        `{"emailOrUsername":"ada","password":"secret123"}`,
			wantStatus:  http.StatusOK,
			wantCookie:  true,
			wantCreated: true,
		},
		{
			name:       "invalid payload",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong password",
			body:       `{"emailOrUsername":"ada","password":"wrong-password"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown user",
			body:       `{"emailOrUsername":"ghost","password":"secret123"}`,
			repoErr:    repositories.ErrUserNotFound,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "repository error",
			body:       `{"emailOrUsername":"ada","password":"secret123"}`,
			repoErr:    genericErr,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:        "session creation fails",
			body:        `{"emailOrUsername":"ada","password":"secret123"}`,
			sessionErr:  genericErr,
			wantStatus:  http.StatusInternalServerError,
			wantCookie:  false,
			wantCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionCreated := false
			userRepo := &testutil.FakeUserRepository{
				GetUserByEmailOrUsernameFunc: func(context.Context, string) (*domains.User, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return user, nil
				},
			}
			sessionRepo := &testutil.FakeSessionRepository{
				CreateSessionFunc: func(context.Context, *domains.Session) error {
					sessionCreated = true
					return tt.sessionErr
				},
			}
			handler := NewAuthHandler(userRepo, sessionRepo)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			handler.LoginUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if sessionCreated != tt.wantCreated {
				t.Fatalf("session created = %v; want %v", sessionCreated, tt.wantCreated)
			}
			cookies := rec.Result().Cookies()
			var sessionCookie *http.Cookie
			for _, c := range cookies {
				if c.Name == middleware.SessionCookieName {
					sessionCookie = c
				}
			}
			if tt.wantCookie {
				if sessionCookie == nil || sessionCookie.Value == "" {
					t.Fatal("expected a session cookie")
				}
				if !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
					t.Fatalf("unexpected cookie attributes: %+v", sessionCookie)
				}
				return
			}
			if sessionCookie != nil && sessionCookie.Value != "" {
				t.Fatalf("unexpected session cookie: %+v", sessionCookie)
			}
		})
	}
}

func TestLogoutUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		withCookie bool
		repoErr    error
		wantStatus int
	}{
		{name: "success", withCookie: true, wantStatus: http.StatusOK},
		{name: "missing cookie", wantStatus: http.StatusUnauthorized},
		{name: "repository error", withCookie: true, repoErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionRepo := &testutil.FakeSessionRepository{
				DeleteSessionByIDFunc: func(_ context.Context, sessionID string) error {
					if sessionID != "session-1" {
						t.Fatalf("DeleteSessionByID(%q); want session-1", sessionID)
					}
					return tt.repoErr
				},
			}
			handler := NewAuthHandler(&testutil.FakeUserRepository{}, sessionRepo)

			req := httptest.NewRequest(http.MethodDelete, "/", nil)
			if tt.withCookie {
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "session-1"})
			}
			rec := httptest.NewRecorder()
			handler.LogoutUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("got %d cookies; want 1", len(cookies))
			}
			if cookies[0].Name != middleware.SessionCookieName || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
				t.Fatalf("session cookie was not cleared: %+v", cookies[0])
			}
		})
	}
}
