package cache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"canvaslms-gui/internal/cache"
	"canvaslms-gui/internal/models"
)

// --- mockStore ---

type mockStore struct {
	mu   sync.RWMutex
	data map[string]mockEntry
}

type mockEntry struct {
	raw       []byte
	expiresAt int64
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]mockEntry)}
}

func (m *mockStore) Get(key string, dest any) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.data[key]
	if !ok {
		return false, nil
	}
	if time.Now().Unix() > e.expiresAt {
		return false, nil
	}
	return true, json.Unmarshal(e.raw, dest)
}

func (m *mockStore) Set(key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = mockEntry{raw: raw, expiresAt: time.Now().Add(ttl).Unix()}
	return nil
}

func (m *mockStore) Delete(pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.data {
		if matchPattern(k, pattern) {
			delete(m.data, k)
		}
	}
	return nil
}

func matchPattern(key, pattern string) bool {
	if key == pattern {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '%' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return false
}

func (m *mockStore) Purge() error {
	now := time.Now().Unix()
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, e := range m.data {
		if now > e.expiresAt {
			delete(m.data, k)
		}
	}
	return nil
}

func (m *mockStore) Close() error {
	return nil
}

// --- mockClient ---

type mockClient struct {
	mu          sync.Mutex
	courses     []models.Course
	assignments []models.Assignment
	students    []models.User
	submissions []models.Submission
	quizzes     []models.Quiz
	questions   []models.QuizQuestion
	quizSubs    *models.QuizSubmissionList
	folders     []models.Folder
	fileInfo    *models.FileInfo
	groups      []models.AssignmentGroup

	calls   map[string]int
	errs    map[string]error
	created *models.Assignment
}

func newMockClient() *mockClient {
	return &mockClient{
		courses:     []models.Course{{ID: 1, Name: "CS101"}},
		assignments: []models.Assignment{{ID: 10, Name: "HW1"}},
		students:    []models.User{{ID: 100, Name: "Alice"}},
		submissions: []models.Submission{{UserID: 100, AssignmentID: 10}},
		quizzes:     []models.Quiz{{ID: 20, Title: "Quiz1"}},
		questions:   []models.QuizQuestion{{ID: 200, QuestionName: "Q1"}},
		quizSubs:    &models.QuizSubmissionList{QuizSubmissions: []models.QuizSubmission{{ID: 300}}},
		folders:     []models.Folder{{ID: 50, Name: "files"}},
		fileInfo:    &models.FileInfo{ID: 60, Name: "upload.pdf"},
		calls:       make(map[string]int),
		errs:        make(map[string]error),
		created:     nil,
	}
}

func (m *mockClient) record(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[method]++
}

func (m *mockClient) count(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[method]
}

func (m *mockClient) setErr(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errs[method] = err
}

func (m *mockClient) getErr(method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errs[method]
}

func (m *mockClient) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	m.record("GetAllCourses")
	if err := m.getErr("GetAllCourses"); err != nil {
		return nil, err
	}
	return m.courses, nil
}

func (m *mockClient) GetFoldersForCourse(ctx context.Context, courseID int) ([]models.Folder, error) {
	m.record("GetFoldersForCourse")
	return m.folders, nil
}

func (m *mockClient) EnsureCourseFolder(ctx context.Context, courseID int, folderPath string) (*models.Folder, error) {
	m.record("EnsureCourseFolder")
	return &models.Folder{ID: 99, Name: folderPath}, nil
}

func (m *mockClient) GetAssignmentsForCourse(ctx context.Context, courseID int) ([]models.Assignment, error) {
	m.record("GetAssignmentsForCourse")
	if err := m.getErr("GetAssignmentsForCourse"); err != nil {
		return nil, err
	}
	return m.assignments, nil
}

func (m *mockClient) CreateAssignment(ctx context.Context, courseID int, data map[string]any) (*models.Assignment, error) {
	m.record("CreateAssignment")
	a := &models.Assignment{ID: 99, Name: "New"}
	m.created = a
	return a, nil
}

func (m *mockClient) EditAssignment(ctx context.Context, courseID, assignmentID int, data map[string]any) (*models.Assignment, error) {
	m.record("EditAssignment")
	return &models.Assignment{ID: assignmentID, Name: "Edited"}, nil
}

func (m *mockClient) DeleteAssignment(ctx context.Context, courseID, assignmentID int) error {
	m.record("DeleteAssignment")
	return nil
}

func (m *mockClient) GetAssignmentGroupsForCourse(ctx context.Context, courseID int) ([]models.AssignmentGroup, error) {
	m.record("GetAssignmentGroupsForCourse")
	return m.groups, nil
}

func (m *mockClient) CreateAssignmentGroup(ctx context.Context, courseID int, name string) (*models.AssignmentGroup, error) {
	m.record("CreateAssignmentGroup")
	m.groups = append(m.groups, models.AssignmentGroup{ID: len(m.groups) + 1, Name: name})
	return &m.groups[len(m.groups)-1], nil
}

func (m *mockClient) GetSubmissionsForAssignment(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	m.record("GetSubmissionsForAssignment")
	return m.submissions, nil
}

func (m *mockClient) GetAssignmentSubmissionsWithAttachments(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	m.record("GetAssignmentSubmissionsWithAttachments")
	return m.submissions, nil
}

func (m *mockClient) GetSubmissionsForStudent(ctx context.Context, courseID, studentID int) ([]models.Submission, error) {
	m.record("GetSubmissionsForStudent")
	return m.submissions, nil
}

func (m *mockClient) GetAllSubmissionsForCourse(ctx context.Context, courseID int) ([]models.Submission, error) {
	m.record("GetAllSubmissionsForCourse")
	return m.submissions, nil
}

func (m *mockClient) GetStudentsForCourse(ctx context.Context, courseID int) ([]models.User, error) {
	m.record("GetStudentsForCourse")
	return m.students, nil
}

func (m *mockClient) GetEnrollmentsForCourse(ctx context.Context, courseID int) ([]models.Enrollment, error) {
	m.record("GetEnrollmentsForCourse")
	return nil, nil
}

func (m *mockClient) GetQuizzesForCourse(ctx context.Context, courseID int) ([]models.Quiz, error) {
	m.record("GetQuizzesForCourse")
	return m.quizzes, nil
}

func (m *mockClient) GetQuizQuestions(ctx context.Context, courseID, quizID int) ([]models.QuizQuestion, error) {
	m.record("GetQuizQuestions")
	return m.questions, nil
}

func (m *mockClient) GetQuizSubmissions(ctx context.Context, courseID, quizID int) (*models.QuizSubmissionList, error) {
	m.record("GetQuizSubmissions")
	return m.quizSubs, nil
}

func (m *mockClient) DownloadFile(ctx context.Context, fileURL, localPath string) error {
	m.record("DownloadFile")
	return nil
}

func (m *mockClient) UploadFileToCourse(ctx context.Context, filePath string, courseID, parentFolderID int) (*models.FileInfo, error) {
	m.record("UploadFileToCourse")
	return m.fileInfo, nil
}

func (m *mockClient) GetFileDownloadURL(ctx context.Context, fileID int) (string, error) {
	m.record("GetFileDownloadURL")
	return fmt.Sprintf("https://canvas.test/files/%d/download?verifier=test", fileID), nil
}

func (m *mockClient) GetAnnouncements(ctx context.Context, courseID int) ([]models.Announcement, error) {
	m.record("GetAnnouncements")
	return nil, nil
}

func (m *mockClient) CreateAnnouncement(ctx context.Context, courseID int, title, message string) (*models.Announcement, error) {
	m.record("CreateAnnouncement")
	return &models.Announcement{ID: 1, Title: title, Message: message}, nil
}

func (m *mockClient) UpdateAnnouncement(ctx context.Context, courseID, topicID int, title, message string) (*models.Announcement, error) {
	m.record("UpdateAnnouncement")
	return &models.Announcement{ID: topicID, Title: title, Message: message}, nil
}

func (m *mockClient) DeleteAnnouncement(ctx context.Context, courseID, topicID int) error {
	m.record("DeleteAnnouncement")
	return nil
}

func (m *mockClient) SubmitGrade(ctx context.Context, courseID, assignmentID, studentID int, grade string, comment string) error {
	m.record("SubmitGrade")
	return nil
}

func (m *mockClient) BatchSubmitGrades(ctx context.Context, courseID, assignmentID int, entries []models.GradeEntry) (string, error) {
	m.record("BatchSubmitGrades")
	return "https://canvas.test/progress/1", nil
}

func (m *mockClient) PollBatchProgress(ctx context.Context, progressURL string) (<-chan models.BatchProgress, error) {
	m.record("PollBatchProgress")
	ch := make(chan models.BatchProgress, 1)
	close(ch)
	return ch, nil
}

func (m *mockClient) QueryProgress(ctx context.Context, progressID int) (*models.BatchProgress, error) {
	m.record("QueryProgress")
	return &models.BatchProgress{WorkflowState: "completed", Completion: 100}, nil
}

// --- Tests ---

func TestCachedClient_GetAllCourses_CacheHit(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	// First call hits real client
	got1, err := client.GetAllCourses(context.Background())
	require.NoError(t, err)
	assert.Len(t, got1, 1)
	assert.Equal(t, 1, real.count("GetAllCourses"))

	// Second call hits cache
	got2, err := client.GetAllCourses(context.Background())
	require.NoError(t, err)
	assert.Equal(t, got1, got2)
	assert.Equal(t, 1, real.count("GetAllCourses")) // no additional call
}

func TestCachedClient_GetAssignments_CacheMissThenHit(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	got1, err := client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, got1, 1)
	assert.Equal(t, 1, real.count("GetAssignmentsForCourse"))

	got2, err := client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, got1, got2)
	assert.Equal(t, 1, real.count("GetAssignmentsForCourse"))
}

func TestCachedClient_ErrorNotCached(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	real.setErr("GetAllCourses", fmt.Errorf("network down"))

	_, err := client.GetAllCourses(context.Background())
	require.Error(t, err)

	// After error, subsequent call should still hit real client
	real.setErr("GetAllCourses", nil)
	got, err := client.GetAllCourses(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 2, real.count("GetAllCourses"))
}

func TestCachedClient_CreateAssignment_InvalidatesCache(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	// Prime cache
	_, err := client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)

	// Mutation
	_, err = client.CreateAssignment(context.Background(), 1, map[string]any{"name": "New"})
	require.NoError(t, err)

	// Cache should be invalidated, so next read hits real client again
	_, err = client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetAssignmentsForCourse"))
}

func TestCachedClient_DeleteAssignment_InvalidatesCache(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	_, err := client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)

	err = client.DeleteAssignment(context.Background(), 1, 10)
	require.NoError(t, err)

	_, err = client.GetAssignmentsForCourse(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetAssignmentsForCourse"))
}

func TestCachedClient_SubmitGrade_InvalidatesSubmissions(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	// Prime multiple submission caches
	_, err := client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	require.NoError(t, err)
	_, err = client.GetAssignmentSubmissionsWithAttachments(context.Background(), 1, 10)
	require.NoError(t, err)
	_, err = client.GetSubmissionsForStudent(context.Background(), 1, 100)
	require.NoError(t, err)

	// Grade submission
	err = client.SubmitGrade(context.Background(), 1, 10, 100, "95", "Good")
	require.NoError(t, err)

	// All should be invalidated
	_, err = client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetSubmissionsForAssignment"))

	_, err = client.GetAssignmentSubmissionsWithAttachments(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetAssignmentSubmissionsWithAttachments"))

	_, err = client.GetSubmissionsForStudent(context.Background(), 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetSubmissionsForStudent"))
}

func TestCachedClient_BatchSubmitGrades_InvalidatesCache(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	_, err := client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	require.NoError(t, err)

	_, err = client.BatchSubmitGrades(context.Background(), 1, 10, nil)
	require.NoError(t, err)

	_, err = client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, real.count("GetSubmissionsForAssignment"))
}

func TestCachedClient_Mutation_AlwaysForwarded(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	_, err := client.CreateAssignment(context.Background(), 1, map[string]any{"name": "New"})
	require.NoError(t, err)
	assert.Equal(t, 1, real.count("CreateAssignment"))

	err = client.DeleteAssignment(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, real.count("DeleteAssignment"))

	err = client.SubmitGrade(context.Background(), 1, 10, 100, "95", "")
	require.NoError(t, err)
	assert.Equal(t, 1, real.count("SubmitGrade"))
}

func TestCachedClient_EnsureCourseFolder_Passthrough(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	f, err := client.EnsureCourseFolder(context.Background(), 1, "folder")
	require.NoError(t, err)
	assert.Equal(t, "folder", f.Name)
	assert.Equal(t, 1, real.count("EnsureCourseFolder"))
}

func TestCachedClient_DownloadFile_Passthrough(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	err := client.DownloadFile(context.Background(), "url", "path")
	require.NoError(t, err)
	assert.Equal(t, 1, real.count("DownloadFile"))
}

func TestCachedClient_UploadFile_Passthrough(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	fi, err := client.UploadFileToCourse(context.Background(), "file", 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 60, fi.ID)
	assert.Equal(t, 1, real.count("UploadFileToCourse"))
}

func TestCachedClient_PollBatchProgress_Passthrough(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	ch, err := client.PollBatchProgress(context.Background(), "url")
	require.NoError(t, err)
	// Drain channel
	for range ch {
	}
	assert.Equal(t, 1, real.count("PollBatchProgress"))
}

func TestCachedClient_QueryProgress_Passthrough(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	p, err := client.QueryProgress(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, "completed", p.WorkflowState)
	assert.Equal(t, 1, real.count("QueryProgress"))
}

func TestCachedClient_CachesAllReadMethods(t *testing.T) {
	real := newMockClient()
	store := newMockStore()
	client := cache.NewCachedCanvasClient(real, store)

	// Call every read method twice; second call should be cached.
	_, _ = client.GetAllCourses(context.Background())
	_, _ = client.GetAllCourses(context.Background())
	assert.Equal(t, 1, real.count("GetAllCourses"))

	_, _ = client.GetFoldersForCourse(context.Background(), 1)
	_, _ = client.GetFoldersForCourse(context.Background(), 1)
	assert.Equal(t, 1, real.count("GetFoldersForCourse"))

	_, _ = client.GetAssignmentsForCourse(context.Background(), 1)
	_, _ = client.GetAssignmentsForCourse(context.Background(), 1)
	assert.Equal(t, 1, real.count("GetAssignmentsForCourse"))

	_, _ = client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	_, _ = client.GetSubmissionsForAssignment(context.Background(), 1, 10)
	assert.Equal(t, 1, real.count("GetSubmissionsForAssignment"))

	_, _ = client.GetAssignmentSubmissionsWithAttachments(context.Background(), 1, 10)
	_, _ = client.GetAssignmentSubmissionsWithAttachments(context.Background(), 1, 10)
	assert.Equal(t, 1, real.count("GetAssignmentSubmissionsWithAttachments"))

	_, _ = client.GetSubmissionsForStudent(context.Background(), 1, 100)
	_, _ = client.GetSubmissionsForStudent(context.Background(), 1, 100)
	assert.Equal(t, 1, real.count("GetSubmissionsForStudent"))

	_, _ = client.GetStudentsForCourse(context.Background(), 1)
	_, _ = client.GetStudentsForCourse(context.Background(), 1)
	assert.Equal(t, 1, real.count("GetStudentsForCourse"))

	_, _ = client.GetEnrollmentsForCourse(context.Background(), 1)
	_, _ = client.GetEnrollmentsForCourse(context.Background(), 1)
	assert.Equal(t, 1, real.count("GetEnrollmentsForCourse"))

	_, _ = client.GetQuizzesForCourse(context.Background(), 1)
	_, _ = client.GetQuizzesForCourse(context.Background(), 1)
	assert.Equal(t, 1, real.count("GetQuizzesForCourse"))

	_, _ = client.GetQuizQuestions(context.Background(), 1, 20)
	_, _ = client.GetQuizQuestions(context.Background(), 1, 20)
	assert.Equal(t, 1, real.count("GetQuizQuestions"))

	_, _ = client.GetQuizSubmissions(context.Background(), 1, 20)
	_, _ = client.GetQuizSubmissions(context.Background(), 1, 20)
	assert.Equal(t, 1, real.count("GetQuizSubmissions"))
}
