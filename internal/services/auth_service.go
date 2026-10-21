package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"net/mail"
	"regexp"
	"strings"
	"time"

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
	dummyPasswordHash = "$2a$12$t2pJYP1xjBB5tIUDSNK0YOLqMhEYw8jg7Zx9VUWJVgzjno.f8VEHu"
)

type AuthService interface {
	RegisterUser(ctx context.Context, username, email, password, country string) (*domains.User, error)
	LoginUser(ctx context.Context, emailOrUsername, password string) (*domains.User, error)
	CreateSession(ctx context.Context, userID int64) (*domains.Session, error)
}

type authService struct {
	userRepository    repositories.UserRepository
	sessionRepository repositories.SessionRepository
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func NewAuthService(userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) AuthService {
	return &authService{
		userRepository:    userRepo,
		sessionRepository: sessionRepo,
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
	userID, err := s.userRepository.CreateUser(ctx, user)
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

func (s *authService) LoginUser(ctx context.Context, emailOrUsername, password string) (*domains.User, error) {
	emailOrUsername = strings.TrimSpace(emailOrUsername)
	user, err := s.userRepository.GetUserByEmailOrUsername(ctx, emailOrUsername)
	if errors.Is(err, repositories.ErrUserNotFound) {
		// Dummy bcrypt comparison prevents timing differences that reveal whether the account exists.
		_ = checkPassword(dummyPasswordHash, password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := checkPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

func checkPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (s *authService) CreateSession(ctx context.Context, userID int64) (*domains.Session, error) {
	createdAt := time.Now()
	session := &domains.Session{
		ID:        generateSessionID(),
		UserID:    userID,
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(time.Hour),
	}
	if err := s.sessionRepository.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func generateSessionID() string {
	return rand.Text()
}
