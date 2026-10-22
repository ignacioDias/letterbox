package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"letterbox/internal/repositories"
	"letterbox/internal/testutil"
)

func TestBlockUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		username   string
		userID     int64
		targetID   int64
		targetErr  error
		blockErr   error
		wantStatus int
	}{
		{name: "success", username: "grace", userID: 7, targetID: 42, wantStatus: http.StatusNoContent},
		{name: "cannot block self", username: "ada", userID: 7, targetID: 7, wantStatus: http.StatusBadRequest},
		{name: "invalid username", username: "a", userID: 7, wantStatus: http.StatusBadRequest},
		{name: "target not found", username: "ghost", userID: 7, targetErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "already blocked", username: "grace", userID: 7, targetID: 42, blockErr: repositories.ErrAlreadyBlocked, wantStatus: http.StatusConflict},
		{name: "repository error", username: "grace", userID: 7, targetID: 42, blockErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockRepo := &testutil.FakeBlockRepository{
				BlockUserFunc: func(context.Context, int64, int64) error {
					return tt.blockErr
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			handler := NewBlockHandler(blockRepo, userRepo)
			rec := authenticatedRequest(t, handler.BlockUserByUsername, tt.userID, tt.username)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestBlockUserHandlerWithoutAuth(t *testing.T) {
	handler := NewBlockHandler(&testutil.FakeBlockRepository{}, &testutil.FakeUserRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.SetPathValue("username", "grace")

	handler.BlockUserByUsername(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", rec.Code)
	}
}

func TestUnblockUserHandler(t *testing.T) {
	genericErr := errors.New("database unavailable")

	tests := []struct {
		name       string
		username   string
		userID     int64
		targetID   int64
		targetErr  error
		unblockErr error
		wantStatus int
	}{
		{name: "success", username: "grace", userID: 7, targetID: 42, wantStatus: http.StatusNoContent},
		{name: "invalid username", username: "a", userID: 7, wantStatus: http.StatusBadRequest},
		{name: "target not found", username: "ghost", userID: 7, targetErr: repositories.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "not blocked", username: "grace", userID: 7, targetID: 42, unblockErr: repositories.ErrNotBlocked, wantStatus: http.StatusConflict},
		{name: "repository error", username: "grace", userID: 7, targetID: 42, unblockErr: genericErr, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockRepo := &testutil.FakeBlockRepository{
				UnblockUserFunc: func(context.Context, int64, int64) error {
					return tt.unblockErr
				},
			}
			userRepo := &testutil.FakeUserRepository{
				GetUserIDByUsernameFunc: func(context.Context, string) (int64, error) {
					return tt.targetID, tt.targetErr
				},
			}

			handler := NewBlockHandler(blockRepo, userRepo)
			rec := authenticatedRequest(t, handler.UnblockUserByUsername, tt.userID, tt.username)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d (body %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
