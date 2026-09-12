package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
)

// scoreColumn describes one assignment/quiz column in the output CSV.
type scoreColumn struct {
	header       string // CSV column header (item name)
	assignmentID int    // Canvas assignment ID used to look up the submission score
}

// ExportScoresCSV writes a grades matrix to filePath as CSV.
//
// Columns: canvas_id, student_name, <item1>, <item2>, ...
//   - Assignments → column uses the assignment's own Canvas ID.
//   - Quizzes     → column uses the quiz's linked assignment_id (Canvas stores
//     quiz grades as assignment submissions). Quizzes without an assignment_id
//     (practice/ungraded) are skipped.
//
// Rows are sorted alphabetically by student name. Score cells contain the
// numeric score or an empty string when the student has not been graded.
func (e *canvasExporter) ExportScoresCSV(courseID int, filePath string) (string, error) {
	// ---- fetch all data ----
	students, err := e.client.GetStudentsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get students: %w", err)
	}

	assignments, err := e.client.GetAssignmentsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get assignments: %w", err)
	}

	quizzes, err := e.client.GetQuizzesForCourse(e.ctx, courseID)
	if err != nil {
		quizzes = nil // quizzes are optional; continue without them
	}

	allSubs, err := e.client.GetAllSubmissionsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get submissions: %w", err)
	}

	// ---- build column list ----
	// Assignments that are represented by a quiz are suppressed (same dedup
	// logic as ListCourseItems) so each item appears only once.
	quizAssignmentIDs := make(map[int]bool, len(quizzes))
	for _, q := range quizzes {
		if q.AssignmentID != nil && *q.AssignmentID > 0 {
			quizAssignmentIDs[*q.AssignmentID] = true
		}
	}

	var columns []scoreColumn
	for _, a := range assignments {
		if quizAssignmentIDs[a.ID] {
			continue // represented by a quiz column below
		}
		columns = append(columns, scoreColumn{header: a.Name, assignmentID: a.ID})
	}
	for _, q := range quizzes {
		if q.AssignmentID == nil || *q.AssignmentID == 0 {
			continue // practice / ungraded quiz — no score to report
		}
		name := q.Name
		if name == "" {
			name = q.Title
		}
		columns = append(columns, scoreColumn{header: name, assignmentID: *q.AssignmentID})
	}

	// ---- build score map: userID → assignmentID → score string ----
	scoreMap := make(map[int]map[int]string, len(students))
	for _, sub := range allSubs {
		if scoreMap[sub.UserID] == nil {
			scoreMap[sub.UserID] = make(map[int]string)
		}
		scoreMap[sub.UserID][sub.AssignmentID] = formatScore(sub.Score)
	}

	// ---- sort students alphabetically ----
	sort.Slice(students, func(i, j int) bool {
		return students[i].Name < students[j].Name
	})

	// ---- write CSV ----
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header row
	header := make([]string, 0, 2+len(columns))
	header = append(header, "canvas_id", "student_name")
	for _, col := range columns {
		header = append(header, col.header)
	}
	if err := w.Write(header); err != nil {
		return "", err
	}

	// One row per student
	row := make([]string, 2+len(columns))
	for _, s := range students {
		row[0] = strconv.Itoa(s.ID)
		row[1] = s.Name
		for i, col := range columns {
			row[2+i] = scoreMap[s.ID][col.assignmentID]
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}

	if err := w.Error(); err != nil {
		return "", err
	}
	return filePath, nil
}

// formatScore converts the Submission.Score (interface{}) to a plain string.
// Returns an empty string for nil / ungraded scores.
func formatScore(score interface{}) string {
	if score == nil {
		return ""
	}
	switch v := score.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		s := fmt.Sprintf("%v", v)
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

