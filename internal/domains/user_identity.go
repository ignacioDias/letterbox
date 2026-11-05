package domains

import "time"

type UserIdentity struct {
	ID             int64     `db:"id"`
	UserID         int64     `db:"user_id"`
	Provider       string    `db:"provider"`
	ProviderUserID string    `db:"provider_user_id"`
	Email          string    `db:"email"`
	CreatedAt      time.Time `db:"created_at"`
}

func NewUserIdentity(userID int64, provider, providerUserID, email string) *UserIdentity {
	return &UserIdentity{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
	}
}
