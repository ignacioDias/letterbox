package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"strings"
)

type FollowService interface {
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	FollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error
	UnfollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error
	GetFollowersCountByUsername(ctx context.Context, username string) (int, error)
	GetFollowingCountByUsername(ctx context.Context, username string) (int, error)
}

type followService struct {
	userRepository   repositories.UserRepository
	followRepository repositories.FollowRepository
	blockRepository  repositories.BlockRepository
}

var ErrCannotFollowSelf = errors.New("cannot follow yourself")
var ErrCannotUnfollowSelf = errors.New("cannot unfollow yourself")

func NewFollowService(followRepo repositories.FollowRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository) FollowService {
	return &followService{
		followRepository: followRepo,
		userRepository:   userRepo,
		blockRepository:  blockRepo,
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

func (s *followService) FollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error {
	usernameFollowing = strings.TrimSpace(usernameFollowing)
	if err := validateUsername(usernameFollowing); err != nil {
		return err
	}
	followingID, err := s.userRepository.GetUserIDByUsername(ctx, usernameFollowing)
	if err != nil {
		return err
	}
	if followingID == userIDFollower {
		return ErrCannotFollowSelf
	}
	blocked, err := s.blockRepository.IsBlockedBetween(ctx, userIDFollower, followingID)
	if err != nil {
		return err
	}
	if blocked {
		return repositories.ErrUserBlocked
	}
	return s.followRepository.FollowUser(ctx, userIDFollower, followingID)
}

func (s *followService) UnfollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error {
	usernameFollowing = strings.TrimSpace(usernameFollowing)
	if err := validateUsername(usernameFollowing); err != nil {
		return err
	}
	followingID, err := s.userRepository.GetUserIDByUsername(ctx, usernameFollowing)
	if err != nil {
		return err
	}
	if followingID == userIDFollower {
		return ErrCannotUnfollowSelf
	}
	return s.followRepository.UnfollowUser(ctx, userIDFollower, followingID)
}

func (s *followService) GetFollowersCountByUsername(ctx context.Context, username string) (int, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return 0, ErrInvalidUsername
	}
	userID, err := s.userRepository.GetUserIDByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	return s.followRepository.GetFollowersCountByUserID(ctx, userID)
}

func (s *followService) GetFollowingCountByUsername(ctx context.Context, username string) (int, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return 0, ErrInvalidUsername
	}
	userID, err := s.userRepository.GetUserIDByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	return s.followRepository.GetFollowingCountByUserID(ctx, userID)
}
