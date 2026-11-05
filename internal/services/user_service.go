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

var userCacheGenerationCounter atomic.Uint64

const userCacheTTLSeconds = 3600

type UserService interface {
	GetUserByID(ctx context.Context, userID int64) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	SearchUsers(ctx context.Context, search string, limit int, offset int) ([]domains.PublicUser, error)

	UpdateUserByID(ctx context.Context, userID int64, username string, email string, country string, profilePicture string) error
	UpdateUserPasswordByID(ctx context.Context, userID int64, currentSessionID string, currentPassword string, newPassword string) error
	DeleteUserByID(ctx context.Context, userID int64) error
}

type userService struct {
	userRepository    repositories.UserRepository
	sessionRepository repositories.SessionRepository
	cache             cache.Cache
	cacheEnabled      bool
}

var ErrInvalidNewPassword = errors.New("invalid new password format")

func NewUserService(userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository, caches ...cache.Cache) UserService {
	userCache := cache.Cache(noopCache{})
	cacheEnabled := false
	if len(caches) > 0 && caches[0] != nil {
		userCache = caches[0]
		cacheEnabled = true
	}
	return &userService{
		userRepository:    userRepo,
		sessionRepository: sessionRepo,
		cache:             userCache,
		cacheEnabled:      cacheEnabled,
	}
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*domains.User, error) {
	key := getKeyForUserByID(userID)
	user, found, err := s.getCachedUser(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return user, nil
	}
	user, err = s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.cacheUser(ctx, key, user)
	return user, nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*domains.User, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	key := getKeyForUserByUsername(username)
	user, found, err := s.getCachedUser(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return user, nil
	}
	user, err = s.userRepository.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	s.cacheUser(ctx, key, user)
	return user, nil
}

func (s *userService) SearchUsers(ctx context.Context, search string, limit int, offset int) ([]domains.PublicUser, error) {
	search = strings.TrimSpace(search)
	if search == "" {
		return nil, ErrInvalidSearchQuery
	}
	generation := s.getCacheGeneration(ctx, getKeyForUserSearchGeneration())
	key := getKeyForUserSearch(search, limit, offset, generation)
	users, found, err := s.getCachedPublicUsers(ctx, key)
	if err != nil {
		return nil, err
	}
	if found {
		return users, nil
	}
	users, err = s.userRepository.SearchUsers(ctx, search, limit, offset)
	if err != nil {
		return nil, err
	}
	s.cachePublicUsers(ctx, key, users)
	return users, nil
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
	if err := s.userRepository.UpdateUser(ctx, newUser); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID, user.Username, username)
	return nil
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
	var username string
	if s.cacheEnabled {
		user, err := s.userRepository.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}
		username = user.Username
	}
	if err := s.userRepository.DeleteUserByID(ctx, userID); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID, username)
	return nil
}

func newUserCacheGeneration() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatUint(userCacheGenerationCounter.Add(1), 10)
}

func resetUserSearchCacheGeneration(ctx context.Context, userCache cache.Cache) {
	if err := userCache.Set(ctx, getKeyForUserSearchGeneration(), newUserCacheGeneration(), userCacheTTLSeconds); err != nil {
		log.Printf("Error resetting user search cache generation: %v", err)
	}
}

func (s *userService) getCacheGeneration(ctx context.Context, key string) string {
	generation, err := s.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrKeyNotFound) {
			log.Printf("Error reading user cache generation: %v", err)
		}
		return "0"
	}
	if generation == "" {
		return "0"
	}
	return generation
}

func (s *userService) getCachedUser(ctx context.Context, key string) (*domains.User, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		log.Printf("Error reading user from cache: %v", err)
		return nil, false, nil
	}
	user, err := generateUserFromCacheResult(result)
	if err == nil {
		return user, true, nil
	}
	log.Printf("Error decoding cached user: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached user: %v", err)
	}
	return nil, false, nil
}

func (s *userService) cacheUser(ctx context.Context, key string, user *domains.User) {
	value, err := generateCacheValueFromUser(user)
	if err != nil {
		log.Printf("Error encoding user for cache: %v", err)
		return
	}
	if err := s.cache.Set(ctx, key, value, userCacheTTLSeconds); err != nil {
		log.Printf("Error saving user to cache: %v", err)
	}
}

func (s *userService) getCachedPublicUsers(ctx context.Context, key string) ([]domains.PublicUser, bool, error) {
	result, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, false, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		log.Printf("Error reading user search results from cache: %v", err)
		return nil, false, nil
	}
	users, err := generatePublicUserListFromCacheResult(result)
	if err == nil {
		return users, true, nil
	}
	log.Printf("Error decoding cached user search results: %v", err)
	if _, err := s.cache.Delete(ctx, []string{key}); err != nil {
		log.Printf("Error deleting corrupt cached user search results: %v", err)
	}
	return nil, false, nil
}

func (s *userService) cachePublicUsers(ctx context.Context, key string, users []domains.PublicUser) {
	value, err := generateCacheValueFromPublicUserList(users)
	if err != nil {
		log.Printf("Error encoding user search results for cache: %v", err)
		return
	}
	if err := s.cache.Set(ctx, key, value, userCacheTTLSeconds); err != nil {
		log.Printf("Error saving user search results to cache: %v", err)
	}
}

func (s *userService) invalidateUserCache(ctx context.Context, userID int64, usernames ...string) {
	if !s.cacheEnabled {
		return
	}
	keys := []string{getKeyForUserByID(userID)}
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username != "" {
			keys = append(keys, getKeyForUserByUsername(username))
		}
	}
	if _, err := s.cache.Delete(ctx, keys); err != nil {
		log.Printf("Error deleting user cache: %v", err)
	}
	resetUserSearchCacheGeneration(ctx, s.cache)
}
