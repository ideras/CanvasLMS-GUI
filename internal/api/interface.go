package api

import (
	"context"

	"canvaslms-gui/internal/models"
)

// CanvasClient defines the interface for all Canvas API operations.
// Every method accepts a context for cancellation and timeout control.
type CanvasClient interface {
	// Courses
	GetAllCourses(ctx context.Context) ([]models.Course, error)

	// Folders
	GetFoldersForCourse(ctx context.Context, courseID int) ([]models.Folder, error)
	EnsureCourseFolder(ctx context.Context, courseID int, folderPath string) (*models.Folder, error)

	// Assignments
	GetAssignmentsForCourse(ctx context.Context, courseID int) ([]models.Assignment, error)
	CreateAssignment(ctx context.Context, courseID int, data map[string]any) (*models.Assignment, error)
	EditAssignment(ctx context.Context, courseID, assignmentID int, data map[string]any) (*models.Assignment, error)
	DeleteAssignment(ctx context.Context, courseID, assignmentID int) error

	// Assignment Groups
	GetAssignmentGroupsForCourse(ctx context.Context, courseID int) ([]models.AssignmentGroup, error)
	CreateAssignmentGroup(ctx context.Context, courseID int, name string) (*models.AssignmentGroup, error)

	// Submissions
	GetSubmissionsForAssignment(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error)
	GetAssignmentSubmissionsWithAttachments(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error)
	GetSubmissionsForStudent(ctx context.Context, courseID, studentID int) ([]models.Submission, error)
	GetAllSubmissionsForCourse(ctx context.Context, courseID int) ([]models.Submission, error)

	// Users / Enrollments
	GetStudentsForCourse(ctx context.Context, courseID int) ([]models.User, error)
	GetEnrollmentsForCourse(ctx context.Context, courseID int) ([]models.Enrollment, error)

	// Quizzes
	GetQuizzesForCourse(ctx context.Context, courseID int) ([]models.Quiz, error)
	GetQuizQuestions(ctx context.Context, courseID, quizID int) ([]models.QuizQuestion, error)
	GetQuizSubmissions(ctx context.Context, courseID, quizID int) (*models.QuizSubmissionList, error)

	// Announcements
	GetAnnouncements(ctx context.Context, courseID int) ([]models.Announcement, error)
	CreateAnnouncement(ctx context.Context, courseID int, title, message string) (*models.Announcement, error)
	UpdateAnnouncement(ctx context.Context, courseID, topicID int, title, message string) (*models.Announcement, error)
	DeleteAnnouncement(ctx context.Context, courseID, topicID int) error

	// Files
	DownloadFile(ctx context.Context, fileURL, localPath string) error
	GetFileDownloadURL(ctx context.Context, fileID int) (string, error)
	UploadFileToCourse(ctx context.Context, filePath string, courseID, parentFolderID int) (*models.FileInfo, error)

	// Grades
	SubmitGrade(ctx context.Context, courseID, assignmentID, studentID int, grade string, comment string) error
	BatchSubmitGrades(ctx context.Context, courseID, assignmentID int, entries []models.GradeEntry) (string, error)
	PollBatchProgress(ctx context.Context, progressURL string) (<-chan models.BatchProgress, error)
	QueryProgress(ctx context.Context, progressID int) (*models.BatchProgress, error)
}
