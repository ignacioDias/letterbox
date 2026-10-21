package domains

type User struct {
	ID                int64  `json:"id" db:"id"`
	Username          string `json:"username" db:"username"`
	Email             string `json:"email" db:"email"`
	PasswordHash      string `json:"-" db:"password_hash"`
	ProfilePictureURL string `json:"profilePictureURL" db:"profile_picture_url"`
	Country           string `json:"country" db:"country"`
}

func NewUser(username, email, passwordHash, profilePictureURL, country string) *User {
	return &User{
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		ProfilePictureURL: profilePictureURL,
		Country:           country,
	}
}
