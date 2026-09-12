package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/models"
)

// --- Mock Canvas Server ---

func newMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// Courses
	mux.HandleFunc("/api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "CS101", "course_code": "CS-101"},
			{"id": 2, "name": "MATH201", "course_code": "MATH-201"},
		})
	})

	// Assignments
	mux.HandleFunc("/api/v1/courses/1/assignments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 101, "name": "Homework 1", "points_possible": 100, "published": true},
		})
	})

	// Submissions for assignment
	mux.HandleFunc("/api/v1/courses/1/assignments/101/submissions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"user_id": 1001, "score": 85.0, "workflow_state": "submitted"},
		})
	})

	// Students
	mux.HandleFunc("/api/v1/courses/1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1001, "name": "Alice", "email": "alice@example.com"},
			{"id": 1002, "name": "Bob", "email": "bob@example.com"},
		})
	})

	// Enrollments
	mux.HandleFunc("/api/v1/courses/1/enrollments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"user_id": 1001, "grades": map[string]any{"current_score": 95.5, "final_score": 94.0}},
		})
	})

	// Quizzes (old API)
	mux.HandleFunc("/api/v1/courses/1/quizzes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 50, "title": "Quiz 1", "quiz_type": "assignment"},
		})
	})

	// Quizzes (new API)
	mux.HandleFunc("/api/quiz/v1/courses/1/quizzes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 51, "title": "New Quiz 1", "quiz_type": "new_quiz"},
		})
	})

	// Quiz questions (old API)
	mux.HandleFunc("/api/v1/courses/1/quizzes/50/questions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "question_name": "Q1", "question_type": "multiple_choice"},
		})
	})

	// Quiz submissions (old API)
	mux.HandleFunc("/api/v1/courses/1/quizzes/50/submissions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"quiz_submissions": []map[string]any{
				{"id": 1, "user_id": 1001, "score": 85, "workflow_state": "complete"},
			},
		})
	})

	// Single grade submit
	mux.HandleFunc("/api/v1/courses/1/assignments/101/submissions/1001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})

	// Batch grade submit
	mux.HandleFunc("/api/v1/courses/1/assignments/101/submissions/update_grades", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":  42,
			"url": "http://" + r.Host + "/api/v1/progress/42",
		})
	})

	// Progress polling
	mux.HandleFunc("/api/v1/progress/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":             42,
			"workflow_state": "completed",
			"completion":     100.0,
		})
	})

	// Folders
	mux.HandleFunc("/api/v1/courses/1/folders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 10, "name": "submissions", "full_name": "course files/submissions"},
		})
	})

	return httptest.NewServer(mux)
}

// --- Helper ---

func newTestClient(t *testing.T, serverURL string) api.CanvasClient {
	t.Helper()
	t.Setenv("CANVAS_API_TOKEN", "test-token")
	client, err := api.NewCanvasClient(serverURL)
	require.NoError(t, err)
	return client
}

// --- Tests: Construction ---

func TestNewCanvasClient_MissingToken(t *testing.T) {
	t.Setenv("CANVAS_API_TOKEN", "")
	_, err := api.NewCanvasClient("https://example.com")
	require.Error(t, err)
}

func TestNewCanvasClient_WithToken(t *testing.T) {
	t.Setenv("CANVAS_API_TOKEN", "test-token")
	client, err := api.NewCanvasClient("https://example.com")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewCanvasClientWithToken(t *testing.T) {
	client, err := api.NewCanvasClientWithToken("https://example.com", "my-token")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewCanvasClientWithToken_EmptyToken(t *testing.T) {
	_, err := api.NewCanvasClientWithToken("https://example.com", "")
	require.Error(t, err)
}

// --- Tests: Courses ---

func TestGetAllCourses_SinglePage(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	courses, err := client.GetAllCourses(context.Background())

	require.NoError(t, err)
	assert.Len(t, courses, 2)
	assert.Equal(t, "CS101", courses[0].Name)
	assert.Equal(t, "CS-101", courses[0].CourseCode)
	assert.Equal(t, "MATH201", courses[1].Name)
	assert.Equal(t, "MATH-201", courses[1].CourseCode)
}

func TestGetAllCourses_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `</api/v1/courses-page2>; rel="next"`)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "CS101"},
		})
	})
	mux.HandleFunc("/api/v1/courses-page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 2, "name": "MATH201"},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	courses, err := client.GetAllCourses(context.Background())

	require.NoError(t, err)
	assert.Len(t, courses, 2)
	assert.Equal(t, "CS101", courses[0].Name)
	assert.Equal(t, "MATH201", courses[1].Name)
}

// --- Tests: Assignments ---

func TestGetAssignmentsForCourse(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	assignments, err := client.GetAssignmentsForCourse(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, 101, assignments[0].ID)
	assert.Equal(t, "Homework 1", assignments[0].Name)
	assert.Equal(t, 100.0, assignments[0].PointsPossible)
}

// --- Tests: Students ---

func TestGetStudentsForCourse(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	students, err := client.GetStudentsForCourse(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, students, 2)
	assert.Equal(t, "Alice", students[0].Name)
	assert.Equal(t, "alice@example.com", students[0].Email)
}

// --- Tests: Enrollments ---

func TestGetEnrollmentsForCourse(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	enrollments, err := client.GetEnrollmentsForCourse(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, enrollments, 1)
	assert.Equal(t, 1001, enrollments[0].UserID)
}

// --- Tests: Quizzes ---

type byId []models.Quiz

func (a byId) Len() int           { return len(a) }
func (a byId) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byId) Less(i, j int) bool { return a[i].ID < a[j].ID }

func TestGetQuizzesForCourse(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	quizzes, err := client.GetQuizzesForCourse(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, quizzes, 2)

	sort.Sort(byId(quizzes))
	assert.Equal(t, quizzes[0].ID, 50)
	assert.Equal(t, quizzes[1].ID, 51)
}

func TestGetQuizQuestions(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	questions, err := client.GetQuizQuestions(context.Background(), 1, 50)

	require.NoError(t, err)
	assert.Len(t, questions, 1)
	assert.Equal(t, "Q1", questions[0].QuestionName)
}

func TestGetQuizSubmissions(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	result, err := client.GetQuizSubmissions(context.Background(), 1, 50)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.QuizSubmissions, 1)
	assert.Equal(t, 1001, result.QuizSubmissions[0].UserID)
}

// --- Tests: Submissions ---

func TestGetSubmissionsForAssignment(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	submissions, err := client.GetSubmissionsForAssignment(context.Background(), 1, 101)

	require.NoError(t, err)
	assert.NotNil(t, submissions)
}

func TestGetAssignmentSubmissionsWithAttachments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/courses/1/assignments/101/submissions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"user_id": 1001,
				"score":   90.0,
				"attachments": []map[string]any{
					{"id": 200, "display_name": "hw1.pdf", "filename": "hw1.pdf", "url": "/files/200", "size": 1024, "content-type": "application/pdf"},
				},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	submissions, err := client.GetAssignmentSubmissionsWithAttachments(context.Background(), 1, 101)

	require.NoError(t, err)
	assert.Len(t, submissions, 1)
	assert.Equal(t, 1001, submissions[0].UserID)
	assert.Len(t, submissions[0].Attachments, 1)
	assert.Equal(t, 200, submissions[0].Attachments[0].ID)
}

// --- Tests: Folders ---

func TestGetFoldersForCourse(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	folders, err := client.GetFoldersForCourse(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, folders, 1)
	assert.Equal(t, 10, folders[0].ID)
	assert.Equal(t, "submissions", folders[0].Name)
}

// --- Tests: Grades ---

func TestSubmitGrade(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	err := client.SubmitGrade(context.Background(), 1, 101, 1001, "95.0", "Good work!")
	require.NoError(t, err)
}

func TestBatchSubmitGrades(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	entries := []models.GradeEntry{
		{StudentID: 1001, Grade: "95.0", Comment: "Excellent"},
		{StudentID: 1002, Grade: "80.0", Comment: "Good"},
	}

	progressURL, err := client.BatchSubmitGrades(context.Background(), 1, 101, entries)

	require.NoError(t, err)
	assert.Contains(t, progressURL, "/progress/42")
}

func TestPollBatchProgress(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/progress/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		state := "running"
		completion := 50.0
		if callCount >= 2 {
			state = "completed"
			completion = 100.0
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":             42,
			"workflow_state": state,
			"completion":     completion,
		})
	})

	mux.HandleFunc("/api/v1/courses/1/assignments/101/submissions/update_grades", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":  42,
			"url": "http://" + r.Host + "/api/v1/progress/42",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	entries := []models.GradeEntry{
		{StudentID: 1001, Grade: "95.0"},
	}

	progressURL, err := client.BatchSubmitGrades(context.Background(), 1, 101, entries)
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := client.PollBatchProgress(ctx, progressURL)
	require.NoError(t, err)

	var states []string
	for p := range ch {
		states = append(states, p.WorkflowState)
	}

	assert.Contains(t, states, "running")
	assert.Equal(t, "completed", states[len(states)-1])
}

func TestQueryProgress(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	progress, err := client.QueryProgress(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, 42, progress.ID)
	assert.Equal(t, "completed", progress.WorkflowState)
	assert.Equal(t, 100.0, progress.Completion)
}

// --- Tests: Error Handling ---

func TestAuthError_401(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "Invalid access token"}},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	_, err := client.GetAllCourses(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth")
}

func TestAPIError_404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "The specified resource does not exist"}},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	_, err := client.GetAllCourses(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestAPIError_500_NoJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	_, err := client.GetAllCourses(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// --- Tests: Context Cancellation ---

func TestContextCancellation(t *testing.T) {
	ts := newMockServer()
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetAllCourses(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
