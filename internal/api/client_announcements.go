package api

import (
	"context"
	"fmt"

	"canvaslms-gui/internal/models"
)

// --- Announcements ---

func (c *httpCanvasClient) GetAnnouncements(ctx context.Context, courseID int) ([]models.Announcement, error) {
	var announcements []models.Announcement
	path := fmt.Sprintf("/courses/%d/discussion_topics?only_announcements=true&per_page=100", courseID)
	if err := c.fetchAllPages(ctx, path, &announcements); err != nil {
		return nil, fmt.Errorf("get announcements for course %d: %w", courseID, err)
	}
	return announcements, nil
}

func (c *httpCanvasClient) CreateAnnouncement(ctx context.Context, courseID int, title, message string) (*models.Announcement, error) {
	body := map[string]any{
		"title":            title,
		"message":          message,
		"is_announcement":  true,
		"published":        true,
	}
	resp, err := c.post(ctx, fmt.Sprintf("/courses/%d/discussion_topics", courseID), body)
	if err != nil {
		return nil, fmt.Errorf("create announcement in course %d: %w", courseID, err)
	}
	var ann models.Announcement
	if err := decodeJSON(resp, &ann); err != nil {
		return nil, fmt.Errorf("decode announcement: %w", err)
	}
	return &ann, nil
}

func (c *httpCanvasClient) UpdateAnnouncement(ctx context.Context, courseID, topicID int, title, message string) (*models.Announcement, error) {
	body := map[string]any{
		"title":   title,
		"message": message,
	}
	resp, err := c.put(ctx, fmt.Sprintf("/courses/%d/discussion_topics/%d", courseID, topicID), body)
	if err != nil {
		return nil, fmt.Errorf("update announcement %d in course %d: %w", topicID, courseID, err)
	}
	var ann models.Announcement
	if err := decodeJSON(resp, &ann); err != nil {
		return nil, fmt.Errorf("decode announcement: %w", err)
	}
	return &ann, nil
}

func (c *httpCanvasClient) DeleteAnnouncement(ctx context.Context, courseID, topicID int) error {
	_, err := c.deleteReq(ctx, fmt.Sprintf("/courses/%d/discussion_topics/%d", courseID, topicID))
	if err != nil {
		return fmt.Errorf("delete announcement %d in course %d: %w", topicID, courseID, err)
	}
	return nil
}
