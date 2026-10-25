package services

import (
	"context"
	"errors"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
)

type ReactionService interface {
	CreateReactionForReviewByID(ctx context.Context, reviewID int64, userID int64, score int16) (*domains.Reaction, error)
	RemoveReactionFromReviewByID(ctx context.Context, reviewID int64, userID int64) error
	CreateReactionForCommentByID(ctx context.Context, commentID int64, userID int64, score int16) (*domains.Reaction, error)
	RemoveReactionFromCommentByID(ctx context.Context, commentID int64, userID int64) error
}

type reactionService struct {
	reactionRepo repositories.ReactionRepository
}

var ErrInvalidReactionValue = errors.New("invalid reaction value: must be 1 or -1")

func NewReactionService(reactionRepo repositories.ReactionRepository) ReactionService {
	return &reactionService{
		reactionRepo: reactionRepo,
	}
}

func (s *reactionService) CreateReactionForReviewByID(ctx context.Context, reviewID int64, userID int64, score int16) (*domains.Reaction, error) {
	if score != 1 && score != -1 {
		return nil, ErrInvalidReactionValue
	}
	reaction := domains.NewReaction(reviewID, userID, score)
	return s.reactionRepo.CreateReactionForReview(ctx, reaction)
}

func (s *reactionService) RemoveReactionFromReviewByID(ctx context.Context, reviewID int64, userID int64) error {
	return s.reactionRepo.RemoveReactionFromReview(ctx, reviewID, userID)
}

func (s *reactionService) CreateReactionForCommentByID(ctx context.Context, commentID int64, userID int64, score int16) (*domains.Reaction, error) {
	if score != 1 && score != -1 {
		return nil, ErrInvalidReactionValue
	}
	reaction := domains.NewReaction(commentID, userID, score)
	return s.reactionRepo.CreateReactionForComment(ctx, reaction)
}

func (s *reactionService) RemoveReactionFromCommentByID(ctx context.Context, commentID int64, userID int64) error {
	return s.reactionRepo.RemoveReactionFromComment(ctx, commentID, userID)
}
