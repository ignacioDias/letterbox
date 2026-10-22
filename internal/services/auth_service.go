package services

import (
	"context"
	"crypto/rand"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
	"time"
)

const dummyPasswordHash = "$2a$12$t2pJYP1xjBB5tIUDSNK0YOLqMhEYw8jg7Zx9VUWJVgzjno.f8VEHu"

type AuthService interface {
	RegisterUser(ctx context.Context, username, email, password, country string) (*domains.User, error)
	LoginUser(ctx context.Context, emailOrUsername, password string) (*domains.User, error)
	CreateSession(ctx context.Context, userID int64) (*domains.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
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

func (s *authService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.sessionRepository.DeleteSessionByID(ctx, sessionID)
}
