package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultBaseURL    = "https://canvas.instructure.com"
	defaultTimeout    = 30 * time.Second
	defaultPerPage    = 100
	maxFileSizeMB     = 50
	uploadTimeout     = 60 * time.Second
)

// httpCanvasClient is the concrete implementation of CanvasClient.
type httpCanvasClient struct {
	baseURL  string
	token    string
	http     *http.Client
	limiter  *rate.Limiter
}

// NewCanvasClient creates a new Canvas API client.
// Token is read from the CANVAS_API_TOKEN environment variable.
func NewCanvasClient(baseURL string) (CanvasClient, error) {
	token := os.Getenv("CANVAS_API_TOKEN")
	if token == "" {
		return nil, &AuthError{Message: "CANVAS_API_TOKEN environment variable is not set"}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &httpCanvasClient{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
		limiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 req/s burst
	}, nil
}

// NewCanvasClientWithToken creates a client with an explicit token (for testing).
func NewCanvasClientWithToken(baseURL, token string) (CanvasClient, error) {
	if token == "" {
		return nil, &AuthError{Message: "token is empty"}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &httpCanvasClient{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
		limiter: rate.NewLimiter(rate.Every(time.Second), 10),
	}, nil
}

// do executes an HTTP request with rate limiting and error handling.
func (c *httpCanvasClient) do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return c.checkError(resp)
}

// get performs a GET request to a Canvas API endpoint.
func (c *httpCanvasClient) get(ctx context.Context, endpoint string) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// getQuiz performs a GET request to the new Canvas Quiz API.
func (c *httpCanvasClient) getQuiz(ctx context.Context, endpoint string) (*http.Response, error) {
	url := c.baseURL + "/api/quiz/v1" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// post performs a POST request with a JSON body.
func (c *httpCanvasClient) post(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + endpoint
	bodyReader, err := jsonBody(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// put performs a PUT request with a JSON body.
func (c *httpCanvasClient) put(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + endpoint
	bodyReader, err := jsonBody(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// deleteReq performs a DELETE request.
func (c *httpCanvasClient) deleteReq(ctx context.Context, endpoint string) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// checkError parses Canvas error responses and returns typed errors.
func (c *httpCanvasClient) checkError(resp *http.Response) (*http.Response, error) {
	if resp.StatusCode < 400 {
		return resp, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, &AuthError{Message: "token rejected by Canvas"}
	}

	var env canvasErrorEnvelope
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &env); err == nil && len(env.Errors) > 0 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: env.Errors[0].Message}
	}

	return nil, &APIError{StatusCode: resp.StatusCode, Message: resp.Status}
}

// decodeJSON reads and unmarshals a JSON response body.
func decodeJSON(resp *http.Response, target any) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// jsonBody marshals a value and returns a reader for use as an HTTP body.
func jsonBody(v any) (io.Reader, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(string(data)), nil
}

// contentType maps file extensions to MIME types.
func contentType(ext string) string {
	types := map[string]string{
		".pdf":  "application/pdf",
		".md":   "text/markdown",
		".txt":  "text/plain",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
	}
	if mime, ok := types[strings.ToLower(ext)]; ok {
		return mime
	}
	return "application/octet-stream"
}

// allowedExtensions for file uploads.
var allowedExtensions = map[string]bool{
	".pdf": true, ".md": true, ".txt": true, ".docx": true,
	".png": true, ".jpg": true, ".jpeg": true, ".mp3": true, ".wav": true,
}

// intPtr returns a pointer to an int.
func intPtr(v int) *int { return &v }

// floatPtr returns a pointer to a float64.
func floatPtr(v float64) *float64 { return &v }

// stringPtr returns a pointer to a string.
func stringPtr(v string) *string { return &v }

// formatDuration formats a duration as a human-readable string.
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	s := seconds % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// formatFileSize formats a byte count as a human-readable string.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
