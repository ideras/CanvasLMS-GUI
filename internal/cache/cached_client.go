package cache

import (
	"context"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/models"
)

// CachedCanvasClient wraps any api.CanvasClient and transparently caches
// read-only calls. Write/mutation calls are always forwarded to the real client.
//
// Drop-in replacement: it satisfies the api.CanvasClient interface
type CachedCanvasClient struct {
	real  api.CanvasClient
	store Store
}

func NewCachedCanvasClient(real api.CanvasClient, store Store) api.CanvasClient {
	return &CachedCanvasClient{real: real, store: store}
}

func (c *CachedCanvasClient) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	var courses []models.Course
	if hit, _ := c.store.Get(KeyCourses(), &courses); hit {
		return courses, nil
	}
	courses, err := c.real.GetAllCourses(ctx)
	if err == nil {
		_ = c.store.Set(KeyCourses(), courses, TTLCourses)
	}
	return courses, err
}

func (c *CachedCanvasClient) GetFoldersForCourse(ctx context.Context, courseID int) ([]models.Folder, error) {
	var folders []models.Folder
	if hit, _ := c.store.Get(KeyFolders(courseID), &folders); hit {
		return folders, nil
	}
	folders, err := c.real.GetFoldersForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyFolders(courseID), folders, TTLFolders)
	}
	return folders, err
}

// EnsureCourseFolder is a mutation — always forwarded live.
func (c *CachedCanvasClient) EnsureCourseFolder(ctx context.Context, courseID int, folderPath string) (*models.Folder, error) {
	return c.real.EnsureCourseFolder(ctx, courseID, folderPath)
}

func (c *CachedCanvasClient) GetAssignmentsForCourse(ctx context.Context, courseID int) ([]models.Assignment, error) {
	var assignments []models.Assignment
	if hit, _ := c.store.Get(KeyAssignments(courseID), &assignments); hit {
		return assignments, nil
	}
	assignments, err := c.real.GetAssignmentsForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyAssignments(courseID), assignments, TTLAssignments)
	}
	return assignments, err
}

// CreateAssignment and DeleteAssignment are mutations — always live.
// They also invalidate the assignment cache for the affected course.
func (c *CachedCanvasClient) CreateAssignment(ctx context.Context, courseID int, data map[string]any) (*models.Assignment, error) {
	result, err := c.real.CreateAssignment(ctx, courseID, data)
	if err == nil {
		_ = c.store.Delete(KeyAssignments(courseID))
	}
	return result, err
}

func (c *CachedCanvasClient) DeleteAssignment(ctx context.Context, courseID, assignmentID int) error {
	err := c.real.DeleteAssignment(ctx, courseID, assignmentID)
	if err == nil {
		_ = c.store.Delete(KeyAssignments(courseID))
	}
	return err
}

// EditAssignment is a mutation — always forwarded live.
// Invalidates the assignment cache for the affected course.
func (c *CachedCanvasClient) EditAssignment(ctx context.Context, courseID, assignmentID int, data map[string]any) (*models.Assignment, error) {
	result, err := c.real.EditAssignment(ctx, courseID, assignmentID, data)
	if err == nil {
		_ = c.store.Delete(KeyAssignments(courseID))
	}
	return result, err
}

func (c *CachedCanvasClient) GetAssignmentGroupsForCourse(ctx context.Context, courseID int) ([]models.AssignmentGroup, error) {
	var groups []models.AssignmentGroup
	if hit, _ := c.store.Get(KeyAssignmentGroups(courseID), &groups); hit {
		return groups, nil
	}
	groups, err := c.real.GetAssignmentGroupsForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyAssignmentGroups(courseID), groups, TTLAssignments)
	}
	return groups, err
}

func (c *CachedCanvasClient) CreateAssignmentGroup(ctx context.Context, courseID int, name string) (*models.AssignmentGroup, error) {
	result, err := c.real.CreateAssignmentGroup(ctx, courseID, name)
	if err == nil {
		_ = c.store.Delete(KeyAssignmentGroups(courseID))
	}
	return result, err
}

func (c *CachedCanvasClient) GetSubmissionsForAssignment(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	var submissions []models.Submission
	key := KeySubmissions(courseID, assignmentID)
	if hit, _ := c.store.Get(key, &submissions); hit {
		return submissions, nil
	}
	submissions, err := c.real.GetSubmissionsForAssignment(ctx, courseID, assignmentID)
	if err == nil {
		_ = c.store.Set(key, submissions, TTLSubmissions)
	}
	return submissions, err
}

func (c *CachedCanvasClient) GetAssignmentSubmissionsWithAttachments(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	var submissions []models.Submission
	key := KeySubmissionsWithAttachments(courseID, assignmentID)
	if hit, _ := c.store.Get(key, &submissions); hit {
		return submissions, nil
	}
	submissions, err := c.real.GetAssignmentSubmissionsWithAttachments(ctx, courseID, assignmentID)
	if err == nil {
		_ = c.store.Set(key, submissions, TTLSubmissions)
	}
	return submissions, err
}

func (c *CachedCanvasClient) GetSubmissionsForStudent(ctx context.Context, courseID, studentID int) ([]models.Submission, error) {
	var submissions []models.Submission
	key := KeyStudentSubmissions(courseID, studentID)
	if hit, _ := c.store.Get(key, &submissions); hit {
		return submissions, nil
	}
	submissions, err := c.real.GetSubmissionsForStudent(ctx, courseID, studentID)
	if err == nil {
		_ = c.store.Set(key, submissions, TTLSubmissions)
	}
	return submissions, err
}

func (c *CachedCanvasClient) GetAllSubmissionsForCourse(ctx context.Context, courseID int) ([]models.Submission, error) {
	var submissions []models.Submission
	key := KeyAllSubmissions(courseID)
	if hit, _ := c.store.Get(key, &submissions); hit {
		return submissions, nil
	}
	submissions, err := c.real.GetAllSubmissionsForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(key, submissions, TTLSubmissions)
	}
	return submissions, err
}

func (c *CachedCanvasClient) GetStudentsForCourse(ctx context.Context, courseID int) ([]models.User, error) {
	var users []models.User
	if hit, _ := c.store.Get(KeyStudents(courseID), &users); hit {
		return users, nil
	}
	users, err := c.real.GetStudentsForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyStudents(courseID), users, TTLStudents)
	}
	return users, err
}

func (c *CachedCanvasClient) GetEnrollmentsForCourse(ctx context.Context, courseID int) ([]models.Enrollment, error) {
	var enrolments []models.Enrollment
	if hit, _ := c.store.Get(KeyEnrolments(courseID), &enrolments); hit {
		return enrolments, nil
	}
	enrolments, err := c.real.GetEnrollmentsForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyEnrolments(courseID), enrolments, TTLEnrolments)
	}
	return enrolments, err
}

func (c *CachedCanvasClient) GetQuizzesForCourse(ctx context.Context, courseID int) ([]models.Quiz, error) {
	var quizzes []models.Quiz
	if hit, _ := c.store.Get(KeyQuizzes(courseID), &quizzes); hit {
		return quizzes, nil
	}
	quizzes, err := c.real.GetQuizzesForCourse(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyQuizzes(courseID), quizzes, TTLQuizzes)
	}
	return quizzes, err
}

func (c *CachedCanvasClient) GetQuizQuestions(ctx context.Context, courseID, quizID int) ([]models.QuizQuestion, error) {
	var questions []models.QuizQuestion
	if hit, _ := c.store.Get(KeyQuizQuestions(courseID, quizID), &questions); hit {
		return questions, nil
	}
	questions, err := c.real.GetQuizQuestions(ctx, courseID, quizID)
	if err == nil {
		_ = c.store.Set(KeyQuizQuestions(courseID, quizID), questions, TTLQuestions)
	}
	return questions, err
}

func (c *CachedCanvasClient) GetQuizSubmissions(ctx context.Context, courseID, quizID int) (*models.QuizSubmissionList, error) {
	var list models.QuizSubmissionList
	if hit, _ := c.store.Get(KeyQuizSubmissions(courseID, quizID), &list); hit {
		return &list, nil
	}
	result, err := c.real.GetQuizSubmissions(ctx, courseID, quizID)
	if err == nil {
		_ = c.store.Set(KeyQuizSubmissions(courseID, quizID), result, TTLSubmissions)
	}
	return result, err
}

func (c *CachedCanvasClient) DownloadFile(ctx context.Context, fileURL, localPath string) error {
	return c.real.DownloadFile(ctx, fileURL, localPath)
}

func (c *CachedCanvasClient) UploadFileToCourse(ctx context.Context, filePath string, courseID, parentFolderID int) (*models.FileInfo, error) {
	return c.real.UploadFileToCourse(ctx, filePath, courseID, parentFolderID)
}

func (c *CachedCanvasClient) GetFileDownloadURL(ctx context.Context, fileID int) (string, error) {
	return c.real.GetFileDownloadURL(ctx, fileID)
}

// --- Announcements ---

func (c *CachedCanvasClient) GetAnnouncements(ctx context.Context, courseID int) ([]models.Announcement, error) {
	var announcements []models.Announcement
	if hit, _ := c.store.Get(KeyAnnouncements(courseID), &announcements); hit {
		return announcements, nil
	}
	result, err := c.real.GetAnnouncements(ctx, courseID)
	if err == nil {
		_ = c.store.Set(KeyAnnouncements(courseID), result, TTLAnnouncements)
	}
	return result, err
}

func (c *CachedCanvasClient) CreateAnnouncement(ctx context.Context, courseID int, title, message string) (*models.Announcement, error) {
	result, err := c.real.CreateAnnouncement(ctx, courseID, title, message)
	if err == nil {
		_ = c.store.Delete(KeyAnnouncements(courseID))
	}
	return result, err
}

func (c *CachedCanvasClient) UpdateAnnouncement(ctx context.Context, courseID, topicID int, title, message string) (*models.Announcement, error) {
	result, err := c.real.UpdateAnnouncement(ctx, courseID, topicID, title, message)
	if err == nil {
		_ = c.store.Delete(KeyAnnouncements(courseID))
	}
	return result, err
}

func (c *CachedCanvasClient) DeleteAnnouncement(ctx context.Context, courseID, topicID int) error {
	err := c.real.DeleteAnnouncement(ctx, courseID, topicID)
	if err == nil {
		_ = c.store.Delete(KeyAnnouncements(courseID))
	}
	return err
}

func (c *CachedCanvasClient) SubmitGrade(ctx context.Context, courseID, assignmentID, studentID int, grade, comment string) error {
	// Invalidate submission cache for this assignment after a successful grade.
	err := c.real.SubmitGrade(ctx, courseID, assignmentID, studentID, grade, comment)
	if err == nil {
		_ = c.store.Delete(KeySubmissions(courseID, assignmentID))
		_ = c.store.Delete(KeySubmissionsWithAttachments(courseID, assignmentID))
		_ = c.store.Delete(KeyStudentSubmissions(courseID, studentID))
	}
	return err
}

func (c *CachedCanvasClient) BatchSubmitGrades(ctx context.Context, courseID, assignmentID int, entries []models.GradeEntry) (string, error) {
	result, err := c.real.BatchSubmitGrades(ctx, courseID, assignmentID, entries)
	if err == nil {
		_ = c.store.Delete(KeySubmissions(courseID, assignmentID))
		_ = c.store.Delete(KeySubmissionsWithAttachments(courseID, assignmentID))
	}
	return result, err
}

func (c *CachedCanvasClient) PollBatchProgress(ctx context.Context, progressURL string) (<-chan models.BatchProgress, error) {
	return c.real.PollBatchProgress(ctx, progressURL)
}

func (c *CachedCanvasClient) QueryProgress(ctx context.Context, progressID int) (*models.BatchProgress, error) {
	return c.real.QueryProgress(ctx, progressID)
}
