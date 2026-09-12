package models

// Folder represents a Canvas course folder.
type Folder struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// FileUploadResponse is the response from Canvas after uploading a file.
type FileUploadResponse struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
}

// UploadParams holds the upload URL and form parameters returned by Canvas
// when requesting a file upload slot.
type UploadParams struct {
	UploadURL    string            `json:"upload_url"`
	UploadParams map[string]string `json:"upload_params"`
}

// FileInfo is the enriched file metadata stored after a successful upload.
type FileInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	PublicURL   string `json:"public_url"`
	FolderPath  string `json:"folder_path"`
}
