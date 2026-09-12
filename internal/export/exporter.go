package export

import (
	"canvaslms-gui/internal/api"
	"context"
	"strings"
)

type Exporter interface {
	ExportStudentsCSV(courseID int, filePath string) (string, error)
	ExportQuizQuestions(courseID, quizID int, format string, filePath string) (string, error)
	ExportQuizSubmissions(courseID, quizID int, format string, dirPath string, progress ProgressFunc) (string, error)
	ExportAssignmentSubmissions(courseID, assignmentID int, dirPath string, progress ProgressFunc) (string, error)
	ExportScoresCSV(courseID int, filePath string) (string, error)
}

type canvasExporter struct {
	ctx    context.Context
	client api.CanvasClient
}

func NewCanvasExporter(ctx context.Context, client api.CanvasClient) Exporter {
	return &canvasExporter{
		ctx:    ctx,
		client: client,
	}
}

func SanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	result := strings.ReplaceAll(strings.TrimSpace(b.String()), " ", "_")
	if len(result) > 30 {
		result = result[:30]
	}
	return result
}
