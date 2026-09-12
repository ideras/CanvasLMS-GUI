package models

// Quiz represents a Canvas quiz (supports both old and New Quizzes/LTI tools).
type Quiz struct {
	// Fields shared with Assignment
	ID                      int      `json:"id"`
	Title                   string   `json:"title"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	DueAt                   *string  `json:"due_at"`
	LockAt                  *string  `json:"lock_at"`
	UnlockAt                *string  `json:"unlock_at"`
	PointsPossible          float64  `json:"points_possible"`
	AssignmentGroupID       *int     `json:"assignment_group_id"`
	SubmissionTypes         []string `json:"submission_types"`
	Published               bool     `json:"published"`
	WorkflowState           string   `json:"workflow_state"`
	HasSubmittedSubmissions bool     `json:"has_submitted_submissions"`
	HTMLURL                 string   `json:"html_url"`
	AssignmentID            *int     `json:"assignment_id"`

	// Quiz-specific fields
	QuizType              string      `json:"quiz_type"`
	QuestionCount         int         `json:"question_count"`
	TimeLimit             *int        `json:"time_limit"` // minutes
	ShuffleAnswers        bool        `json:"shuffle_answers"`
	ShowCorrectAnswers    *bool       `json:"show_correct_answers"` // nil defaults to true
	OneQuestionAtATime    bool        `json:"one_question_at_a_time"`
	CantGoBack            bool        `json:"cant_go_back"`
	AllowedAttempts       *int        `json:"allowed_attempts"` // -1 = unlimited
	AccessCode            *string     `json:"access_code"`
	ScoringPolicy         string      `json:"scoring_policy"`
	HideResults           interface{} `json:"hide_results"`
	LockedForUser         bool        `json:"locked_for_user"`
	LockExplanation       *string     `json:"lock_explanation"`
}

// QuizQuestion represents a single question in a quiz.
type QuizQuestion struct {
	ID                int          `json:"id"`
	QuestionName      string       `json:"question_name"`
	QuestionType      string       `json:"question_type"`
	PointsPossible    float64      `json:"points_possible"`
	QuestionText      string       `json:"question_text"`
	Answers           []QuizAnswer `json:"answers"`
	CorrectComments   *string      `json:"correct_comments"`
	IncorrectComments *string      `json:"incorrect_comments"`
	NeutralComments   *string      `json:"neutral_comments"`
}

// QuizAnswer represents an answer option for a quiz question.
type QuizAnswer struct {
	Text   string `json:"text"`
	Weight int    `json:"weight"` // >0 indicates correct
}

// QuizSubmission represents a student's submission for a quiz.
type QuizSubmission struct {
	ID                 int     `json:"id"`
	UserID             int     `json:"user_id"`
	StartedAt          *string `json:"started_at"`
	FinishedAt         *string `json:"finished_at"`
	Score              float64 `json:"score"`
	KeptScore          float64 `json:"kept_score"`
	Attempt            int     `json:"attempt"`
	TimeSpent          int     `json:"time_spent"` // seconds
	WorkflowState      string  `json:"workflow_state"`
	HTMLURL            string  `json:"html_url"`
	QuizID             int     `json:"quiz_id"`
	QuizPointsPossible float64 `json:"quiz_points_possible"`
}

// QuizSubmissionList is the wrapper returned by the Canvas quiz submissions API.
type QuizSubmissionList struct {
	QuizSubmissions []QuizSubmission `json:"quiz_submissions"`
}
