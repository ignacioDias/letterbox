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
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
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

func (s *userService) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	return s.userRepository.GetFollowersByUsername(ctx, username, limit, offset)
}

func (s *userService) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	return s.userRepository.GetFollowingByUsername(ctx, username, limit, offset)
}
