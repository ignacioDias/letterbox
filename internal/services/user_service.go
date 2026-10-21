package services

import (
	"context"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
)

type UserService interface {
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
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
