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
	CreateUserWithIdentity(ctx context.Context, user *domains.User, identity *domains.UserIdentity) error
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
	if err := insertIdentity(ctx, r.db, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

// CreateUserWithIdentity creates the user and its external identity atomically so
// a failed identity insert never leaves an orphaned account behind.
func (r *userIdentityRepository) CreateUserWithIdentity(ctx context.Context, user *domains.User, identity *domains.UserIdentity) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := insertUser(ctx, tx, user); err != nil {
		return err
	}
	identity.UserID = user.ID
	if err := insertIdentity(ctx, tx, identity); err != nil {
		return err
	}
	return tx.Commit()
}

func insertIdentity(ctx context.Context, q sqlx.QueryerContext, identity *domains.UserIdentity) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, NULLIF($4, '')::citext)
		RETURNING id, created_at
	`
	err := q.QueryRowxContext(ctx, query, identity.UserID, identity.Provider, identity.ProviderUserID, identity.Email).Scan(&identity.ID, &identity.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserIdentityExists
		}
		return err
	}
	return nil
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
