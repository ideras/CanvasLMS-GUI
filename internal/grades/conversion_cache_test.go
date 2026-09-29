package grades_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"canvaslms-gui/internal/grades"
	"github.com/stretchr/testify/require"
)

type variablePDFConverter struct {
	calls int
	fail  bool
}

func (c *variablePDFConverter) ConvertFile(_, output string) error {
	c.calls++
	if c.fail {
		return fmt.Errorf("conversion failed")
	}
	// Simulate non-deterministic creation-date metadata in generated PDFs.
	return os.WriteFile(output, []byte(fmt.Sprintf("PDF generation %d", c.calls)), 0600)
}

func TestConversionRetryKeepsIdenticalPDFAndDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	input := writeFile(t, dir, "feedback.md", "# Feedback")
	output := filepath.Join(dir, "feedback.pdf")
	cache := grades.NewUploadCache()
	real := &variablePDFConverter{}
	converter := cache.Converter(real)
	require.NoError(t, converter.ConvertFile(input, output))
	original, err := os.ReadFile(output)
	require.NoError(t, err)
	require.NoError(t, converter.ConvertFile(input, output))
	require.Equal(t, 1, real.calls)
	unchanged, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
	// Editing the source, editing the PDF, or deleting the PDF forces repair.
	require.NoError(t, os.WriteFile(input, []byte("# Revised feedback"), 0600))
	require.NoError(t, converter.ConvertFile(input, output))
	require.Equal(t, 2, real.calls)
	require.NoError(t, os.WriteFile(output, []byte("externally modified"), 0600))
	require.NoError(t, converter.ConvertFile(input, output))
	require.Equal(t, 3, real.calls)
	require.NoError(t, os.Remove(output))
	real.fail = true
	require.Error(t, converter.ConvertFile(input, output))
	real.fail = false
	require.NoError(t, converter.ConvertFile(input, output))
	require.Equal(t, 5, real.calls, "failed conversions must never be cached as successful")
}
