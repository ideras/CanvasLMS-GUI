package grades

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	apperrors "canvaslms-gui/internal/errors"
)

// MarkdownConverter converts Markdown files to PDF.
type MarkdownConverter interface {
	ConvertFile(inputPath, outputPath string) error
}

// StudentGrade holds one validated row from the grades CSV.
type StudentGrade struct {
	StudentID    string
	Grade        float64
	Comment      string
	MDExamFile1  string // resolved path to Markdown exam file 1
	MDExamFile2  string // resolved path to Markdown exam file 2
	MDEvalFile   string // resolved path to Markdown evaluation file
	PDFExamFile1 string // resolved path to PDF exam file 1
	PDFExamFile2 string // resolved path to PDF exam file 2
	PDFEvalFile  string // resolved path to PDF evaluation file
}

// LoadResult holds the parsed and validated grades data.
type LoadResult struct {
	Students []StudentGrade
	HasMD    bool // true if any Markdown columns were found
}

// mdColumns lists the Markdown file column names.
var mdColumns = []string{"md_exam_file1", "md_exam_file2", "md_eval_file"}

// pdfColumns lists the PDF file column names.
var pdfColumns = []string{"pdf_exam_file1", "pdf_exam_file2", "pdf_eval_file"}

// validColumns lists all recognized CSV columns.
var validColumns = map[string]bool{
	"canvas_id":      true,
	"student_id":     true,
	"grade":          true,
	"comment":        true,
	"comments":       true,
	"md_exam_file1":  true,
	"pdf_exam_file1": true,
	"md_exam_file2":  true,
	"pdf_exam_file2": true,
	"md_eval_file":   true,
	"pdf_eval_file":  true,
}

// columnRenames maps input column names to canonical names.
var columnRenames = map[string]string{
	"canvas_id": "student_id",
	"comments":  "comment",
}

// Loader parses and validates a Canvas grades CSV file.
type Loader struct {
	csvPath string
	rootDir string
}

// NewLoader creates a new Loader. rootDir is the base directory for resolving
// relative file paths; if empty, the CSV file's directory is used.
func NewLoader(csvPath, rootDir string) *Loader {
	csvPath = NormalizeCSVPath(csvPath)
	if rootDir == "" {
		rootDir = filepath.Dir(csvPath)
	}
	return &Loader{csvPath: csvPath, rootDir: rootDir}
}

// Load reads and validates both the CSV and its referenced files.
func (l *Loader) Load() (*LoadResult, error) {
	result, err := l.Parse()
	if err != nil {
		return nil, err
	}
	if err := l.CheckFiles(result); err != nil {
		return nil, err
	}
	return result, nil
}

// Parse validates CSV data without touching attachments. Roster matching must
// happen before file validation/conversion so ignored students have no side effects.
func (l *Loader) Parse() (*LoadResult, error) {
	f, err := os.Open(l.csvPath)
	if err != nil {
		return nil, fmt.Errorf("open CSV: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("CSV path is not a regular file: %s", l.csvPath)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if !utf8.Valid(data) {
		return nil, &apperrors.ValidationError{Field: "csv", Message: "bad CSV encoding: save as UTF-8"}
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, &apperrors.ValidationError{Field: "csv", Message: fmt.Sprintf("failed to read CSV: %v", err)}
	}

	if len(records) == 0 {
		return nil, &apperrors.ValidationError{Field: "csv", Message: "CSV file is empty"}
	}

	// Build column index from header row
	header := records[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		name := strings.TrimSpace(h)
		if canonical, ok := columnRenames[name]; ok {
			name = canonical
		}
		if _, duplicate := colIndex[name]; duplicate {
			return nil, &apperrors.ValidationError{Field: "csv", Message: "duplicate column: " + name}
		}
		colIndex[name] = i
	}

	// Apply renames to the column index
	for old, newName := range columnRenames {
		if idx, ok := colIndex[old]; ok {
			colIndex[newName] = idx
			delete(colIndex, old)
		}
	}

	// Validate all columns are known
	for name := range colIndex {
		if !validColumns[name] {
			return nil, &apperrors.ValidationError{
				Field:   "csv",
				Message: fmt.Sprintf("invalid column in CSV file: %s", name),
			}
		}
	}

	// Required columns
	studentIDIdx, ok := colIndex["student_id"]
	if !ok {
		return nil, &apperrors.ValidationError{Field: "csv", Message: "missing required column: student_id"}
	}
	gradeIdx, ok := colIndex["grade"]
	if !ok {
		return nil, &apperrors.ValidationError{Field: "csv", Message: "missing required column: grade"}
	}

	// Detect which MD/PDF columns are present
	var hasMD bool
	mdIdx := make(map[string]int)
	for _, c := range mdColumns {
		if idx, ok := colIndex[c]; ok {
			mdIdx[c] = idx
			hasMD = true
		}
	}

	// Parse rows
	var students []StudentGrade
	seenIDs := make(map[int]bool)
	for i := 1; i < len(records); i++ {
		row := records[i]
		rowNum := i + 1 // 1-based, accounting for header

		sid := strings.TrimSpace(getColumn(row, studentIDIdx))
		if sid == "" {
			return nil, &apperrors.ValidationError{
				Field:   "student_id",
				Message: fmt.Sprintf("empty student_id on row %d", rowNum),
			}
		}

		gradeStr := strings.TrimSpace(getColumn(row, gradeIdx))
		if gradeStr == "" {
			return nil, &apperrors.ValidationError{
				Field:   "grade",
				Message: fmt.Sprintf("empty grade on row %d (student %s)", rowNum, sid),
			}
		}

		grade, err := strconv.ParseFloat(gradeStr, 64)
		if err != nil || math.IsNaN(grade) || math.IsInf(grade, 0) {
			return nil, &apperrors.ValidationError{
				Field:   "grade",
				Message: fmt.Sprintf("invalid grade on row %d (student %s): %s", rowNum, sid, gradeStr),
			}
		}

		// Validate student_id is numeric
		id, err := strconv.Atoi(sid)
		if err != nil || id <= 0 {
			return nil, &apperrors.ValidationError{
				Field:   "student_id",
				Message: fmt.Sprintf("invalid student ID on row %d: %s", rowNum, sid),
			}
		}

		if seenIDs[id] {
			return nil, &apperrors.ValidationError{Field: "student_id", Message: fmt.Sprintf("duplicate student ID on row %d: %s", rowNum, sid)}
		}
		seenIDs[id] = true

		sg := StudentGrade{
			StudentID: sid,
			Grade:     grade,
			Comment: func() string {
				if idx, ok := colIndex["comment"]; ok {
					return strings.TrimSpace(getColumn(row, idx))
				}
				return ""
			}(),
		}

		// Resolve file paths
		for _, c := range mdColumns {
			path := l.resolve(row, colIndex, c)
			switch c {
			case "md_exam_file1":
				sg.MDExamFile1 = path
			case "md_exam_file2":
				sg.MDExamFile2 = path
			case "md_eval_file":
				sg.MDEvalFile = path
			}
		}
		for _, c := range pdfColumns {
			path := l.resolve(row, colIndex, c)
			switch c {
			case "pdf_exam_file1":
				sg.PDFExamFile1 = path
			case "pdf_exam_file2":
				sg.PDFExamFile2 = path
			case "pdf_eval_file":
				sg.PDFEvalFile = path
			}
		}

		students = append(students, sg)
	}

	result := &LoadResult{
		Students: students,
		HasMD:    hasMD,
	}

	return result, nil
}

// ConvertMarkdownFiles converts all Markdown files referenced in the result to PDF.
// On success, the PDF file paths in Students are updated to point to the converted files.
func (l *Loader) ConvertMarkdownFiles(result *LoadResult, converter MarkdownConverter) error {
	if !result.HasMD {
		return nil
	}

	var issues []FileIssue
	for i := range result.Students {
		sg := &result.Students[i]

		pairs := []struct {
			mdPath  string
			pdfDest *string
		}{
			{sg.MDExamFile1, &sg.PDFExamFile1},
			{sg.MDExamFile2, &sg.PDFExamFile2},
			{sg.MDEvalFile, &sg.PDFEvalFile},
		}

		for _, p := range pairs {
			if p.mdPath == "" {
				continue
			}
			pdfPath := replaceExt(p.mdPath, ".pdf")
			if err := converter.ConvertFile(p.mdPath, pdfPath); err != nil {
				issues = append(issues, FileIssue{sg.StudentID, p.mdPath, "conversion error: " + err.Error()})
				continue
			}
			*p.pdfDest = pdfPath
		}
	}
	if len(issues) > 0 {
		return &FileProblems{Issues: issues}
	}
	return nil
}

// --- internal helpers ---

func getColumn(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func (l *Loader) resolve(row []string, colIndex map[string]int, colName string) string {
	idx, ok := colIndex[colName]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	path := strings.TrimSpace(row[idx])
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.rootDir, path)
	}
	return filepath.Clean(path)
}

// CheckFiles validates attachments for the matched rows only.
func (l *Loader) CheckFiles(result *LoadResult) error {
	return l.checkFiles(result.Students)
}

func (l *Loader) checkFiles(students []StudentGrade) error {
	var issues []FileIssue
	for i := range students {
		sg := &students[i]

		type fileCheck struct {
			path string
			ext  string
			kind string
		}
		checks := []fileCheck{
			{sg.MDExamFile1, ".md", "Markdown"},
			{sg.MDExamFile2, ".md", "Markdown"},
			{sg.MDEvalFile, ".md", "Markdown"},
			{sg.PDFExamFile1, ".pdf", "PDF"},
			{sg.PDFExamFile2, ".pdf", "PDF"},
			{sg.PDFEvalFile, ".pdf", "PDF"},
		}

		for _, c := range checks {
			if c.path == "" {
				continue
			}
			if err := CheckReadable(c.path); err != nil {
				issues = append(issues, FileIssue{sg.StudentID, c.path, fileReason(err)})
				continue
			}
			if !strings.HasSuffix(strings.ToLower(c.path), c.ext) {
				issues = append(issues, FileIssue{sg.StudentID, c.path, "invalid " + c.kind + " file extension (expected " + c.ext + ")"})
			}
		}
	}
	if len(issues) > 0 {
		return &FileProblems{Issues: issues}
	}
	return nil
}

// NormalizeCSVPath accepts paths copied with balanced Explorer/shell quotes.
func NormalizeCSVPath(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && ((path[0] == '"' && path[len(path)-1] == '"') || (path[0] == '\'' && path[len(path)-1] == '\'')) {
		path = strings.TrimSpace(path[1 : len(path)-1])
	}
	return path
}

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)] + newExt
}
