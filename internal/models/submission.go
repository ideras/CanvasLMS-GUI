package models

// Submission represents a student submission for an assignment.
type Submission struct {
	AssignmentID      int                    `json:"assignment_id"`
	UserID            int                    `json:"user_id"`
	Score             interface{}            `json:"score"` // float64 or null
	SubmittedAt       *string                `json:"submitted_at"`
	WorkflowState     string                 `json:"workflow_state"`
	Late              bool                   `json:"late"`
	Grade             *string                `json:"grade"`
	GradedAt          *string                `json:"graded_at"`
	GraderID          *int                   `json:"grader_id"`
	Missing           bool                   `json:"missing"`
	Excused           bool                   `json:"excused"`
	SubmissionType    string                 `json:"submission_type"`
	Body              string                 `json:"body"`
	URL               string                 `json:"url"`
	PreviewURL        string                 `json:"preview_url"`
	Attempt           int                    `json:"attempt"`
	Attachments       []Attachment           `json:"attachments"`
	SubmissionHistory []SubmissionHistoryItem `json:"submission_history"`
}

// SubmissionHistoryItem represents one entry in a submission's history.
type SubmissionHistoryItem struct {
	Attachments    []Attachment         `json:"attachments"`
	SubmissionData []SubmissionDataItem `json:"submission_data"`
}

// SubmissionDataItem holds one question answer from a quiz submission.
type SubmissionDataItem struct {
	QuestionID interface{} `json:"question_id"` // int or string from Canvas
	Text       string      `json:"text"`
	Points     float64     `json:"points"`
	Correct    interface{} `json:"correct"` // string: "correct", "incorrect", "undefined"
}

// Attachment represents a file attached to a submission.
type Attachment struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content-type"`
}
