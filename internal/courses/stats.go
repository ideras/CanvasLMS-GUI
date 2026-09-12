package courses

import (
	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/models"
	"context"
	"fmt"
)

func GetCourseStats(ctx context.Context, client api.CanvasClient, courseID int) (*models.CourseStats, error) {
	students, err := client.GetStudentsForCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("get students: %w", err)
	}
	items, err := ListCourseItems(ctx, client, courseID)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	stats := &models.CourseStats{
		StudentCount: len(students),
		TotalItems:   len(items),
	}
	for _, item := range items {
		if item.Published {
			stats.PublishedItems++
		}
		stats.NeedsGrading += item.NeedsGradingCount
	}
	return stats, nil
}

func ListCourseItems(ctx context.Context, client api.CanvasClient, courseID int) ([]models.CourseItem, error) {
	assignments, err := client.GetAssignmentsForCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	quizzes, err := client.GetQuizzesForCourse(ctx, courseID)
	if err != nil {
		quizzes = nil
	}

	// Build set of assignment IDs linked from quizzes so we can deduplicate.
	quizAssignmentIDs := make(map[int]bool)
	for _, q := range quizzes {
		if q.AssignmentID != nil {
			quizAssignmentIDs[*q.AssignmentID] = true
		}
	}

	items := make([]models.CourseItem, 0, len(assignments)+len(quizzes))
	for _, a := range assignments {
		// Skip assignments that are already represented by a quiz.
		if quizAssignmentIDs[a.ID] {
			continue
		}
		items = append(items, models.CourseItem{
			ID:                a.ID,
			Name:              a.Name,
			Type:              "assignment",
			DueAt:             a.DueAt,
			Points:            a.PointsPossible,
			Published:         a.Published,
			NeedsGradingCount: a.NeedsGradingCount,
		})
	}
	for _, q := range quizzes {
		name := q.Name
		if name == "" {
			name = q.Title
		}
		assignID := 0
		if q.AssignmentID != nil {
			assignID = *q.AssignmentID
		}
		// Determine if this is an old-style quiz (has downloadable questions).
		// New quizzes (LTI / external_tool) don't expose questions via the API.
		isOld := q.QuizType != "" && q.QuizType != "quizzes.next" && q.QuizType != "new_quiz"

		items = append(items, models.CourseItem{
			ID:           q.ID,
			Name:         name,
			Type:         "quiz",
			DueAt:        q.DueAt,
			Points:       q.PointsPossible,
			Published:    q.Published,
			QuizType:     q.QuizType,
			AssignmentID: assignID,
			IsOldQuiz:    isOld,
		})
	}
	return items, nil
}
