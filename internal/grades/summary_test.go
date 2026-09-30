package grades_test

import (
	"context"
	"encoding/json"
	"testing"

	"canvaslms-gui/internal/grades"
	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

func TestMatchCanvasIDs(t *testing.T) {
	rows := []grades.StudentGrade{{StudentID: "001001"}, {StudentID: "9999"}, {StudentID: "1002"}}
	matched, skipped := grades.MatchStudents(rows, []models.User{{ID: 1001}, {ID: 1002}})
	require.Len(t, matched, 2)
	require.Equal(t, "1001", matched[0].StudentID)
	require.Equal(t, []grades.StudentIssue{{StudentID: "9999", Row: 3, Reason: "not found in course roster"}}, skipped)
}

func TestIgnoreDoesNotUploadUnmatchedFiles(t *testing.T) {
	client := newMockClient()
	u := grades.NewUploader(client, nil)
	u.ConfirmUnmatched = func(_ context.Context, students []grades.StudentIssue) error {
		require.Len(t, students, 1)
		return nil
	}
	u.Prepare = func(result *grades.LoadResult) error {
		require.Len(t, result.Students, 1)
		require.Equal(t, "1001", result.Students[0].StudentID)
		return nil
	}
	result := &grades.LoadResult{Students: []grades.StudentGrade{
		{StudentID: "1001", Grade: 95},
		{StudentID: "9999", PDFEvalFile: "/missing/skipped.pdf"},
	}}
	require.NoError(t, u.UploadGrades(context.Background(), 1, 101, "", result))
	require.Len(t, client.gradeEntries, 1)
	require.Empty(t, client.uploaded)
}

func TestAllUnmatchedDoesNotWriteToCanvas(t *testing.T) {
	client := newMockClient()
	u := grades.NewUploader(client, nil)
	u.ConfirmUnmatched = func(context.Context, []grades.StudentIssue) error { return nil }
	require.NoError(t, u.UploadGrades(context.Background(), 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "9999"}}}))
	require.Empty(t, client.folders)
	require.Empty(t, client.gradeEntries)
}

func TestConfirmationCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, grades.ConfirmStudents(ctx, &spyEmitter{}, nil, nil), context.Canceled)
	decisions := make(chan string, 1)
	decisions <- "ignore"
	require.NoError(t, grades.ConfirmStudents(context.Background(), &spyEmitter{}, decisions, nil))
}

func TestCanvasJobIssues(t *testing.T) {
	for _, results := range []string{
		`{"errors":{"1001":"grade rejected"}}`,
		`{"failed_students":[{"user_id":1001,"message":"grade rejected"}]}`,
		`{"missing_user_ids":[1001]}`,
	} {
		issues := grades.JobIssues(models.BatchProgress{Results: json.RawMessage(results)})
		require.Len(t, issues, 1)
		require.Equal(t, "1001", issues[0].StudentID)
	}
	issues := grades.JobIssues(models.BatchProgress{Message: "Couldn't find User(s) with API ids '1002', '1003'"})
	require.Len(t, issues, 2)
	require.Empty(t, grades.JobIssues(models.BatchProgress{}))
	require.Len(t, grades.JobIssues(models.BatchProgress{Results: json.RawMessage(`{"unexpected":"details"}`)}), 1)
}
