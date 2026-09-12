package models

// User represents a Canvas user (typically a student in the context of a course).
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	SISUserID string `json:"sis_user_id"`
}
