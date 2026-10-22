package services

import (
	"context"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
)

type UserService interface {
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	DeleteUserByID(ctx context.Context, userID int64) error
	UpdateUserByID(ctx context.Context, userID int64, username string, email string, country string, profilePicture string) error
}

type userService struct {
	userRepository repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepository: userRepo,
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

func (s *userService) DeleteUserByID(ctx context.Context, userID int64) error {
	return s.userRepository.DeleteUserByID(ctx, userID)
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
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := validateEmail(email); err != nil {
		return err
	}
	newUser := domains.NewUser(username, email, user.PasswordHash, profilePicture, country)
	newUser.ID = userID
	return s.userRepository.UpdateUser(ctx, newUser)
}
