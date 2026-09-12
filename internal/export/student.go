package export

import (
	"encoding/csv"
	"fmt"
	"os"
)

// ExportStudentsCSV writes the student roster as CSV to the specified path.
// Returns the path of the saved file, or an empty string if the user cancelled.
func (e *canvasExporter) ExportStudentsCSV(courseID int, path string) (string, error) {
	students, err := e.client.GetStudentsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get students: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	w.Write([]string{"canvas_id", "name", "email", "sis_user_id"})

	for _, s := range students {
		w.Write([]string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			s.Email,
			s.SISUserID,
		})
	}

	return path, w.Error()
}
