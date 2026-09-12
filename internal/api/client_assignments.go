package api

import (
	"context"
	"fmt"

	"canvaslms-gui/internal/models"
)

// --- Assignments ---

func (c *httpCanvasClient) GetAssignmentsForCourse(ctx context.Context, courseID int) ([]models.Assignment, error) {
	var assignments []models.Assignment
	endpoint := fmt.Sprintf("/courses/%d/assignments?per_page=100&include[]=assignment_group", courseID)
	if err := c.fetchAllPages(ctx, endpoint, &assignments); err != nil {
		return nil, fmt.Errorf("get assignments for course %d: %w", courseID, err)
	}
	return assignments, nil
}

func (c *httpCanvasClient) CreateAssignment(ctx context.Context, courseID int, data map[string]any) (*models.Assignment, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/courses/%d/assignments", courseID), data)
	if err != nil {
		return nil, fmt.Errorf("create assignment in course %d: %w", courseID, err)
	}
	var assignment models.Assignment
	if err := decodeJSON(resp, &assignment); err != nil {
		return nil, fmt.Errorf("decode assignment: %w", err)
	}
	return &assignment, nil
}

func (c *httpCanvasClient) DeleteAssignment(ctx context.Context, courseID, assignmentID int) error {
	resp, err := c.deleteReq(ctx, fmt.Sprintf("/courses/%d/assignments/%d", courseID, assignmentID))
	if err != nil {
		return fmt.Errorf("delete assignment %d in course %d: %w", assignmentID, courseID, err)
	}
	resp.Body.Close()
	return nil
}

func (c *httpCanvasClient) EditAssignment(ctx context.Context, courseID, assignmentID int, data map[string]any) (*models.Assignment, error) {
	resp, err := c.put(ctx, fmt.Sprintf("/courses/%d/assignments/%d", courseID, assignmentID), data)
	if err != nil {
		return nil, fmt.Errorf("edit assignment %d in course %d: %w", assignmentID, courseID, err)
	}
	var assignment models.Assignment
	if err := decodeJSON(resp, &assignment); err != nil {
		return nil, fmt.Errorf("decode assignment: %w", err)
	}
	return &assignment, nil
}

// --- Assignment Groups ---

func (c *httpCanvasClient) GetAssignmentGroupsForCourse(ctx context.Context, courseID int) ([]models.AssignmentGroup, error) {
	var groups []models.AssignmentGroup
	endpoint := fmt.Sprintf("/courses/%d/assignment_groups?per_page=100", courseID)
	if err := c.fetchAllPages(ctx, endpoint, &groups); err != nil {
		return nil, fmt.Errorf("get assignment groups for course %d: %w", courseID, err)
	}
	return groups, nil
}

func (c *httpCanvasClient) CreateAssignmentGroup(ctx context.Context, courseID int, name string) (*models.AssignmentGroup, error) {
	resp, err := c.post(ctx, fmt.Sprintf("/courses/%d/assignment_groups", courseID), map[string]string{
		"name": name,
	})
	if err != nil {
		return nil, fmt.Errorf("create assignment group in course %d: %w", courseID, err)
	}
	var group models.AssignmentGroup
	if err := decodeJSON(resp, &group); err != nil {
		return nil, fmt.Errorf("decode assignment group: %w", err)
	}
	return &group, nil
}

// --- Submissions ---

func (c *httpCanvasClient) GetSubmissionsForAssignment(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	var submissions []models.Submission
	endpoint := fmt.Sprintf("/courses/%d/assignments/%d/submissions?include[]=user&per_page=100", courseID, assignmentID)
	if err := c.fetchAllPages(ctx, endpoint, &submissions); err != nil {
		return nil, fmt.Errorf("get submissions for assignment %d: %w", assignmentID, err)
	}
	return submissions, nil
}

func (c *httpCanvasClient) GetAssignmentSubmissionsWithAttachments(ctx context.Context, courseID, assignmentID int) ([]models.Submission, error) {
	endpoint := fmt.Sprintf(
		"/courses/%d/assignments/%d/submissions?include[]=user&include[]=submission_history&include[]=attachments&per_page=100",
		courseID, assignmentID,
	)
	resp, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("get submissions with attachments: %w", err)
	}
	var submissions []models.Submission
	if err := decodeJSON(resp, &submissions); err != nil {
		return nil, fmt.Errorf("decode submissions: %w", err)
	}
	return submissions, nil
}

func (c *httpCanvasClient) GetSubmissionsForStudent(ctx context.Context, courseID, studentID int) ([]models.Submission, error) {
	var submissions []models.Submission
	endpoint := fmt.Sprintf("/courses/%d/students/submissions?student_ids[]=%d&per_page=100", courseID, studentID)
	if err := c.fetchAllPages(ctx, endpoint, &submissions); err != nil {
		return nil, fmt.Errorf("get submissions for student %d: %w", studentID, err)
	}
	return submissions, nil
}

func (c *httpCanvasClient) GetAllSubmissionsForCourse(ctx context.Context, courseID int) ([]models.Submission, error) {
	var submissions []models.Submission
	endpoint := fmt.Sprintf("/courses/%d/students/submissions?student_ids[]=all&per_page=100", courseID)
	if err := c.fetchAllPages(ctx, endpoint, &submissions); err != nil {
		return nil, fmt.Errorf("get all submissions for course %d: %w", courseID, err)
	}
	return submissions, nil
}

// --- Users / Enrollments ---

func (c *httpCanvasClient) GetStudentsForCourse(ctx context.Context, courseID int) ([]models.User, error) {
	var users []models.User
	endpoint := fmt.Sprintf("/courses/%d/users?enrollment_type[]=student&per_page=100", courseID)
	if err := c.fetchAllPages(ctx, endpoint, &users); err != nil {
		return nil, fmt.Errorf("get students for course %d: %w", courseID, err)
	}
	return users, nil
}

func (c *httpCanvasClient) GetEnrollmentsForCourse(ctx context.Context, courseID int) ([]models.Enrollment, error) {
	var enrollments []models.Enrollment
	endpoint := fmt.Sprintf("/courses/%d/enrollments?type[]=StudentEnrollment&per_page=100", courseID)
	if err := c.fetchAllPages(ctx, endpoint, &enrollments); err != nil {
		return nil, fmt.Errorf("get enrollments for course %d: %w", courseID, err)
	}
	return enrollments, nil
}
