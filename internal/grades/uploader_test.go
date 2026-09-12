package grades_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"canvaslms-gui/internal/grades"
	"canvaslms-gui/internal/models"
)

// --- Mocks ---

type mockClient struct {
	students     []models.User
	folders      map[string]*models.Folder
	uploaded     map[string]int // path → fileID
	gradeEntries []models.GradeEntry
	progressURL  string
	failProgress bool
	mu           sync.Mutex
}

func newMockClient() *mockClient {
	return &mockClient{
		students: []models.User{
			{ID: 1001, Name: "Alice"},
			{ID: 1002, Name: "Bob"},
		},
		folders:     make(map[string]*models.Folder),
		uploaded:    make(map[string]int),
		progressURL: "https://canvas.test/api/v1/progress/42",
	}
}

func (m *mockClient) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	return nil, nil
}

func (m *mockClient) GetFoldersForCourse(ctx context.Context, courseID int) ([]models.Folder, error) {
	return nil, nil
}

func (m *mockClient) EnsureCourseFolder(ctx context.Context, courseID int, folderPath string) (*models.Folder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.folders[folderPath]; ok {
		return f, nil
	}
	f := &models.Folder{ID: 99, Name: folderPath}
	m.folders[folderPath] = f
	return f, nil
}

func (m *mockClient) GetAssignmentsForCourse(ctx context.Context, courseID int) ([]models.Assignment, error) {
	return nil, nil
}

func (m *mockClient) CreateAssignment(ctx context.Context, courseID int, data map[string]any) (*models.Assignment, error) {
	return nil, nil
}

func (m *mockClient) EditAssignment(ctx context.Context, courseID, assignmentID int, data map[string]any) (*models.Assignment, error) {
	return nil, nil
}

func (m *mockClient) DeleteAssignment(ctx context.Context, courseID, assignmentID int) error {
	return nil
}

func (m *mockClient) GetAssignmentGroupsForCourse(ctx context.Context, courseID int) ([]models.AssignmentGroup, error) {
	return nil, nil
}

func (m *mockClient) CreateAssignmentGroup(ctx context.Context, courseID int, name string) (*models.AssignmentGroup, error) {
	return &models.AssignmentGroup{ID: 1, Name: name}, nil
}

func (m *mockClient) GetSubmissionsForAssignment(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	return nil, nil
}

func (m *mockClient) GetSubmissionsForStudent(ctx context.Context, courseID, studentID int) ([]models.Submission, error) {
	return nil, nil
}

func (m *mockClient) GetAllSubmissionsForCourse(ctx context.Context, courseID int) ([]models.Submission, error) {
	return nil, nil
}

func (m *mockClient) GetAssignmentSubmissionsWithAttachments(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	return nil, nil
}

func (m *mockClient) GetStudentsForCourse(ctx context.Context, courseID int) ([]models.User, error) {
	return m.students, nil
}

func (m *mockClient) GetEnrollmentsForCourse(ctx context.Context, courseID int) ([]models.Enrollment, error) {
	return nil, nil
}

func (m *mockClient) GetQuizzesForCourse(ctx context.Context, courseID int) ([]models.Quiz, error) {
	return nil, nil
}

func (m *mockClient) GetQuizQuestions(ctx context.Context, courseID, quizID int) ([]models.QuizQuestion, error) {
	return nil, nil
}

func (m *mockClient) GetQuizSubmissions(ctx context.Context, courseID, quizID int) (*models.QuizSubmissionList, error) {
	return nil, nil
}

func (m *mockClient) DownloadFile(ctx context.Context, fileURL, localPath string) error {
	return nil
}

func (m *mockClient) UploadFileToCourse(ctx context.Context, filePath string, courseID, parentFolderID int) (*models.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fileID := len(m.uploaded) + 200
	m.uploaded[filePath] = fileID
	return &models.FileInfo{
		ID:          fileID,
		Name:        filePath,
		URL:         fmt.Sprintf("https://canvas.test/files/%d", fileID),
		DownloadURL: fmt.Sprintf("https://canvas.test/files/%d/download", fileID),
		PublicURL:   "",
	}, nil
}

func (m *mockClient) GetFileDownloadURL(ctx context.Context, fileID int) (string, error) {
	return fmt.Sprintf("https://canvas.test/files/%d/download?verifier=test", fileID), nil
}

func (m *mockClient) GetAnnouncements(ctx context.Context, courseID int) ([]models.Announcement, error) {
	return nil, nil
}

func (m *mockClient) CreateAnnouncement(ctx context.Context, courseID int, title, message string) (*models.Announcement, error) {
	return &models.Announcement{ID: 1, Title: title, Message: message}, nil
}

func (m *mockClient) UpdateAnnouncement(ctx context.Context, courseID, topicID int, title, message string) (*models.Announcement, error) {
	return &models.Announcement{ID: topicID, Title: title, Message: message}, nil
}

func (m *mockClient) DeleteAnnouncement(ctx context.Context, courseID, topicID int) error {
	return nil
}

func (m *mockClient) SubmitGrade(ctx context.Context, courseID, assignmentID, studentID int, grade string, comment string) error {
	return nil
}

func (m *mockClient) BatchSubmitGrades(ctx context.Context, courseID, assignmentID int, entries []models.GradeEntry) (string, error) {
	m.mu.Lock()
	m.gradeEntries = entries
	m.mu.Unlock()
	return m.progressURL, nil
}

func (m *mockClient) PollBatchProgress(ctx context.Context, progressURL string) (<-chan models.BatchProgress, error) {
	ch := make(chan models.BatchProgress, 2)
	go func() {
		defer close(ch)
		if m.failProgress {
			ch <- models.BatchProgress{WorkflowState: "failed", Message: "Canvas rejected batch"}
			return
		}
		ch <- models.BatchProgress{WorkflowState: "running", Completion: 50}
		ch <- models.BatchProgress{WorkflowState: "completed", Completion: 100}
	}()
	return ch, nil
}

func (m *mockClient) QueryProgress(ctx context.Context, progressID int) (*models.BatchProgress, error) {
	return &models.BatchProgress{WorkflowState: "completed", Completion: 100}, nil
}

// --- Test helpers ---

type spyEmitter struct {
	events []string
	mu     sync.Mutex
}

func (s *spyEmitter) Emit(event string, data map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *spyEmitter) contains(t *testing.T, event string) bool {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.events, event)
}

// --- Tests ---

func TestUploadGrades_Success(t *testing.T) {
	client := newMockClient()
	emitter := &spyEmitter{}
	uploader := grades.NewUploader(client, emitter)

	result := &grades.LoadResult{
		Students: []grades.StudentGrade{
			{StudentID: "1001", Grade: 95.0, Comment: "Great work"},
			{StudentID: "1002", Grade: 80.0},
		},
	}

	err := uploader.UploadGrades(context.Background(), 1, 101, "Homework 1", result)
	require.NoError(t, err)

	// Check grade entries were submitted
	client.mu.Lock()
	defer client.mu.Unlock()
	assert.Len(t, client.gradeEntries, 2)
	assert.Equal(t, 1001, client.gradeEntries[0].StudentID)
	assert.Equal(t, "95", client.gradeEntries[0].Grade)
	assert.Equal(t, 1002, client.gradeEntries[1].StudentID)
	assert.Equal(t, "80", client.gradeEntries[1].Grade)

	// Check events
	assert.True(t, emitter.contains(t, "upload:done"))
	assert.True(t, emitter.contains(t, "upload:batch_progress"))
}

func TestUploadGrades_StudentNotFound(t *testing.T) {
	client := newMockClient()
	emitter := &spyEmitter{}
	uploader := grades.NewUploader(client, emitter)

	result := &grades.LoadResult{
		Students: []grades.StudentGrade{
			{StudentID: "9999", Grade: 90.0}, // not in roster
		},
	}

	err := uploader.UploadGrades(context.Background(), 1, 101, "HW1", result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in course roster")
}

func TestUploadGrades_EmptyResult(t *testing.T) {
	client := newMockClient()
	uploader := grades.NewUploader(client, nil)

	err := uploader.UploadGrades(context.Background(), 1, 101, "HW1", &grades.LoadResult{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no students")
}

func TestUploadGrades_NilResult(t *testing.T) {
	client := newMockClient()
	uploader := grades.NewUploader(client, nil)

	err := uploader.UploadGrades(context.Background(), 1, 101, "HW1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no students")
}

func TestUploadGrades_FileUpload(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	pdfPath := dir + "/eval.pdf"
	writeFile(t, dir, "eval.pdf", "pdf content")

	client := newMockClient()
	emitter := &spyEmitter{}
	uploader := grades.NewUploader(client, emitter)

	result := &grades.LoadResult{
		Students: []grades.StudentGrade{
			{StudentID: "1001", Grade: 90.0, PDFEvalFile: pdfPath},
		},
	}

	err := uploader.UploadGrades(context.Background(), 1, 101, "HW1", result)
	require.NoError(t, err)

	// File should have been uploaded
	client.mu.Lock()
	assert.Len(t, client.uploaded, 1)
	assert.True(t, emitter.contains(t, "upload:file_progress"))
	client.mu.Unlock()
}

func TestUploadGrades_ProgressFailure(t *testing.T) {
	client := newMockClient()
	client.failProgress = true

	emitter := &spyEmitter{}
	uploader := grades.NewUploader(client, emitter)

	result := &grades.LoadResult{
		Students: []grades.StudentGrade{
			{StudentID: "1001", Grade: 90.0},
		},
	}

	err := uploader.UploadGrades(context.Background(), 1, 101, "HW1", result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Canvas rejected batch")
}

func TestGenerateFeedbackFolderName(t *testing.T) {
	// We can test this indirectly through the uploader
	client := newMockClient()
	uploader := grades.NewUploader(client, nil)

	result := &grades.LoadResult{
		Students: []grades.StudentGrade{
			{StudentID: "1001", Grade: 90.0},
		},
	}

	err := uploader.UploadGrades(context.Background(), 1, 101, "Homework 1", result)
	require.NoError(t, err)

	client.mu.Lock()
	defer client.mu.Unlock()
	// Verify folder was created with expected prefix
	found := false
	for path := range client.folders {
		if len(path) > 0 {
			found = true
			assert.Contains(t, path, "Grade_Feedback/")
			assert.Contains(t, path, "Homework_1")
		}
	}
	assert.True(t, found, "expected feedback folder to be created")
}
