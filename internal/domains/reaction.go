package domains

import "time"

type Reaction struct {
	TargetID  int64     `json:"target_id" db:"target_id"` // The review or comment being reacted to
	UserID    int64     `json:"user_id" db:"user_id"`
	Value     int16     `json:"value" db:"value"` // 1 for like, -1 for dislike
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func NewReaction(targetID, userID int64, value int16) *Reaction {
	return &Reaction{
		TargetID: targetID,
		UserID:   userID,
		Value:    value,
	}
}
