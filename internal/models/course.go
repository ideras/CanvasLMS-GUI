package models

// Course represents a Canvas course.
type Course struct {
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	CourseCode string      `json:"course_code"`
	Term       *CourseTerm `json:"term,omitempty"`
}

// CourseTerm represents the nested term object in a Course.
type CourseTerm struct {
	Name string `json:"name"`
}

// AssignmentGroup represents a Canvas assignment group (weighting category).
type AssignmentGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Enrollment represents a student enrollment in a course.
type Enrollment struct {
	UserID int               `json:"user_id"`
	Grades *EnrollmentGrades `json:"grades,omitempty"`
}

// EnrollmentGrades holds computed grade values for an enrollment.
type EnrollmentGrades struct {
	CurrentScore any `json:"current_score"` // float or ""
	FinalScore   any `json:"final_score"`   // float or ""
	FinalGrade   any `json:"final_grade"`   // string or ""
}
