package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/cache"
	"canvaslms-gui/internal/courses"
	"canvaslms-gui/internal/export"
	"canvaslms-gui/internal/models"
	"canvaslms-gui/internal/uploader"
	markdown "github.com/ideras/md-to-pdf"
)

// appConfig holds the parsed config.json fields.
type appConfig struct {
	CanvasBaseURL string `json:"canvas_base_url"`
	APIToken      string `json:"api_token"`
	LastCourseID  int    `json:"last_course_id"`
	CourseSet     []int  `json:"course_set"`
}

// App holds application state and exposes methods to the Svelte frontend.
type App struct {
	ctx           context.Context
	client        api.CanvasClient
	exporter      export.Exporter
	config        *appConfig
	configPath    string
	currentCourse *models.Course
	cancelUpload  context.CancelFunc
	cacheStore    cache.Store
}

// NewApp creates the application struct.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails at launch.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	a.config = loadConfig()
	if a.config == nil {
		a.config = &appConfig{}
	}

	// If no token is resolvable, leave client nil — the frontend will detect
	// NeedsSetup() and open the settings dialog.
	baseURL, token := a.effectiveCredentials()
	if token == "" {
		return
	}

	if err := a.initClientAndCache(baseURL, token); err != nil {
		runtime.EventsEmit(ctx, "app:error", map[string]any{"message": err.Error()})
		return
	}

	// Auto-restore last course.
	if a.config.LastCourseID != 0 {
		if _, err := a.SelectCourse(a.config.LastCourseID); err != nil {
			runtime.EventsEmit(ctx, "app:error", map[string]any{
				"message": fmt.Sprintf("failed to restore course %d: %v", a.config.LastCourseID, err),
			})
		}
	}
}

// effectiveCredentials returns the Canvas base URL and API token to use,
// resolving from env vars first then falling back to config.json.
func (a *App) effectiveCredentials() (baseURL, token string) {
	token = os.Getenv("CANVAS_API_TOKEN")
	if token == "" && a.config != nil {
		token = a.config.APIToken
	}
	baseURL = os.Getenv("CANVAS_BASE_URL")
	if baseURL == "" && a.config != nil {
		baseURL = a.config.CanvasBaseURL
	}
	return
}

// initClientAndCache creates a new Canvas API client backed by a per-token
// SQLite cache and installs it as the active client.  The old cache store
// (if any) is closed first.
func (a *App) initClientAndCache(baseURL, token string) error {
	client, err := api.NewCanvasClientWithToken(baseURL, token)
	if err != nil {
		return err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	cacheDir = filepath.Join(cacheDir, "canvaslms-gui")
	if mkErr := os.MkdirAll(cacheDir, 0o700); mkErr == nil {
		dbName := fmt.Sprintf("cache_%s.db", tokenCacheID(token))
		store, storeErr := cache.NewSQLiteStore(filepath.Join(cacheDir, dbName))
		if storeErr == nil {
			if a.cacheStore != nil {
				_ = a.cacheStore.Close()
			}
			a.cacheStore = store
			client = cache.NewCachedCanvasClient(client, store)
		}
	}

	a.client = client
	if a.ctx != nil {
		a.exporter = export.NewCanvasExporter(a.ctx, client)
	}
	return nil
}

// tokenCacheID returns the first 8 hex characters of the SHA-256 of the token.
// Used as a suffix for the per-token SQLite cache filename.
func tokenCacheID(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h[:4])
}

// configPaths returns possible locations for config.json.
func configPaths() []string {
	paths := []string{"config.json"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.json"))
	}
	return paths
}

func loadConfig() *appConfig {
	for _, p := range configPaths() {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg appConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		return &cfg
	}
	return nil
}

func saveConfig(cfg *appConfig) error {
	for _, p := range configPaths() {
		data, err := json.MarshalIndent(cfg, "", "    ")
		if err != nil {
			continue
		}
		if err := os.WriteFile(p, data, 0644); err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not write config.json")
}

// --- Settings ---

// NeedsSetup returns true when no API token is configured, meaning the
// frontend should open the settings dialog before attempting any API calls.
func (a *App) NeedsSetup() bool {
	return a.client == nil
}

// TokenFromEnv reports whether the API token is being supplied via the
// CANVAS_API_TOKEN environment variable (i.e. it cannot be changed from the UI).
func (a *App) TokenFromEnv() bool {
	return os.Getenv("CANVAS_API_TOKEN") != ""
}

// SaveSettings persists a new Canvas base URL and API token to config.json,
// then re-initialises the client and cache.  The current course is reset so
// no stale data from a previous session is shown.
func (a *App) SaveSettings(baseURL, apiToken string) error {
	if apiToken == "" {
		return fmt.Errorf("API token is required")
	}
	if baseURL == "" {
		return fmt.Errorf("Canvas base URL is required")
	}

	a.config.APIToken = apiToken
	a.config.CanvasBaseURL = baseURL
	if err := saveConfig(a.config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Re-initialise using effective credentials (env vars still take priority).
	effURL, effToken := a.effectiveCredentials()
	if err := a.initClientAndCache(effURL, effToken); err != nil {
		return err
	}

	// Reset course state — data belongs to the previous session.
	a.currentCourse = nil
	a.config.LastCourseID = 0
	_ = saveConfig(a.config)
	return nil
}

// --- Cache ---

func (a *App) InvalidateCache() error {
	if a.cacheStore == nil {
		return nil
	}
	return a.cacheStore.Delete("%") // LIKE '%' matches every row
}

func (a *App) InvalidateCourse(courseID int) error {
	if a.cacheStore == nil {
		return nil
	}
	return a.cacheStore.Delete(cache.PatternCourse(courseID))
}

// --- Courses ---

// ListCourses returns all active courses.
func (a *App) ListCourses() ([]models.Course, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not initialized — check CANVAS_API_TOKEN")
	}
	return a.client.GetAllCourses(a.ctx)
}

// SelectCourse sets the active course by ID and returns it.
// Persists the selection as last_course_id in config.json.
func (a *App) SelectCourse(courseID int) (*models.Course, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not initialized — check CANVAS_API_TOKEN")
	}
	courses, err := a.client.GetAllCourses(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("select course: %w", err)
	}
	for _, c := range courses {
		if c.ID == courseID {
			a.currentCourse = &c
			if a.config != nil {
				a.config.LastCourseID = courseID
				_ = saveConfig(a.config)
			}
			return &c, nil
		}
	}
	return nil, fmt.Errorf("course %d not found", courseID)
}

// GetCurrentCourse returns the currently selected course, if any.
func (a *App) GetCurrentCourse() *models.Course {
	return a.currentCourse
}

// GetConfig returns the current app configuration for the frontend.
func (a *App) GetConfig() *appConfig {
	return a.config
}

// SaveCourseSet persists the course set to config.json.
func (a *App) SaveCourseSet(ids []int) error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}
	a.config.CourseSet = ids
	return saveConfig(a.config)
}

// ListCourseItems returns merged assignments and quizzes for a course.
// Graded quizzes that also appear as assignments are deduplicated (quiz wins).
func (a *App) ListCourseItems(courseID int) ([]models.CourseItem, error) {
	return courses.ListCourseItems(a.ctx, a.client, courseID)
}

// GetCourseStats computes aggregate statistics for a course.
func (a *App) GetCourseStats(courseID int) (*models.CourseStats, error) {
	return courses.GetCourseStats(a.ctx, a.client, courseID)
}

// GetStudentSubmissions returns all submissions for a student across all assignments.
func (a *App) GetStudentSubmissions(courseID, studentID int) ([]models.Submission, error) {
	return a.client.GetSubmissionsForStudent(a.ctx, courseID, studentID)
}

// --- Assignments ---

// ListAssignments returns assignments for a course.
func (a *App) ListAssignments(courseID int) ([]models.Assignment, error) {
	return a.client.GetAssignmentsForCourse(a.ctx, courseID)
}

// CreateAssignment creates a new assignment in a course.
func (a *App) CreateAssignment(courseID int, data map[string]any) (*models.Assignment, error) {
	return a.client.CreateAssignment(a.ctx, courseID, data)
}

// EditAssignment updates an existing assignment in a course.
func (a *App) EditAssignment(courseID, assignmentID int, data map[string]any) (*models.Assignment, error) {
	return a.client.EditAssignment(a.ctx, courseID, assignmentID, data)
}

// GetAssignmentGroups returns assignment groups for a course.
func (a *App) GetAssignmentGroups(courseID int) ([]models.AssignmentGroup, error) {
	return a.client.GetAssignmentGroupsForCourse(a.ctx, courseID)
}

// CreateAssignmentGroup creates a new assignment group in a course.
func (a *App) CreateAssignmentGroup(courseID int, name string) (*models.AssignmentGroup, error) {
	return a.client.CreateAssignmentGroup(a.ctx, courseID, name)
}

// --- Announcements ---

// GetAnnouncements returns announcements for a course.
func (a *App) GetAnnouncements(courseID int) ([]models.Announcement, error) {
	return a.client.GetAnnouncements(a.ctx, courseID)
}

// CreateAnnouncement creates a new announcement in a course.
func (a *App) CreateAnnouncement(courseID int, title, message string) error {
	_, err := a.client.CreateAnnouncement(a.ctx, courseID, title, message)
	return err
}

// UpdateAnnouncement updates an existing announcement in a course.
func (a *App) UpdateAnnouncement(courseID, topicID int, title, message string) error {
	_, err := a.client.UpdateAnnouncement(a.ctx, courseID, topicID, title, message)
	return err
}

// DeleteAnnouncement deletes an announcement from a course.
func (a *App) DeleteAnnouncement(courseID, topicID int) error {
	return a.client.DeleteAnnouncement(a.ctx, courseID, topicID)
}

// UploadAnnouncementAttachment opens a file picker, ensures an "Anuncios"
// folder in the course, uploads the file, and returns its metadata so the
// frontend can embed a download link in the announcement message.
func (a *App) UploadAnnouncementAttachment(courseID int) (*models.FileInfo, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select file to attach",
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	folder, err := a.client.EnsureCourseFolder(a.ctx, courseID, "Anuncios")
	if err != nil {
		return nil, fmt.Errorf("ensure announcements folder: %w", err)
	}
	return a.client.UploadFileToCourse(a.ctx, path, courseID, folder.ID)
}

// DownloadAnnouncementFile downloads a file linked from an announcement.
// The frontend passes the link text as the suggested filename and the
// Canvas file URL.  We parse the file ID, get a pre-authenticated download
// URL from the REST API (the web URL requires a session cookie), then
// download the raw bytes.
func (a *App) DownloadAnnouncementFile(fileURL, suggestedName string) error {
	defaultName := strings.TrimSpace(suggestedName)
	if defaultName == "" {
		defaultName = "download"
	}

	fileID, err := parseFileID(fileURL)
	if err != nil {
		return err
	}

	dlURL, err := a.client.GetFileDownloadURL(a.ctx, fileID)
	if err != nil {
		return fmt.Errorf("resolve download URL for file %d: %w", fileID, err)
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Announcement Attachment",
		DefaultFilename: defaultName,
	})
	if err != nil {
		return err
	}
	if path == "" {
		return nil // user cancelled
	}

	return a.client.DownloadFile(a.ctx, dlURL, path)
}

// parseFileID extracts the numeric file ID from a Canvas file URL.
// Supports paths like /courses/123/files/456/download and
// /courses/123/files/456.
func parseFileID(rawURL string) (int, error) {
	// strip query string
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	// strip trailing slash
	rawURL = strings.TrimRight(rawURL, "/")

	// find /files/ segment
	idx := strings.Index(rawURL, "/files/")
	if idx < 0 {
		return 0, fmt.Errorf("not a Canvas file URL: %s", rawURL)
	}
	rest := rawURL[idx+len("/files/"):]
	// take the next path segment
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		rest = rest[:slash]
	}

	var id int
	if _, err := fmt.Sscanf(rest, "%d", &id); err != nil {
		return 0, fmt.Errorf("parse file ID from %q: %w", rest, err)
	}
	return id, nil
}

// --- Students ---

// ListStudents returns students enrolled in a course.
func (a *App) ListStudents(courseID int) ([]models.User, error) {
	return a.client.GetStudentsForCourse(a.ctx, courseID)
}

// ExportStudentsCSV opens a save dialog and writes the student roster as CSV.
// Returns the path of the saved file, or an empty string if the user cancelled.
func (a *App) ExportStudentsCSV(courseID int) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Student List",
		DefaultFilename: fmt.Sprintf("students_course_%d.csv", courseID),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}

	return a.exporter.ExportStudentsCSV(courseID, path)
}

// ExportScoresCSV opens a save dialog and writes a grades matrix CSV
// (canvas_id, student_name, one column per assignment/quiz).
func (a *App) ExportScoresCSV(courseID int) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Scores",
		DefaultFilename: fmt.Sprintf("scores_course_%d.csv", courseID),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}

	return a.exporter.ExportScoresCSV(courseID, path)
}

// --- Submissions ---

// ListSubmissions returns submissions for an assignment.
func (a *App) ListSubmissions(courseID, assignmentID int) ([]models.Submission, error) {
	return a.client.GetAssignmentSubmissionsWithAttachments(a.ctx, courseID, assignmentID)
}

// --- Grades Upload ---

func (a *App) BrowseCSVFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Grades CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
}

// BrowseDirectory opens a native OS directory picker and returns the selected path.
func (a *App) BrowseDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Folder",
	})
}

// UploadGrades runs the full grade upload workflow in the background.
// Progress is reported via Wails events: upload:status, upload:file_progress,
// upload:batch_progress, upload:done, upload:error.
func (a *App) UploadGrades(courseID int, assignmentID int, csvPath string) {
	a.cancelUpload = uploader.UploadGrades(a.ctx, a.client, courseID, assignmentID, csvPath)
}

// CancelUpload cancels a running grade upload.
func (a *App) CancelUpload() {
	if a.cancelUpload != nil {
		a.cancelUpload()
		a.cancelUpload = nil
	}
}

// --- Markdown Conversion ---

// ConvertMarkdown converts a Markdown file to PDF. Returns the output path.
func (a *App) ConvertMarkdown(inputPath, outputPath string) error {
	return markdown.ConvertFile(inputPath, outputPath)
}

func (app *App) ExportQuizQuestions(courseID, quizID int, quizTitle string, format string) (string, error) {
	safeTitle := export.SanitizeFilename(quizTitle)
	defaultName := fmt.Sprintf("quiz_%d_%s_questions.md", quizID, safeTitle)

	path, err := runtime.SaveFileDialog(app.ctx, runtime.SaveDialogOptions{
		Title:           "Save Quiz Questions",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown Files (*.md)", Pattern: "*.md"},
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
			{DisplayName: "HTML Files (*.html)", Pattern: "*.html"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	if path == "" {
		return "", nil
	}

	return app.exporter.ExportQuizQuestions(courseID, quizID, format, path)
}

// ExportQuizSubmissions downloads all submissions for an old-style quiz.
// Opens a directory picker, then writes one file per student in the background.
// Progress is reported via Wails events: submissions:progress, submissions:done, submissions:error.
func (app *App) ExportQuizSubmissions(courseID, quizID int, format string) {
	dir, err := runtime.OpenDirectoryDialog(app.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Folder for Quiz Submissions",
	})
	if err != nil {
		runtime.EventsEmit(app.ctx, "submissions:error", map[string]any{"error": err.Error()})
		return
	}
	if dir == "" {
		return
	}

	go func() {
		runtime.EventsEmit(app.ctx, "submissions:start", map[string]any{
			"message": "Downloading submissions...",
		})

		var lastTotal int
		msg, err := app.exporter.ExportQuizSubmissions(courseID, quizID, format, dir,
			func(current, total int, student string) {
				lastTotal = total
				runtime.EventsEmit(app.ctx, "submissions:progress", map[string]any{
					"current": current,
					"total":   total,
					"student": student,
				})
				time.Sleep(80 * time.Millisecond)
			})
		if err != nil {
			runtime.EventsEmit(app.ctx, "submissions:error", map[string]any{"error": err.Error()})
			return
		}
		runtime.EventsEmit(app.ctx, "submissions:done", map[string]any{
			"message": msg,
			"total":   lastTotal,
		})
	}()
}

// ExportAssignmentSubmissions downloads all file attachments for an assignment.
// dirPath is chosen by the user beforehand via BrowseDirectory.
// Progress is reported via Wails events: assign-dl:start, assign-dl:progress,
// assign-dl:done, assign-dl:error.
func (app *App) ExportAssignmentSubmissions(courseID, assignmentID int, dirPath string) {
	go func() {
		runtime.EventsEmit(app.ctx, "assign-dl:start", map[string]any{
			"message": "Downloading submissions...",
		})

		var lastTotal int
		msg, err := app.exporter.ExportAssignmentSubmissions(courseID, assignmentID, dirPath,
			func(current, total int, student string) {
				lastTotal = total
				runtime.EventsEmit(app.ctx, "assign-dl:progress", map[string]any{
					"current": current,
					"total":   total,
					"student": student,
				})
			},
		)
		if err != nil {
			runtime.EventsEmit(app.ctx, "assign-dl:error", map[string]any{"error": err.Error()})
			return
		}
		runtime.EventsEmit(app.ctx, "assign-dl:done", map[string]any{
			"message": msg,
			"total":   lastTotal,
		})
	}()
}
