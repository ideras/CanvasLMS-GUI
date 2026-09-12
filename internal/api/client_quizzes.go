package api

import (
	"context"
	"fmt"

	"canvaslms-gui/internal/models"
)

// --- Quizzes ---

func (c *httpCanvasClient) GetQuizzesForCourse(ctx context.Context, courseID int) ([]models.Quiz, error) {
	var quizList []models.Quiz

	// Old quiz API
	oldResp, err := c.get(ctx, fmt.Sprintf("/courses/%d/quizzes?per_page=100", courseID))
	if err == nil {
		var oldQuizzes []models.Quiz
		if decodeErr := decodeJSON(oldResp, &oldQuizzes); decodeErr == nil {
			quizList = append(quizList, oldQuizzes...)
		}
	}

	// New quiz API (LTI / external tools)
	newResp, err := c.getQuiz(ctx, fmt.Sprintf("/courses/%d/quizzes?per_page=100", courseID))
	if err == nil {
		// Response can be a list or a dict with a "quizzes" key
		body, readErr := readResponseBody(newResp)
		if readErr == nil {
			if len(body) > 0 && body[0] == '[' {
				var newQuizzes []models.Quiz
				if jsonErr := jsonUnmarshal(body, &newQuizzes); jsonErr == nil {
					quizList = append(quizList, newQuizzes...)
				}
			} else {
				var wrapper struct {
					Quizzes []models.Quiz `json:"quizzes"`
				}
				if jsonErr := jsonUnmarshal(body, &wrapper); jsonErr == nil {
					quizList = append(quizList, wrapper.Quizzes...)
				}
			}
		}
	}

	if len(quizList) == 0 {
		return nil, fmt.Errorf("no quizzes found for course %d (both old and new API failed)", courseID)
	}
	return quizList, nil
}

func (c *httpCanvasClient) GetQuizQuestions(ctx context.Context, courseID, quizID int) ([]models.QuizQuestion, error) {
	var questions []models.QuizQuestion

	// Try old API first
	oldResp, oldErr := c.get(ctx, fmt.Sprintf("/courses/%d/quizzes/%d/questions?per_page=100", courseID, quizID))
	if oldErr == nil {
		var oldQ []models.QuizQuestion
		if err := decodeJSON(oldResp, &oldQ); err == nil {
			questions = append(questions, oldQ...)
		}
	}

	// Try new quiz API
	newResp, newErr := c.getQuiz(ctx, fmt.Sprintf("/courses/%d/quizzes/%d/questions?per_page=100", courseID, quizID))
	if newErr == nil {
		body, readErr := readResponseBody(newResp)
		if readErr == nil {
			if len(body) > 0 && body[0] == '[' {
				var newQ []models.QuizQuestion
				if jsonErr := jsonUnmarshal(body, &newQ); jsonErr == nil {
					questions = append(questions, newQ...)
				}
			} else {
				var wrapper struct {
					Questions []models.QuizQuestion `json:"questions"`
				}
				if jsonErr := jsonUnmarshal(body, &wrapper); jsonErr == nil {
					questions = append(questions, wrapper.Questions...)
				}
			}
		}
	}

	// Both failed
	if oldErr != nil && newErr != nil {
		return nil, fmt.Errorf("get quiz questions for quiz %d: old API: %w, new API: %w", quizID, oldErr, newErr)
	}

	return questions, nil
}

func (c *httpCanvasClient) GetQuizSubmissions(ctx context.Context, courseID, quizID int) (*models.QuizSubmissionList, error) {
	// Try old API first
	oldResp, oldErr := c.get(ctx, fmt.Sprintf("/courses/%d/quizzes/%d/submissions?per_page=100", courseID, quizID))
	if oldErr == nil {
		body, readErr := readResponseBody(oldResp)
		if readErr == nil {
			// Old API may return a list or dict
			if len(body) > 0 && body[0] == '[' {
				var subs []models.QuizSubmission
				if jsonErr := jsonUnmarshal(body, &subs); jsonErr == nil {
					return &models.QuizSubmissionList{QuizSubmissions: subs}, nil
				}
			} else {
				var result models.QuizSubmissionList
				if jsonErr := jsonUnmarshal(body, &result); jsonErr == nil {
					return &result, nil
				}
			}
		}
	}

	// Try new quiz API
	newResp, newErr := c.getQuiz(ctx, fmt.Sprintf("/courses/%d/quizzes/%d/submissions?per_page=100", courseID, quizID))
	if newErr == nil {
		body, readErr := readResponseBody(newResp)
		if readErr == nil {
			// Response can be list, dict with "quiz_submissions", or dict with "submissions"
			if len(body) > 0 && body[0] == '[' {
				var subs []models.QuizSubmission
				if jsonErr := jsonUnmarshal(body, &subs); jsonErr == nil {
					return &models.QuizSubmissionList{QuizSubmissions: subs}, nil
				}
			} else {
				var wrapper struct {
					QuizSubmissions []models.QuizSubmission `json:"quiz_submissions"`
					Submissions     []models.QuizSubmission `json:"submissions"`
				}
				if jsonErr := jsonUnmarshal(body, &wrapper); jsonErr == nil {
					if len(wrapper.QuizSubmissions) > 0 {
						return &models.QuizSubmissionList{QuizSubmissions: wrapper.QuizSubmissions}, nil
					}
					return &models.QuizSubmissionList{QuizSubmissions: wrapper.Submissions}, nil
				}
			}
		}
	}

	// Both failed
	if oldErr != nil && newErr != nil {
		return nil, fmt.Errorf("get quiz submissions for quiz %d: old API: %w, new API: %w", quizID, oldErr, newErr)
	}

	return &models.QuizSubmissionList{}, nil
}
