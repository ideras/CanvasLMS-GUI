package models

// GradeEntry is one row in the batch grade submission payload.
type GradeEntry struct {
	StudentID int    `json:"student_id"`
	Grade     string `json:"posted_grade"`
	Comment   string `json:"text_comment,omitempty"`
}

// BatchGradePayload is the request body for the update_grades endpoint.
type BatchGradePayload struct {
	GradeData map[string]GradeData `json:"grade_data"`
}

// GradeData contains the grade and optional comment for one student.
type GradeData struct {
	PostedGrade string `json:"posted_grade"`
	TextComment string `json:"text_comment,omitempty"`
}

// BatchProgress mirrors the Canvas Progress object returned by the progress URL.
type BatchProgress struct {
	ID             int     `json:"id"`
	WorkflowState  string  `json:"workflow_state"` // "queued" | "running" | "completed" | "failed"
	Completion     float64 `json:"completion"`     // 0–100
	Message        string  `json:"message"`        // populated on failure
}
