package grades_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"canvaslms-gui/internal/grades"
	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

func TestFilePreflightCollectsAllProblems(t *testing.T) {
	dir := t.TempDir()
	result := &grades.LoadResult{Students: []grades.StudentGrade{
		{StudentID: "1001", PDFExamFile1: filepath.Join(dir, "missing.pdf"), PDFEvalFile: dir},
		{StudentID: "1002", PDFExamFile2: filepath.Join(dir, "also-missing.pdf")},
	}}
	err := grades.NewLoader("", "").CheckFiles(result)
	var problems *grades.FileProblems
	require.ErrorAs(t, err, &problems)
	require.Len(t, problems.Issues, 3)
	require.Equal(t, "not found", problems.Issues[0].Reason)
	require.Contains(t, problems.Issues[1].Reason, "not a regular file")
	client := newMockClient()
	require.Error(t, grades.NewUploader(client, nil).UploadGrades(context.Background(), 1, 101, "", result))
	require.Empty(t, client.folders)
	require.Empty(t, client.uploaded)
	require.Empty(t, client.gradeEntries)
}

func TestFilePermissionPreflight(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("POSIX non-root permissions required")
	}
	path := writeFile(t, t.TempDir(), "denied.pdf", "content")
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { _ = os.Chmod(path, 0600) })
	err := grades.NewLoader("", "").CheckFiles(&grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "1001", PDFEvalFile: path}}})
	var problems *grades.FileProblems
	require.ErrorAs(t, err, &problems)
	require.Equal(t, "permission denied", problems.Issues[0].Reason)
}

type flakyFileClient struct {
	*mockClient
	lock  sync.Mutex
	calls map[string]int
	bad   string
}

func (c *flakyFileClient) UploadFileToCourse(ctx context.Context, path string, courseID, folderID int) (*models.FileInfo, error) {
	c.lock.Lock()
	c.calls[path]++
	bad := c.bad == path
	c.lock.Unlock()
	if bad {
		return nil, fmt.Errorf("simulated upload failure")
	}
	return c.mockClient.UploadFileToCourse(ctx, path, courseID, folderID)
}

func TestWholeRetryReusesConfirmedUploadsAndNotChangedFiles(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "good.pdf", "good")
	bad := writeFile(t, dir, "bad.pdf", "bad")
	client := &flakyFileClient{mockClient: newMockClient(), calls: make(map[string]int), bad: bad}
	cache := grades.NewUploadCache()
	run := func() error {
		u := grades.NewUploader(client, nil)
		u.Cache = cache
		return u.UploadGrades(context.Background(), 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "1001", PDFExamFile1: good, PDFEvalFile: bad}}})
	}
	var problems *grades.FileProblems
	require.ErrorAs(t, run(), &problems)
	require.Len(t, problems.Issues, 1)
	require.Equal(t, "1001", problems.Issues[0].StudentID)
	require.Equal(t, bad, problems.Issues[0].File)
	require.Empty(t, client.gradeEntries)
	client.bad = ""
	require.NoError(t, run())
	require.Equal(t, 1, client.calls[good], "successful files must not be reuploaded")
	require.Equal(t, 2, client.calls[bad])
	require.NoError(t, os.WriteFile(good, []byte("changed"), 0600))
	require.NoError(t, run())
	require.Equal(t, 2, client.calls[good], "changed content must be uploaded again")
	require.Equal(t, 2, client.calls[bad])
}

func TestSharedFileIsUploadedOnlyOnce(t *testing.T) {
	path := writeFile(t, t.TempDir(), "shared.pdf", "shared feedback")
	client := &flakyFileClient{mockClient: newMockClient(), calls: make(map[string]int)}
	u := grades.NewUploader(client, nil)
	err := u.UploadGrades(context.Background(), 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{
		{StudentID: "1001", PDFExamFile1: path, PDFEvalFile: path},
		{StudentID: "1002", PDFEvalFile: path},
	}})
	require.NoError(t, err)
	require.Equal(t, 1, client.calls[path])
	require.Len(t, client.gradeEntries, 2)
}

func TestCSVRepairAndPathNormalization(t *testing.T) {
	path := writeCSV(t, "student_id,grade\n1001,bad\n")
	loader := grades.NewLoader(" \""+path+"\" ", "")
	_, err := loader.Parse()
	require.ErrorContains(t, err, "invalid grade")
	require.NoError(t, os.WriteFile(path, []byte("\ufeffcanvas_id,grade\n1001,95\n"), 0600))
	result, err := loader.Parse()
	require.NoError(t, err)
	require.Len(t, result.Students, 1)
	for _, content := range []string{"student_id,grade\n1001,NaN\n", "student_id,grade\n1001,5\n1001,6\n", "student_id,canvas_id,grade\n1001,1001,5\n", "student_id,grade\n1001,\xff\n", "student_id,grade\n1001,\"unterminated\n"} {
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		_, err := loader.Parse()
		require.Error(t, err)
	}
}
