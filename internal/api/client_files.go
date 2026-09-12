package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"canvaslms-gui/internal/models"
)

// --- File Download ---

func (c *httpCanvasClient) DownloadFile(ctx context.Context, fileURL, localPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Message: resp.Status}
	}

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// GetFileDownloadURL returns a temporary pre-authenticated download URL for a
// Canvas file, obtained via the REST API (GET /api/v1/files/:id).
func (c *httpCanvasClient) GetFileDownloadURL(ctx context.Context, fileID int) (string, error) {
	var result struct {
		URL         string `json:"url"`
		DisplayName string `json:"display_name"`
	}
	resp, err := c.get(ctx, fmt.Sprintf("/files/%d", fileID))
	if err != nil {
		return "", fmt.Errorf("get file info for %d: %w", fileID, err)
	}
	if err := decodeJSON(resp, &result); err != nil {
		return "", fmt.Errorf("decode file info: %w", err)
	}
	if result.URL == "" {
		return "", fmt.Errorf("no download URL for file %d", fileID)
	}
	return result.URL, nil
}

// --- File Upload ---

func (c *httpCanvasClient) UploadFileToCourse(ctx context.Context, filePath string, courseID, parentFolderID int) (*models.FileInfo, error) {
	// Validate file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Validate size
	maxSize := int64(maxFileSizeMB * 1024 * 1024)
	if info.Size() > maxSize {
		return nil, &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file too large: %d bytes (max: %d bytes)", info.Size(), maxSize),
		}
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !allowedExtensions[ext] {
		return nil, &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file type not allowed: %s", ext),
		}
	}

	// Step 1: Request upload slot from Canvas
	uploadInfo := map[string]any{
		"name":             filepath.Base(filePath),
		"size":             info.Size(),
		"content_type":     contentType(ext),
		"parent_folder_id": parentFolderID,
	}

	resp, err := c.post(ctx, fmt.Sprintf("/courses/%d/files", courseID), uploadInfo)
	if err != nil {
		return nil, fmt.Errorf("request upload slot: %w", err)
	}

	var uploadParams models.UploadParams
	if err := decodeJSON(resp, &uploadParams); err != nil {
		return nil, fmt.Errorf("decode upload params: %w", err)
	}

	// Step 2: POST file to the upload URL
	fileData, err := c.postFile(ctx, uploadParams.UploadURL, uploadParams.UploadParams, filePath)
	if err != nil {
		return nil, fmt.Errorf("upload file data: %w", err)
	}

	fileInfo := &models.FileInfo{
		ID:          fileData.ID,
		Name:        fileData.DisplayName,
		URL:         fmt.Sprintf("%s/courses/%d/files/%d", c.baseURL, courseID, fileData.ID),
		DownloadURL: fmt.Sprintf("%s/courses/%d/files/%d/download", c.baseURL, courseID, fileData.ID),
		PublicURL:   fileData.URL,
	}

	return fileInfo, nil
}

// postFile uploads a file with multipart form data including additional params.
func (c *httpCanvasClient) postFile(ctx context.Context, uploadURL string, params map[string]string, filePath string) (*models.FileUploadResponse, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add form params
	for k, v := range params {
		w.WriteField(k, v)
	}

	// Add file
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy file data: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &b)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: uploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	var fileData models.FileUploadResponse
	if err := jsonDecodeReader(resp.Body, &fileData); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	return &fileData, nil
}
