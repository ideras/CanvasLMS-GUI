package grades_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "canvaslms-gui/internal/errors"
	"canvaslms-gui/internal/grades"
)

// writeCSV creates a temporary CSV file and returns its path.
func writeCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "grades.csv")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

// writeFile creates a file with content in dir and returns its path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

// --- Table-driven CSV parsing tests ---

func TestLoad_CSVValidation(t *testing.T) {
	tests := []struct {
		name      string
		csv       string
		files     map[string]string // relative name → content (created in CSV dir)
		wantErr   bool
		errField  string // expected ValidationError.Field (if wantErr)
		wantCount int
		wantGrade float64
	}{
		{
			name:      "valid minimal input",
			csv:       "student_id,grade\n1001,95.5\n1002,80\n",
			wantCount: 2,
			wantGrade: 95.5,
		},
		{
			name:    "missing grade column",
			csv:     "student_id,file\n1001,rubric.md\n",
			wantErr: true,
		},
		{
			name:    "empty file",
			csv:     "",
			wantErr: true,
		},
		{
			name:    "invalid student ID",
			csv:     "student_id,grade\nXXX,95.5\n",
			wantErr: true,
		},
		{
			name:    "invalid column name",
			csv:     "student_id,grade,unknown_column\n1001,95.5,x\n",
			wantErr: true,
		},
		{
			name: "canvas_id renamed to student_id",
			csv:  "canvas_id,grade\n1001,88\n",
			wantCount: 1,
			wantGrade: 88,
		},
		{
			name: "comments renamed to comment",
			csv:  "student_id,grade,comments\n1001,90,Nice work\n",
			wantCount: 1,
		},
		{
			name:    "empty student_id returns error",
			csv:     "student_id,grade\n,95.5\n",
			wantErr: true,
		},
		{
			name:    "empty grade returns error",
			csv:     "student_id,grade\n1001,\n",
			wantErr: true,
		},
		{
			name:    "non-numeric grade returns error",
			csv:     "student_id,grade\n1001,ABC\n",
			wantErr: true,
		},
		{
			name: "valid with comment column",
			csv:  "student_id,grade,comment\n1001,95.5,Great job\n1002,80,Needs improvement\n",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeCSV(t, tt.csv)

			if tt.files != nil {
				dir := filepath.Dir(path)
				for name, content := range tt.files {
					writeFile(t, dir, name, content)
				}
			}

			loader := grades.NewLoader(path, "")
			result, err := loader.Load()

			if tt.wantErr {
				require.Error(t, err)
				var valErr *apperrors.ValidationError
				if assert.ErrorAs(t, err, &valErr) {
					if tt.errField != "" {
						assert.Equal(t, tt.errField, valErr.Field)
					}
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Students, tt.wantCount)
			if tt.wantGrade > 0 && len(result.Students) > 0 {
				assert.Equal(t, tt.wantGrade, result.Students[0].Grade)
			}
		})
	}
}

// --- File existence and extension tests ---

func TestLoad_FileValidation(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		files   map[string]string
		wantErr bool
	}{
		{
			name: "PDF file exists",
			csv:  "student_id,grade,pdf_exam_file1\n1001,90,exam1.pdf\n",
			files: map[string]string{
				"exam1.pdf": "pdf content",
			},
			wantErr: false,
		},
		{
			name: "PDF file missing",
			csv:  "student_id,grade,pdf_exam_file1\n1001,90,missing.pdf\n",
			wantErr: true,
		},
		{
			name: "Markdown file exists",
			csv:  "student_id,grade,md_eval_file\n1001,90,eval.md\n",
			files: map[string]string{
				"eval.md": "# Evaluation",
			},
			wantErr: false,
		},
		{
			name: "Markdown file with wrong extension",
			csv:  "student_id,grade,md_eval_file\n1001,90,eval.txt\n",
			files: map[string]string{
				"eval.txt": "# Evaluation",
			},
			wantErr: true,
		},
		{
			name: "PDF column with .txt file",
			csv:  "student_id,grade,pdf_exam_file1\n1001,90,exam.txt\n",
			files: map[string]string{
				"exam.txt": "pdf content",
			},
			wantErr: true,
		},
		{
			name: "relative paths resolved against CSV dir",
			csv:  "student_id,grade,pdf_eval_file\n1001,90,subdir/eval.pdf\n",
			files: map[string]string{
				"subdir/eval.pdf": "evaluation",
			},
			wantErr: false,
		},
		{
			name: "empty file path skipped",
			csv:  "student_id,grade,pdf_exam_file1\n1001,90,\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "grades.csv")
			err := os.WriteFile(path, []byte(tt.csv), 0644)
			require.NoError(t, err)

			for name, content := range tt.files {
				fp := filepath.Join(dir, name)
				os.MkdirAll(filepath.Dir(fp), 0755)
				err := os.WriteFile(fp, []byte(content), 0644)
				require.NoError(t, err)
			}

			loader := grades.NewLoader(path, dir)
			result, err := loader.Load()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Students, 1)
		})
	}
}

// --- Markdown detection ---

func TestLoad_HasMD(t *testing.T) {
	path := writeCSV(t, "student_id,grade,md_eval_file\n1001,90,eval.md\n")
	dir := filepath.Dir(path)
	writeFile(t, dir, "eval.md", "# Eval")

	loader := grades.NewLoader(path, "")
	result, err := loader.Load()

	require.NoError(t, err)
	assert.True(t, result.HasMD)
}

func TestLoad_NoMD(t *testing.T) {
	path := writeCSV(t, "student_id,grade\n1001,90\n")

	loader := grades.NewLoader(path, "")
	result, err := loader.Load()

	require.NoError(t, err)
	assert.False(t, result.HasMD)
}

// --- Markdown conversion ---

type mockConverter struct {
	converted []string
}

func (m *mockConverter) ConvertFile(inputPath, outputPath string) error {
	m.converted = append(m.converted, inputPath)
	return nil
}

func TestConvertMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "grades.csv")
	mdPath := filepath.Join(dir, "eval.md")
	err := os.WriteFile(csvPath, []byte("student_id,grade,md_eval_file\n1001,90,eval.md\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(mdPath, []byte("# Evaluation"), 0644)
	require.NoError(t, err)

	loader := grades.NewLoader(csvPath, dir)
	result, err := loader.Load()
	require.NoError(t, err)
	require.True(t, result.HasMD)

	conv := &mockConverter{}
	err = loader.ConvertMarkdownFiles(result, conv)
	require.NoError(t, err)

	assert.Len(t, conv.converted, 1)
	assert.Equal(t, mdPath, conv.converted[0])

	// PDF path should be updated
	assert.Equal(t, filepath.Join(dir, "eval.pdf"), result.Students[0].PDFEvalFile)
}

func TestConvertMarkdownFiles_NoMD(t *testing.T) {
	path := writeCSV(t, "student_id,grade\n1001,90\n")

	loader := grades.NewLoader(path, "")
	result, err := loader.Load()
	require.NoError(t, err)

	conv := &mockConverter{}
	err = loader.ConvertMarkdownFiles(result, conv)
	require.NoError(t, err)
	assert.Empty(t, conv.converted)
}

func TestLoad_NoComment(t *testing.T) {
	path := writeCSV(t, "student_id,grade\n1001,95.5\n")

	loader := grades.NewLoader(path, "")
	result, err := loader.Load()

	require.NoError(t, err)
	require.Len(t, result.Students, 1)
	assert.Equal(t, "", result.Students[0].Comment)
}

// --- CSV file not found ---

func TestLoad_FileNotFound(t *testing.T) {
	loader := grades.NewLoader("/nonexistent/grades.csv", "")
	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open CSV")
}

// --- rootDir defaults to CSV directory ---

func TestNewLoader_DefaultRootDir(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "grades.csv")
	pdfPath := filepath.Join(dir, "exam.pdf")
	os.WriteFile(csvPath, []byte("student_id,grade,pdf_exam_file1\n1001,90,exam.pdf\n"), 0644)
	os.WriteFile(pdfPath, []byte("pdf"), 0644)

	loader := grades.NewLoader(csvPath, "")
	result, err := loader.Load()

	require.NoError(t, err)
	assert.Contains(t, result.Students[0].PDFExamFile1, dir)
}
