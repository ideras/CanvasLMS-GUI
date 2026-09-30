package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"canvaslms-gui/internal/models"
)

// --- Single Grade ---

func (c *httpCanvasClient) SubmitGrade(ctx context.Context, courseID, assignmentID, studentID int, grade string, comment string) error {
	endpoint := fmt.Sprintf("/courses/%d/assignments/%d/submissions/%d", courseID, assignmentID, studentID)

	body := map[string]any{
		"submission": map[string]any{
			"posted_grade": grade,
		},
	}
	if comment != "" {
		body["comment"] = map[string]any{
			"text_comment": comment,
		}
	}

	resp, err := c.put(ctx, endpoint, body)
	if err != nil {
		return fmt.Errorf("submit grade for student %d: %w", studentID, err)
	}
	resp.Body.Close()
	return nil
}

// --- Batch Grades ---

func (c *httpCanvasClient) BatchSubmitGrades(ctx context.Context, courseID, assignmentID int, entries []models.GradeEntry) (string, error) {
	endpoint := fmt.Sprintf("/courses/%d/assignments/%d/submissions/update_grades", courseID, assignmentID)

	gradeData := make(map[string]models.GradeData, len(entries))
	for _, e := range entries {
		gradeData[fmt.Sprintf("%d", e.StudentID)] = models.GradeData{
			PostedGrade: e.Grade,
			TextComment: e.Comment,
		}
	}

	payload := models.BatchGradePayload{GradeData: gradeData}

	resp, err := c.post(ctx, endpoint, payload)
	if err != nil {
		return "", fmt.Errorf("batch submit grades: %w", err)
	}

	// Canvas returns a Progress object or a direct response.
	// Extract the progress URL.
	var result struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	if err := decodeJSON(resp, &result); err != nil {
		return "", fmt.Errorf("decode batch submit response: %w", err)
	}

	// Return the progress URL if available, otherwise construct it from the ID.
	if result.URL != "" {
		return result.URL, nil
	}
	if result.ID <= 0 {
		return "", fmt.Errorf("Canvas returned no progress URL or valid job ID; submission outcome is unknown")
	}
	return fmt.Sprintf("%s/api/v1/progress/%d", c.baseURL, result.ID), nil
}

// --- Progress Polling ---

func (c *httpCanvasClient) QueryProgress(ctx context.Context, progressID int) (*models.BatchProgress, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/progress/%d", progressID))
	if err != nil {
		return nil, fmt.Errorf("query progress %d: %w", progressID, err)
	}
	var p models.BatchProgress
	if err := decodeJSON(resp, &p); err != nil {
		return nil, fmt.Errorf("decode progress: %w", err)
	}
	return &p, nil
}

func (c *httpCanvasClient) PollBatchProgress(ctx context.Context, progressURL string) (<-chan models.BatchProgress, error) {
	// Never send the Canvas token to a foreign origin supplied as a job URL.
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Canvas base URL")
	}
	jobURL, err := url.Parse(progressURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Canvas progress URL")
	}
	jobURL = base.ResolveReference(jobURL)
	if jobURL.User != nil || jobURL.Scheme != base.Scheme || jobURL.Host != base.Host {
		return nil, fmt.Errorf("Canvas progress URL has an unexpected origin")
	}
	updates := pollProgress(ctx, func(ctx context.Context) (models.BatchProgress, error) {
		var p models.BatchProgress
		err := c.getJSON(ctx, jobURL.String(), &p)
		return p, err
	}, gradePollPolicy)
	return updates, nil
}

// getJSON fetches a URL and decodes the JSON response.
func (c *httpCanvasClient) getJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}

	return decodeJSON(resp, target)
}
