package services

import (
	"context"
	"errors"
	"letterbox/internal/cache"
	"letterbox/internal/domains"
	"letterbox/internal/repositories"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var followCacheGenerationCounter atomic.Uint64

const followCacheTTLSeconds = 3600

const (
	followKindFollowers = "followers"
	followKindFollowing = "following"
)

type FollowService interface {
	FollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error
	UnfollowUserByUsername(ctx context.Context, userIDFollower int64, usernameFollowing string) error
	GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error)
	GetFollowersCountByUsername(ctx context.Context, username string) (int, error)
	GetFollowingCountByUsername(ctx context.Context, username string) (int, error)
}

type followService struct {
	userRepository   repositories.UserRepository
	followRepository repositories.FollowRepository
	blockRepository  repositories.BlockRepository
	cache            cache.Cache
	cacheEnabled     bool
}

var ErrCannotFollowSelf = errors.New("cannot follow yourself")
var ErrCannotUnfollowSelf = errors.New("cannot unfollow yourself")

func NewFollowService(followRepo repositories.FollowRepository, userRepo repositories.UserRepository, blockRepo repositories.BlockRepository, caches ...cache.Cache) FollowService {
	followCache := cache.Cache(noopCache{})
	cacheEnabled := false
	if len(caches) > 0 && caches[0] != nil {
		followCache = caches[0]
		cacheEnabled = true
	}
	return &followService{
		followRepository: followRepo,
		userRepository:   userRepo,
		blockRepository:  blockRepo,
		cache:            followCache,
		cacheEnabled:     cacheEnabled,
	}
}

type noopCache struct{}

func (noopCache) Get(context.Context, string) (string, error) {
	return "", cache.ErrKeyNotFound
}

func (noopCache) Set(context.Context, string, string, int) error {
	return nil
}

func (noopCache) Delete(context.Context, []string) (int64, error) {
	return 0, nil
}

func (noopCache) CloseClient() {}

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
	if err := s.followRepository.FollowUser(ctx, userIDFollower, followingID); err != nil {
		return err
	}
	s.invalidateFollowCache(ctx, userIDFollower, usernameFollowing)
	return nil
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
	if err := s.followRepository.UnfollowUser(ctx, userIDFollower, followingID); err != nil {
		return err
	}
	s.invalidateFollowCache(ctx, userIDFollower, usernameFollowing)
	return nil
}

func (s *followService) GetFollowersByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	generation := s.getFollowCacheGeneration(ctx, followKindFollowers, username)
	key := getKeyForFollowList(followKindFollowers, username, limit, offset, generation)
	users, found, err := s.getPublicUsersFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return users, nil
	}
	users, err = s.followRepository.GetFollowersByUsername(ctx, username, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cachePublicUsers(ctx, key, users)
	return users, nil
}

func (s *followService) GetFollowingByUsername(ctx context.Context, username string, limit, offset int) ([]*domains.PublicUser, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	generation := s.getFollowCacheGeneration(ctx, followKindFollowing, username)
	key := getKeyForFollowList(followKindFollowing, username, limit, offset, generation)
	users, found, err := s.getPublicUsersFromCache(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return users, nil
	}
	users, err = s.followRepository.GetFollowingByUsername(ctx, username, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cachePublicUsers(ctx, key, users)
	return users, nil
}

func (s *followService) GetFollowersCountByUsername(ctx context.Context, username string) (int, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return 0, ErrInvalidUsername
	}
	return s.getFollowCount(ctx, followKindFollowers, username, func(userID int64) (int, error) {
		return s.followRepository.GetFollowersCountByUserID(ctx, userID)
	})
}

func (s *followService) GetFollowingCountByUsername(ctx context.Context, username string) (int, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return 0, ErrInvalidUsername
	}
	return s.getFollowCount(ctx, followKindFollowing, username, func(userID int64) (int, error) {
		return s.followRepository.GetFollowingCountByUserID(ctx, userID)
	})
}

func (s *followService) getFollowCount(ctx context.Context, kind, username string, countByUserID func(userID int64) (int, error)) (int, error) {
	key := getKeyForFollowCount(kind, username)
	count, found, err := s.getCountFromCache(ctx, key)
	if err != nil {
		return 0, err
	}
	if found {
		return count, nil
	}
	userID, err := s.userRepository.GetUserIDByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	count, err = countByUserID(userID)
	if err != nil {
		return 0, err
	}
	s.cacheCount(ctx, key, count)
	return count, nil
}

func (s *followService) getFollowCacheGeneration(ctx context.Context, kind, username string) string {
	key := getKeyForFollowGeneration(kind, username)
	generation, err := s.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrKeyNotFound) {
			log.Printf("Error reading %s cache generation for %s: %v", kind, username, err)
		}
		return "0"
	}
	if generation == "" {
		return "0"
	}
	return generation
}

func (s *followService) getPublicUsersFromCache(ctx context.Context, key string) ([]*domains.PublicUser, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		log.Printf("Error reading follow list from cache: %v", err)
		return nil, false, nil
	}
	users, err := generatePublicUsersFromCacheResult(result)
	if err == nil {
		return users, true, nil
	}
	log.Printf("Error decoding cached follow list: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached follow list: %v", err)
	}
	return nil, false, nil
}

func (s *followService) cachePublicUsers(ctx context.Context, key string, users []*domains.PublicUser) {
	value, err := generateCacheValueFromPublicUsers(users)
	if err != nil {
		log.Printf("Error encoding follow list for cache: %v", err)
		return
	}
	if err := s.cache.Set(ctx, key, value, followCacheTTLSeconds); err != nil {
		log.Printf("Error saving follow list to cache: %v", err)
	}
}

func (s *followService) getCountFromCache(ctx context.Context, key string) (int, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return 0, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, false, ctxErr
		}
		log.Printf("Error reading follow count from cache: %v", err)
		return 0, false, nil
	}
	count, err := strconv.Atoi(result)
	if err == nil {
		return count, true, nil
	}
	log.Printf("Error decoding cached follow count: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached follow count: %v", err)
	}
	return 0, false, nil
}

func (s *followService) cacheCount(ctx context.Context, key string, count int) {
	if err := s.cache.Set(ctx, key, strconv.Itoa(count), followCacheTTLSeconds); err != nil {
		log.Printf("Error saving follow count to cache: %v", err)
	}
}

func (s *followService) invalidateFollowCache(ctx context.Context, followerID int64, followedUsername string) {
	if !s.cacheEnabled {
		return
	}
	followedUsername = strings.TrimSpace(followedUsername)
	keys := []string{getKeyForFollowCount(followKindFollowers, followedUsername)}
	s.resetFollowCacheGeneration(ctx, followKindFollowers, followedUsername)

	follower, err := s.userRepository.GetUserByID(ctx, followerID)
	if err != nil {
		log.Printf("Error resolving follower %d for cache invalidation: %v", followerID, err)
	} else {
		followerUsername := strings.TrimSpace(follower.Username)
		keys = append(keys, getKeyForFollowCount(followKindFollowing, followerUsername))
		s.resetFollowCacheGeneration(ctx, followKindFollowing, followerUsername)
	}

	if _, err := s.cache.Delete(ctx, keys); err != nil {
		log.Printf("Error deleting follow count cache: %v", err)
	}
}

func (s *followService) resetFollowCacheGeneration(ctx context.Context, kind, username string) {
	generation := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(followCacheGenerationCounter.Add(1), 10)
	key := getKeyForFollowGeneration(kind, username)
	if err := s.cache.Set(ctx, key, generation, followCacheTTLSeconds); err != nil {
		log.Printf("Error resetting %s cache generation for %s: %v", kind, username, err)
	}
}
