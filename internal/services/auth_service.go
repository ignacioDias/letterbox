package services

import (
	"context"
	"errors"
	"fmt"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("invalid password")

	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
)

type AuthService interface {
	RegisterUser(ctx context.Context, username, email, password, country string) (*domains.User, error)
}

type authService struct {
	authRepository repositories.AuthRepository
}

func NewAuthService(authRepo repositories.AuthRepository) AuthService {
	return &authService{
		authRepository: authRepo,
	}
}

func (s *authService) RegisterUser(ctx context.Context, username, email, password, country string) (*domains.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	user := domains.NewUser(username, email, hashedPassword, "", country)
	userID, err := s.authRepository.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = userID
	return user, nil
}

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("%w: must be 3-20 characters and contain only letters, numbers, or underscores", ErrInvalidUsername)
	}
	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fmt.Errorf("%w: must be a valid email address", ErrInvalidEmail)
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrInvalidPassword, minPasswordLength)
	}
	if len(password) > maxPasswordLength {
		return fmt.Errorf("%w: must be at most %d characters", ErrInvalidPassword, maxPasswordLength)
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hashedPass), err
}
