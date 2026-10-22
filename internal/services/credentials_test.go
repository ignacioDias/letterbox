package services

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{name: "valid simple", username: "ada"},
		{name: "valid with digits", username: "ada123"},
		{name: "valid with underscore", username: "ada_lovelace"},
		{name: "valid max length", username: strings.Repeat("a", 20)},
		{name: "empty", username: "", wantErr: true},
		{name: "too short", username: "ab", wantErr: true},
		{name: "too long", username: strings.Repeat("a", 21), wantErr: true},
		{name: "hyphen", username: "ada-lovelace", wantErr: true},
		{name: "space", username: "ada lovelace", wantErr: true},
		{name: "non ascii", username: "ádá", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)
			if tt.wantErr && !errors.Is(err, ErrInvalidUsername) {
				t.Fatalf("validateUsername(%q) = %v; want ErrInvalidUsername", tt.username, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateUsername(%q) = %v; want nil", tt.username, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "valid", email: "ada@example.com"},
		{name: "valid with plus", email: "ada+movies@example.com"},
		{name: "trimmed", email: "  ada@example.com  "},
		{name: "empty", email: "", wantErr: true},
		{name: "no at", email: "ada.example.com", wantErr: true},
		{name: "no domain", email: "ada@", wantErr: true},
		{name: "with display name", email: "Ada <ada@example.com>", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if tt.wantErr && !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("validateEmail(%q) = %v; want ErrInvalidEmail", tt.email, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateEmail(%q) = %v; want nil", tt.email, err)
			}
		})
	}
}

func TestValidateCountry(t *testing.T) {
	tests := []struct {
		name    string
		country string
		wantErr bool
	}{
		{name: "empty", country: ""},
		{name: "valid uppercase", country: "AR"},
		{name: "valid lowercase", country: "ar"},
		{name: "too short", country: "A", wantErr: true},
		{name: "too long", country: "ARG", wantErr: true},
		{name: "non letters", country: "A1", wantErr: true},
		{name: "space", country: "A R", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCountry(tt.country)
			if tt.wantErr && !errors.Is(err, ErrInvalidCountry) {
				t.Fatalf("validateCountry(%q) = %v; want ErrInvalidCountry", tt.country, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateCountry(%q) = %v; want nil", tt.country, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid minimum", password: strings.Repeat("a", 8)},
		{name: "valid maximum", password: strings.Repeat("a", 72)},
		{name: "too short", password: strings.Repeat("a", 7), wantErr: true},
		{name: "too long", password: strings.Repeat("a", 73), wantErr: true},
		{name: "empty", password: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantErr && !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("validatePassword(%q) = %v; want ErrInvalidPassword", tt.password, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validatePassword(%q) = %v; want nil", tt.password, err)
			}
		})
	}
}

func TestPasswordHashing(t *testing.T) {
	const password = "secret123"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if hash == password {
		t.Fatal("hashPassword returned the plaintext password")
	}
	if err := checkPassword(hash, password); err != nil {
		t.Fatalf("checkPassword with correct password: %v", err)
	}
	if err := checkPassword(hash, "wrong-password"); err == nil {
		t.Fatal("checkPassword with wrong password returned nil")
	}
}
