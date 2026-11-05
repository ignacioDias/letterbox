package repositories

import (
	"context"
	"database/sql"
	"errors"
	"letterbox/internal/domains"

	"github.com/jmoiron/sqlx"
)

type UserIdentityRepository interface {
	GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*domains.UserIdentity, error)
	GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*domains.UserIdentity, error)
	CreateIdentity(ctx context.Context, identity *domains.UserIdentity) (*domains.UserIdentity, error)
}

type userIdentityRepository struct {
	db *sqlx.DB
}

func NewUserIdentityRepository(db *sqlx.DB) UserIdentityRepository {
	return &userIdentityRepository{
		db: db,
	}
}

func (r *userIdentityRepository) GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*domains.UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, COALESCE(email::text, '') AS email, created_at
		FROM user_identities
		WHERE provider = $1 AND provider_user_id = $2
	`
	return r.getIdentity(ctx, query, provider, providerUserID)
}

func (r *userIdentityRepository) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*domains.UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, COALESCE(email::text, '') AS email, created_at
		FROM user_identities
		WHERE user_id = $1 AND provider = $2
	`
	return r.getIdentity(ctx, query, userID, provider)
}

func (r *userIdentityRepository) CreateIdentity(ctx context.Context, identity *domains.UserIdentity) (*domains.UserIdentity, error) {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, NULLIF($4, '')::citext)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, identity.UserID, identity.Provider, identity.ProviderUserID, identity.Email).Scan(&identity.ID, &identity.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserIdentityExists
		}
		return nil, err
	}
	return identity, nil
}

func (r *userIdentityRepository) getIdentity(ctx context.Context, query string, args ...any) (*domains.UserIdentity, error) {
	var identity domains.UserIdentity
	if err := r.db.GetContext(ctx, &identity, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserIdentityNotFound
		}
		return nil, err
	}
	return &identity, nil
}
