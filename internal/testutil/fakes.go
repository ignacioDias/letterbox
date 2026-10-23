package testutil

import (
	"context"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
)

type FakeUserRepository struct {
	repositories.UserRepository

	CreateUserFunc               func(ctx context.Context, user *domains.User) (int64, error)
	GetUserIDByUsernameFunc      func(ctx context.Context, username string) (int64, error)
	GetUserByEmailOrUsernameFunc func(ctx context.Context, emailOrUsername string) (*domains.User, error)
	GetUserByUsernameFunc        func(ctx context.Context, username string) (*domains.User, error)
	GetUserByIDFunc              func(ctx context.Context, userID int64) (*domains.User, error)
	GetUserHashedPasswordFunc    func(ctx context.Context, userID int64) (string, error)
	DeleteUserByIDFunc           func(ctx context.Context, userID int64) error
	UpdateUserFunc               func(ctx context.Context, user *domains.User) error
	UpdatePasswordByIDFunc       func(ctx context.Context, userID int64, newPasswordHash string) error
}

func (f *FakeUserRepository) CreateUser(ctx context.Context, user *domains.User) (int64, error) {
	return f.CreateUserFunc(ctx, user)
}

func (f *FakeUserRepository) GetUserIDByUsername(ctx context.Context, username string) (int64, error) {
	return f.GetUserIDByUsernameFunc(ctx, username)
}

func (f *FakeUserRepository) GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (*domains.User, error) {
	return f.GetUserByEmailOrUsernameFunc(ctx, emailOrUsername)
}

func (f *FakeUserRepository) GetUserByUsername(ctx context.Context, username string) (*domains.User, error) {
	return f.GetUserByUsernameFunc(ctx, username)
}

func (f *FakeUserRepository) GetUserByID(ctx context.Context, userID int64) (*domains.User, error) {
	return f.GetUserByIDFunc(ctx, userID)
}

func (f *FakeUserRepository) GetUserHashedPassword(ctx context.Context, userID int64) (string, error) {
	return f.GetUserHashedPasswordFunc(ctx, userID)
}

func (f *FakeUserRepository) DeleteUserByID(ctx context.Context, userID int64) error {
	return f.DeleteUserByIDFunc(ctx, userID)
}

func (f *FakeUserRepository) UpdateUser(ctx context.Context, user *domains.User) error {
	return f.UpdateUserFunc(ctx, user)
}

func (f *FakeUserRepository) UpdatePasswordByID(ctx context.Context, userID int64, newPasswordHash string) error {
	return f.UpdatePasswordByIDFunc(ctx, userID, newPasswordHash)
}

type FakeSessionRepository struct {
	repositories.SessionRepository

	CreateSessionFunc         func(ctx context.Context, session *domains.Session) error
	GetSessionByIDFunc        func(ctx context.Context, sessionID string) (*domains.Session, error)
	DeleteSessionByIDFunc     func(ctx context.Context, sessionID string) error
	DeleteOtherSessionsFunc   func(ctx context.Context, userID int64, keepSessionID string) error
	DeleteExpiredSessionsFunc func(ctx context.Context) error
}

func (f *FakeSessionRepository) CreateSession(ctx context.Context, session *domains.Session) error {
	return f.CreateSessionFunc(ctx, session)
}

func (f *FakeSessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*domains.Session, error) {
	return f.GetSessionByIDFunc(ctx, sessionID)
}

func (f *FakeSessionRepository) DeleteSessionByID(ctx context.Context, sessionID string) error {
	return f.DeleteSessionByIDFunc(ctx, sessionID)
}

func (f *FakeSessionRepository) DeleteOtherSessions(ctx context.Context, userID int64, keepSessionID string) error {
	return f.DeleteOtherSessionsFunc(ctx, userID, keepSessionID)
}

func (f *FakeSessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	return f.DeleteExpiredSessionsFunc(ctx)
}

type FakeFollowRepository struct {
	repositories.FollowRepository

	GetFollowersByUsernameFunc    func(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsernameFunc    func(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowersCountByUserIDFunc func(ctx context.Context, userID int64) (int, error)
	GetFollowingCountByUserIDFunc func(ctx context.Context, userID int64) (int, error)
	UnfollowUserFunc              func(ctx context.Context, userIDFollower, userIDFollowing int64) error
	FollowUserFunc                func(ctx context.Context, userIDFollower, userIDFollowing int64) error
}

func (f *FakeFollowRepository) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	return f.GetFollowersByUsernameFunc(ctx, username, limit, offset)
}

func (f *FakeFollowRepository) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	return f.GetFollowingByUsernameFunc(ctx, username, limit, offset)
}

func (f *FakeFollowRepository) GetFollowersCountByUserID(ctx context.Context, userID int64) (int, error) {
	return f.GetFollowersCountByUserIDFunc(ctx, userID)
}

func (f *FakeFollowRepository) GetFollowingCountByUserID(ctx context.Context, userID int64) (int, error) {
	return f.GetFollowingCountByUserIDFunc(ctx, userID)
}

func (f *FakeFollowRepository) UnfollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error {
	return f.UnfollowUserFunc(ctx, userIDFollower, userIDFollowing)
}

func (f *FakeFollowRepository) FollowUser(ctx context.Context, userIDFollower, userIDFollowing int64) error {
	return f.FollowUserFunc(ctx, userIDFollower, userIDFollowing)
}

type FakeReviewRepository struct {
	repositories.ReviewRepository

	GetReviewsFromUserByUserIDFunc func(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error)
	GetReviewsFromMovieByIDFunc    func(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error)
	CreateReviewForMovieByIDFunc   func(ctx context.Context, review *domains.Review) (*domains.Review, error)
}

func (f *FakeReviewRepository) GetReviewsFromUserByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domains.Review, error) {
	return f.GetReviewsFromUserByUserIDFunc(ctx, userID, limit, offset)
}

func (f *FakeReviewRepository) GetReviewsFromMovieByID(ctx context.Context, movieID int32, limit, offset int) ([]*domains.Review, error) {
	return f.GetReviewsFromMovieByIDFunc(ctx, movieID, limit, offset)
}

func (f *FakeReviewRepository) CreateReviewForMovieByID(ctx context.Context, review *domains.Review) (*domains.Review, error) {
	return f.CreateReviewForMovieByIDFunc(ctx, review)
}

type FakeBlockRepository struct {
	repositories.BlockRepository

	BlockUserFunc        func(ctx context.Context, userIDBlocker, userIDBlocked int64) error
	UnblockUserFunc      func(ctx context.Context, userIDBlocker, userIDBlocked int64) error
	IsBlockedBetweenFunc func(ctx context.Context, userID1, userID2 int64) (bool, error)
}

func (f *FakeBlockRepository) BlockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error {
	return f.BlockUserFunc(ctx, userIDBlocker, userIDBlocked)
}

func (f *FakeBlockRepository) UnblockUser(ctx context.Context, userIDBlocker, userIDBlocked int64) error {
	return f.UnblockUserFunc(ctx, userIDBlocker, userIDBlocked)
}

func (f *FakeBlockRepository) IsBlockedBetween(ctx context.Context, userID1, userID2 int64) (bool, error) {
	return f.IsBlockedBetweenFunc(ctx, userID1, userID2)
}
