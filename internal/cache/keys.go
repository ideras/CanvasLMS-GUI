package cache

import (
	"fmt"
	"time"
)

// TTL constants — tuned to how frequently Canvas data changes.
const (
	TTLCourses     = 24 * time.Hour  // Courses change once per semester
	TTLStudents    = 6 * time.Hour   // Enrolments are stable mid-semester
	TTLEnrolments  = 6 * time.Hour
	TTLAssignments = 1 * time.Hour   // Professors edit occasionally
	TTLQuizzes     = 1 * time.Hour
	TTLQuestions   = 1 * time.Hour
	TTLFolders     = 1 * time.Hour
	TTLSubmissions    = 15 * time.Minute // Students are actively submitting
	TTLAnnouncements  = 5 * time.Minute  // Announcements change occasionally
)

// Key builders — all course-scoped keys use the prefix "course:<id>:"
// so a single Delete("course:123:%") invalidates everything for that course.

func KeyCourses() string { return "courses:all" }

func KeyAssignments(courseID int) string {
	return fmt.Sprintf("course:%d:assignments", courseID)
}

func KeyStudents(courseID int) string {
	return fmt.Sprintf("course:%d:students", courseID)
}

func KeyEnrolments(courseID int) string {
	return fmt.Sprintf("course:%d:enrolments", courseID)
}

func KeyQuizzes(courseID int) string {
	return fmt.Sprintf("course:%d:quizzes", courseID)
}

func KeyQuizQuestions(courseID, quizID int) string {
	return fmt.Sprintf("course:%d:quiz:%d:questions", courseID, quizID)
}

func KeyQuizSubmissions(courseID, quizID int) string {
	return fmt.Sprintf("course:%d:quiz:%d:submissions", courseID, quizID)
}

func KeyAssignmentGroups(courseID int) string {
	return fmt.Sprintf("course:%d:assignment_groups", courseID)
}

func KeyFolders(courseID int) string {
	return fmt.Sprintf("course:%d:folders", courseID)
}

func KeySubmissions(courseID, assignmentID int) string {
	return fmt.Sprintf("course:%d:assignment:%d:submissions", courseID, assignmentID)
}

func KeySubmissionsWithAttachments(courseID, assignmentID int) string {
	return fmt.Sprintf("course:%d:assignment:%d:submissions_with_attachments", courseID, assignmentID)
}

func KeyStudentSubmissions(courseID, studentID int) string {
	return fmt.Sprintf("course:%d:student:%d:submissions", courseID, studentID)
}

func KeyAllSubmissions(courseID int) string {
	return fmt.Sprintf("course:%d:all_submissions", courseID)
}

func KeyAnnouncements(courseID int) string {
	return fmt.Sprintf("course:%d:announcements", courseID)
}

// PatternCourse returns a LIKE pattern that matches every key scoped to courseID.
func PatternCourse(courseID int) string {
	return fmt.Sprintf("course:%d:%%", courseID)
}
