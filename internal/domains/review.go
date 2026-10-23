package domains

import "time"

type Review struct {
	ID               int64     `json:"id" db:"id"`
	UserID           int64     `json:"user_id" db:"user_id"`
	MovieID          int32     `json:"movie_id" db:"movie_id"`
	Score            int16     `json:"score" db:"score"`
	Content          string    `json:"content" db:"content"`
	ContainsSpoilers bool      `json:"containsSpoilers" db:"contains_spoilers"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

func NewReview(userID int64, movieID int32, score int16, content string, containsSpoilers bool) *Review {
	return &Review{
		UserID:           userID,
		MovieID:          movieID,
		Score:            score,
		Content:          content,
		ContainsSpoilers: containsSpoilers,
	}
}
