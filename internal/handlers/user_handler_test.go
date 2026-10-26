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
	"letterbox/internal/repositories"
	"letterbox/internal/testutil"

	"golang.org/x/crypto/bcrypt"
)

func requestWithUsername(handler http.HandlerFunc, username string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("username", username)
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func requestWithUserID(handler http.HandlerFunc, userID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("user_id", userID)
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestGetCurrentUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com"}

	tests := []struct {
		name       string
		userErr    error
		wantStatus int
	}{
		{name: "success", wantStatus: http.StatusOK},
		{name: "not found", userErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", userErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserByIDFunc: func(_ context.Context, userID int64) (*domains.User, error) {
					if userID != 7 {
						t.Fatalf("GetUserByID(%d); want 7", userID)
					}
					return user, tt.userErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			rec := authenticatedRequest(t, handler.GetCurrentUser, 7, "")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.User
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.ID != 7 || got.Username != "ada" {
				t.Fatalf("unexpected response: %+v", got)
			}
		})
	}
}

func TestGetCurrentUserHandlerWithoutAuth(t *testing.T) {
	handler := NewUserHandler(&testutil.FakeUserRepository{}, &testutil.FakeSessionRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.GetCurrentUser(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func TestGetUserByUsernameHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com"}

	tests := []struct {
		name       string
		username   string
		userErr    error
		wantStatus int
	}{
		{name: "success", username: "ada", wantStatus: http.StatusOK},
		{name: "invalid username", username: "a", wantStatus: http.StatusBadRequest},
		{name: "not found", username: "ghost", userErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", username: "ada", userErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserByUsernameFunc: func(context.Context, string) (*domains.User, error) {
					return user, tt.userErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			rec := requestWithUsername(handler.GetUserByUsername, tt.username)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.PublicUser
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.ID != 7 || got.Username != "ada" {
				t.Fatalf("unexpected response: %+v", got)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if _, ok := body["email"]; ok {
				t.Fatalf("response leaked email: %s", rec.Body.String())
			}
		})
	}
}

func TestSearchUsersHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	users := []domains.PublicUser{{ID: 7, Username: "ada", ProfilePictureURL: "pic.png", Country: "AR"}}

	tests := []struct {
		name       string
		query      string
		result     []domains.PublicUser
		serviceErr error
		wantStatus int
		wantCalled bool
		wantSearch string
		wantLimit  int
		wantOffset int
	}{
		{name: "success", query: "?search=ada&limit=5&offset=10", result: users, wantStatus: http.StatusOK, wantCalled: true, wantSearch: "ada", wantLimit: 5, wantOffset: 10},
		{name: "empty result", query: "?search=ghost", result: []domains.PublicUser{}, wantStatus: http.StatusOK, wantCalled: true, wantSearch: "ghost", wantLimit: defaultListLimit},
		{name: "missing search", query: "", wantStatus: http.StatusBadRequest, wantCalled: false},
		{name: "invalid limit", query: "?search=ada&limit=abc", wantStatus: http.StatusBadRequest, wantCalled: false},
		{name: "invalid offset", query: "?search=ada&offset=-1", wantStatus: http.StatusBadRequest, wantCalled: false},
		{name: "service error", query: "?search=ada", serviceErr: genericErr, wantStatus: http.StatusInternalServerError, wantCalled: true, wantSearch: "ada", wantLimit: defaultListLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var gotSearch string
			var gotLimit, gotOffset int
			userRepo := &testutil.FakeUserRepository{
				SearchUsersFunc: func(_ context.Context, search string, limit int, offset int) ([]domains.PublicUser, error) {
					called = true
					gotSearch = search
					gotLimit = limit
					gotOffset = offset
					return tt.result, tt.serviceErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.SearchUsers(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("repository called = %v; want %v", called, tt.wantCalled)
			}
			if tt.wantCalled && (gotSearch != tt.wantSearch || gotLimit != tt.wantLimit || gotOffset != tt.wantOffset) {
				t.Fatalf("repository called with search %q, limit %d, offset %d; want %q, %d, %d",
					gotSearch, gotLimit, gotOffset, tt.wantSearch, tt.wantLimit, tt.wantOffset)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got []domains.PublicUser
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got) != len(tt.result) {
				t.Fatalf("response has %d users; want %d", len(got), len(tt.result))
			}
			for i := range got {
				if got[i] != tt.result[i] {
					t.Fatalf("response user %d = %+v; want %+v", i, got[i], tt.result[i])
				}
			}
			if len(tt.result) == 0 && strings.TrimSpace(rec.Body.String()) != "[]" {
				t.Fatalf("empty result body = %q; want []", rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), `"email"`) {
				t.Fatalf("response leaked email: %s", rec.Body.String())
			}
		})
	}
}

func TestGetUserByIDHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	user := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com", Country: "AR"}

	tests := []struct {
		name       string
		userID     string
		userErr    error
		wantStatus int
	}{
		{name: "success", userID: "7", wantStatus: http.StatusOK},
		{name: "invalid ID", userID: "abc", wantStatus: http.StatusBadRequest},
		{name: "zero ID", userID: "0", wantStatus: http.StatusBadRequest},
		{name: "negative ID", userID: "-1", wantStatus: http.StatusBadRequest},
		{name: "not found", userID: "7", userErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", userID: "7", userErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserByIDFunc: func(_ context.Context, userID int64) (*domains.User, error) {
					if userID != 7 {
						t.Fatalf("GetUserByID(%d); want 7", userID)
					}
					return user, tt.userErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			rec := requestWithUserID(handler.GetUserByID, tt.userID)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got domains.PublicUser
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.ID != 7 || got.Username != "ada" || got.Country != "AR" {
				t.Fatalf("unexpected response: %+v", got)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if _, ok := body["email"]; ok {
				t.Fatalf("response leaked email: %s", rec.Body.String())
			}
		})
	}
}

func TestDeleteCurrentUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		deleteErr  error
		wantStatus int
	}{
		{name: "success", wantStatus: http.StatusNoContent},
		{name: "not found", deleteErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", deleteErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				DeleteUserByIDFunc: func(_ context.Context, userID int64) error {
					if userID != 7 {
						t.Fatalf("DeleteUserByID(%d); want 7", userID)
					}
					return tt.deleteErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			rec := authenticatedRequest(t, handler.DeleteCurrentUser, 7, "")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusNoContent {
				return
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
				t.Fatalf("session cookie was not cleared: %+v", cookies)
			}
		})
	}
}

func TestUpdateCurrentUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	current := &domains.User{ID: 7, Username: "ada", Email: "ada@example.com", Country: "AR", PasswordHash: "hash"}

	tests := []struct {
		name       string
		body       string
		getErr     error
		updateErr  error
		wantStatus int
	}{
		{name: "success", body: `{"country":"UY"}`, wantStatus: http.StatusNoContent},
		{name: "invalid payload", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "body too large", body: `{"country":"` + strings.Repeat("a", userBodyLimit) + `"}`, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "invalid username", body: `{"username":"a"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid country", body: `{"country":"ARG"}`, wantStatus: http.StatusBadRequest},
		{name: "username conflict", body: `{"username":"grace"}`, updateErr: repositories.ErrUsernameAlreadyExists, wantStatus: http.StatusConflict},
		{name: "not found", body: `{}`, getErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", body: `{}`, updateErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updated *domains.User
			userRepo := &testutil.FakeUserRepository{
				GetUserByIDFunc: func(context.Context, int64) (*domains.User, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return current, nil
				},
				UpdateUserFunc: func(_ context.Context, user *domains.User) error {
					updated = user
					return tt.updateErr
				},
			}
			handler := NewUserHandler(userRepo, &testutil.FakeSessionRepository{})
			rec := authenticatedRequestWithBody(t, handler.UpdateCurrentUser, tt.body, 7)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusNoContent && (updated == nil || updated.Country != "UY") {
				t.Fatalf("unexpected updated user: %+v", updated)
			}
		})
	}
}

func TestUpdateCurrentUserPasswordHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")
	currentHash, err := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	tests := []struct {
		name       string
		body       string
		hash       string
		hashErr    error
		wantStatus int
	}{
		{name: "success", body: `{"currentPassword":"oldpassword","newPassword":"newpassword"}`, hash: string(currentHash), wantStatus: http.StatusNoContent},
		{name: "invalid payload", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "body too large", body: `{"currentPassword":"` + strings.Repeat("a", passwordBodyLimit) + `"}`, hash: string(currentHash), wantStatus: http.StatusRequestEntityTooLarge},
		{name: "invalid new password", body: `{"currentPassword":"oldpassword","newPassword":"short"}`, hash: string(currentHash), wantStatus: http.StatusBadRequest},
		{name: "wrong current password", body: `{"currentPassword":"wrongpassword","newPassword":"newpassword"}`, hash: string(currentHash), wantStatus: http.StatusUnauthorized},
		{name: "user not found", body: `{"currentPassword":"oldpassword","newPassword":"newpassword"}`, hashErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "repository error", body: `{"currentPassword":"oldpassword","newPassword":"newpassword"}`, hashErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testutil.FakeUserRepository{
				GetUserHashedPasswordFunc: func(context.Context, int64) (string, error) {
					return tt.hash, tt.hashErr
				},
				UpdatePasswordByIDFunc: func(context.Context, int64, string) error {
					return nil
				},
			}
			sessionRepo := &testutil.FakeSessionRepository{
				DeleteOtherSessionsFunc: func(context.Context, int64, string) error {
					return nil
				},
			}
			handler := NewUserHandler(userRepo, sessionRepo)
			rec := authenticatedRequestWithBody(t, handler.UpdateCurrentUserPassword, tt.body, 7)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestUpdateCurrentUserPasswordHandlerWithoutAuth(t *testing.T) {
	handler := NewUserHandler(&testutil.FakeUserRepository{}, &testutil.FakeSessionRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{}`))

	handler.UpdateCurrentUserPassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}
