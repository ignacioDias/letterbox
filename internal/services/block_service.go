package services

import (
	"context"
	"errors"
	"letterbox/internal/repositories"
	"strings"
)

type BlockService interface {
	BlockUserByUsername(ctx context.Context, userIDBlocker int64, usernameBlocked string) error
	UnblockUserByUsername(ctx context.Context, userIDBlocker int64, usernameBlocked string) error
}

type blockService struct {
	blockRepository repositories.BlockRepository
	userRepository  repositories.UserRepository
}

var ErrCannotBlockSelf = errors.New("cannot block yourself")

func NewBlockService(blockRepo repositories.BlockRepository, userRepo repositories.UserRepository) BlockService {
	return &blockService{
		blockRepository: blockRepo,
		userRepository:  userRepo,
	}
}

func (s *blockService) BlockUserByUsername(ctx context.Context, userIDBlocker int64, usernameBlocked string) error {
	usernameBlocked = strings.TrimSpace(usernameBlocked)
	if err := validateUsername(usernameBlocked); err != nil {
		return ErrInvalidUsername
	}
	blockedID, err := s.userRepository.GetUserIDByUsername(ctx, usernameBlocked)
	if err != nil {
		return err
	}
	if blockedID == userIDBlocker {
		return ErrCannotBlockSelf
	}
	return s.blockRepository.BlockUser(ctx, userIDBlocker, blockedID)
}

func (s *blockService) UnblockUserByUsername(ctx context.Context, userIDBlocker int64, usernameBlocked string) error {
	usernameBlocked = strings.TrimSpace(usernameBlocked)
	if err := validateUsername(usernameBlocked); err != nil {
		return ErrInvalidUsername
	}
	blockedID, err := s.userRepository.GetUserIDByUsername(ctx, usernameBlocked)
	if err != nil {
		return err
	}
	return s.blockRepository.UnblockUser(ctx, userIDBlocker, blockedID)
}
