package models

// Assignment represents a Canvas assignment.
type Assignment struct {
	ID                      int      `json:"id"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	DueAt                   *string  `json:"due_at"`
	LockAt                  *string  `json:"lock_at"`
	UnlockAt                *string  `json:"unlock_at"`
	PointsPossible          float64  `json:"points_possible"`
	AssignmentGroupID       *int     `json:"assignment_group_id"`
	SubmissionTypes         []string `json:"submission_types"`
	AllowedAttempts         *int     `json:"allowed_attempts"` // -1 = unlimited
	Published               bool     `json:"published"`
	WorkflowState           string   `json:"workflow_state"`
	HasSubmittedSubmissions bool     `json:"has_submitted_submissions"`
	NeedsGradingCount       int      `json:"needs_grading_count"`
	HTMLURL                 string   `json:"html_url"`
	Position                int      `json:"position"`
}

// CourseItem merges assignments and quizzes for the frontend course view.
type CourseItem struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Type              string  `json:"type"` // "assignment" or "quiz"
	DueAt             *string `json:"due_at"`
	Points            float64 `json:"points"`
	Published         bool    `json:"published"`
	NeedsGradingCount int     `json:"needs_grading_count"`
	QuizType          string  `json:"quiz_type,omitempty"`
	AssignmentID      int     `json:"assignment_id"` // for quizzes: the linked Canvas assignment ID for grade upload
	IsOldQuiz         bool    `json:"is_old_quiz"`   // true if this is an old-style Canvas quiz with downloadable questions
}

// CourseStats holds computed statistics for the Info tab.
type CourseStats struct {
	StudentCount   int `json:"student_count"`
	TotalItems     int `json:"total_items"`
	PublishedItems int `json:"published_items"`
	NeedsGrading   int `json:"needs_grading"`
}
