package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
)

type UserService interface {
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	UpdateUserByID(ctx context.Context, userID int64, username string, email string, country string, profilePicture string) error
	UpdateUserPasswordByID(ctx context.Context, userID int64, currentSessionID string, currentPassword string, newPassword string) error
	DeleteUserByID(ctx context.Context, userID int64) error
}

type userService struct {
	userRepository    repositories.UserRepository
	sessionRepository repositories.SessionRepository
}

var ErrInvalidNewPassword = errors.New("invalid new password format")

func NewUserService(userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) UserService {
	return &userService{
		userRepository:    userRepo,
		sessionRepository: sessionRepo,
	}
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*domains.User, error) {
	return s.userRepository.GetUserByID(ctx, userID)
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*domains.User, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	return s.userRepository.GetUserByUsername(ctx, username)
}

func (s *userService) UpdateUserByID(ctx context.Context, userID int64, username string, email string, country string, profilePicture string) error {
	user, err := s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if username == "" {
		username = user.Username
	}
	if email == "" {
		email = user.Email
	}
	if country == "" {
		country = user.Country
	}
	if profilePicture == "" {
		profilePicture = user.ProfilePictureURL
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	country = strings.TrimSpace(country)
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := validateEmail(email); err != nil {
		return err
	}
	if err := validateCountry(country); err != nil {
		return err
	}
	newUser := domains.NewUser(username, email, user.PasswordHash, profilePicture, country)
	newUser.ID = userID
	return s.userRepository.UpdateUser(ctx, newUser)
}

func (s *userService) UpdateUserPasswordByID(ctx context.Context, userID int64, currentSessionID string, currentPassword string, newPassword string) error {
	if err := validatePassword(currentPassword); err != nil {
		return ErrInvalidPassword
	}
	passwordHash, err := s.userRepository.GetUserHashedPassword(ctx, userID)
	if err != nil {
		return err
	}
	if err := validatePassword(newPassword); err != nil {
		return ErrInvalidNewPassword
	}
	if err := checkPassword(passwordHash, currentPassword); err != nil {
		return ErrInvalidPassword
	}
	newPasswordHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.userRepository.UpdatePasswordByID(ctx, userID, newPasswordHash); err != nil {
		return err
	}
	return s.sessionRepository.DeleteOtherSessions(ctx, userID, currentSessionID)
}

func (s *userService) DeleteUserByID(ctx context.Context, userID int64) error {
	return s.userRepository.DeleteUserByID(ctx, userID)
}
