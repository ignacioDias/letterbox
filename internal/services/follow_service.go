package services

import (
	"context"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
)

type FollowService interface {
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
}

type followService struct {
	followRepository repositories.FollowRepository
}

func NewFollowService(followRepo repositories.FollowRepository) FollowService {
	return &followService{
		followRepository: followRepo,
	}
}

func (s *followService) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	return s.followRepository.GetFollowersByUsername(ctx, username, limit, offset)
}

func (s *followService) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	return s.followRepository.GetFollowingByUsername(ctx, username, limit, offset)
}
