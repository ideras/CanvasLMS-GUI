package api

import (
	"context"
	"fmt"
	"strings"

	"canvaslms-gui/internal/models"
)

// --- Courses ---

func (c *httpCanvasClient) GetAllCourses(ctx context.Context) ([]models.Course, error) {
	var courses []models.Course
	if err := c.fetchAllPages(ctx, "/courses?enrollment_state=active&per_page=100", &courses); err != nil {
		return nil, fmt.Errorf("get all courses: %w", err)
	}
	return courses, nil
}

// --- Folders ---

func (c *httpCanvasClient) GetFoldersForCourse(ctx context.Context, courseID int) ([]models.Folder, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/courses/%d/folders", courseID))
	if err != nil {
		return nil, fmt.Errorf("get folders for course %d: %w", courseID, err)
	}
	var folders []models.Folder
	if err := decodeJSON(resp, &folders); err != nil {
		return nil, fmt.Errorf("decode folders: %w", err)
	}
	return folders, nil
}

func (c *httpCanvasClient) EnsureCourseFolder(ctx context.Context, courseID int, folderPath string) (*models.Folder, error) {
	// Normalize path
	normPath := strings.Trim(folderPath, "/")
	parts := []string{}
	for _, p := range strings.Split(normPath, "/") {
		if p != "" && p != "." {
			parts = append(parts, p)
		}
	}

	// Try to get the folder by path directly
	resp, err := c.get(ctx, fmt.Sprintf("/courses/%d/folders/by_path/%s", courseID, normPath))
	if err == nil {
		var chain []models.Folder
		if decodeErr := decodeJSON(resp, &chain); decodeErr == nil && len(chain) > 0 {
			return &chain[len(chain)-1], nil
		}
	}

	// Build folder path step by step
	rootResp, err := c.get(ctx, fmt.Sprintf("/courses/%d/folders/root", courseID))
	if err != nil {
		return nil, fmt.Errorf("get root folder: %w", err)
	}
	var root models.Folder
	if err := decodeJSON(rootResp, &root); err != nil {
		return nil, fmt.Errorf("decode root folder: %w", err)
	}

	current := &root
	for _, seg := range parts {
		resp, err := c.get(ctx, fmt.Sprintf("/folders/%d/folders", current.ID))
		if err != nil {
			return nil, fmt.Errorf("list folders under %d: %w", current.ID, err)
		}
		var children []models.Folder
		if err := decodeJSON(resp, &children); err != nil {
			return nil, fmt.Errorf("decode children: %w", err)
		}

		found := false
		for i := range children {
			if children[i].Name == seg {
				current = &children[i]
				found = true
				break
			}
		}

		if !found {
			createResp, err := c.post(ctx, fmt.Sprintf("/folders/%d/folders", current.ID), map[string]string{
				"name": seg,
			})
			if err != nil {
				return nil, fmt.Errorf("create folder %s: %w", seg, err)
			}
			var created models.Folder
			if err := decodeJSON(createResp, &created); err != nil {
				return nil, fmt.Errorf("decode created folder: %w", err)
			}
			current = &created
		}
	}

	return current, nil
}
