package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"canvaslms-gui/internal/models"
)

// ExportQuizSubmissions downloads all submissions for an old-style quiz.
// Each student gets a separate file in outputDir. Format is determined by
// the file extension: .md, .json, or .html.
// ProgressFunc is called as each student submission file is written.
type ProgressFunc func(current, total int, student string)

func (e *canvasExporter) ExportQuizSubmissions(courseID, quizID int, format string, dirPath string, progress ProgressFunc) (string, error) {
	quizzes, err := e.client.GetQuizzesForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get quizzes: %w", err)
	}

	var quiz *models.Quiz
	for i := range quizzes {
		if quizzes[i].ID == quizID {
			quiz = &quizzes[i]
			break
		}
	}
	if quiz == nil {
		return "", fmt.Errorf("quiz %d not found in course %d", quizID, courseID)
	}

	// Only old-style quizzes are supported
	isOld := quiz.QuizType != "" && quiz.QuizType != "quizzes.next" && quiz.QuizType != "new_quiz"
	if !isOld {
		return "", fmt.Errorf("quiz submissions are only available for old-style quizzes")
	}

	title := quiz.Name
	if title == "" {
		title = quiz.Title
	}

	// Get quiz submissions
	subsData, err := e.client.GetQuizSubmissions(e.ctx, courseID, quizID)
	if err != nil {
		return "", fmt.Errorf("get quiz submissions: %w", err)
	}
	submissions := subsData.QuizSubmissions

	// Get assignment-linked submissions for answer details
	var assignmentSubMap map[int]models.Submission
	if quiz.AssignmentID != nil {
		assignmentSubs, err := e.client.GetAssignmentSubmissionsWithAttachments(e.ctx, courseID, *quiz.AssignmentID)
		if err == nil {
			assignmentSubMap = make(map[int]models.Submission)
			for _, s := range assignmentSubs {
				assignmentSubMap[s.UserID] = s
			}
		}
	}

	// Get student roster
	students, err := e.client.GetStudentsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get students: %w", err)
	}
	studentMap := make(map[int]models.User)
	for _, s := range students {
		studentMap[s.ID] = s
	}

	safeTitle := SanitizeFilename(title)
	outputDir := filepath.Join(dirPath, fmt.Sprintf("quiz_%d_%s_submissions", quizID, safeTitle))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	count := 0
	for _, sub := range submissions {
		student := studentMap[sub.UserID]
		studentName := fmt.Sprintf("user_%d", sub.UserID)
		if student.Name != "" {
			studentName = SanitizeFilename(student.Name)
		}

		var filename string
		switch {
		case strings.HasSuffix(format, ".json") || format == "json":
			filename = filepath.Join(outputDir, fmt.Sprintf("submission_%d_%s.json", sub.ID, studentName))
			if err := writeSubmissionJSON(filename, quiz, &sub, student, assignmentSubMap); err != nil {
				return "", err
			}
		case strings.HasSuffix(format, ".html") || format == "html":
			filename = filepath.Join(outputDir, fmt.Sprintf("%s.html", studentName))
			if err := writeSubmissionHTML(filename, quiz, &sub, student, assignmentSubMap); err != nil {
				return "", err
			}
		default: // .md
			filename = filepath.Join(outputDir, fmt.Sprintf("%s.md", studentName))
			if err := writeSubmissionMarkdown(filename, quiz, &sub, student, assignmentSubMap); err != nil {
				return "", err
			}
		}
		count++
		_ = filename
		if progress != nil {
			progress(count, len(submissions), studentName)
		}
	}

	return fmt.Sprintf("%d submissions saved to %s", count, outputDir), nil
}

// --- submission writers ---

func writeSubmissionMarkdown(path string, quiz *models.Quiz, sub *models.QuizSubmission, student models.User, assignMap map[int]models.Submission) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := f
	title := quiz.Title
	if title == "" {
		title = quiz.Name
	}

	fmt.Fprintf(w, "# Quiz Submission: %s\n\n", title)
	fmt.Fprintf(w, "**Student Information**\n\n")
	fmt.Fprintf(w, "- **Name:** %s\n", studentName(student))
	fmt.Fprintf(w, "- **Email:** %s\n", student.Email)
	fmt.Fprintf(w, "- **User ID:** %d\n", sub.UserID)
	fmt.Fprintf(w, "- **Submission ID:** %d\n\n", sub.ID)

	fmt.Fprintf(w, "**Submission Details**\n\n")
	if sub.StartedAt != nil {
		fmt.Fprintf(w, "- **Started:** %s\n", *sub.StartedAt)
	}
	if sub.FinishedAt != nil {
		fmt.Fprintf(w, "- **Finished:** %s\n", *sub.FinishedAt)
	}
	fmt.Fprintf(w, "- **Time Spent:** %s\n", formatDuration(sub.TimeSpent))
	fmt.Fprintf(w, "- **Attempt:** %d\n", sub.Attempt)
	fmt.Fprintf(w, "- **Score:** %s/%s\n", formatPointsMD(sub.Score), formatPointsMD(sub.QuizPointsPossible))
	fmt.Fprintf(w, "- **Status:** %s\n", sub.WorkflowState)
	fmt.Fprintf(w, "- **Quiz URL:** %s\n\n", sub.HTMLURL)

	// Answers from assignment submission
	if assignMap != nil {
		if aSub, ok := assignMap[sub.UserID]; ok {
			if len(aSub.SubmissionHistory) > 0 {
				history := aSub.SubmissionHistory[0]
				if len(history.SubmissionData) > 0 {
					fmt.Fprintf(w, "## Student Answers\n\n")
					for _, ans := range history.SubmissionData {
						qID := fmt.Sprintf("%v", ans.QuestionID)
						if qID == "" || qID == "<nil>" {
							qID = "Unknown"
						}
						fmt.Fprintf(w, "### Question ID: %s\n\n", qID)
						fmt.Fprintf(w, "**Points:** %s  \n", formatPointsMD(ans.Points))
						fmt.Fprintf(w, "**Correct:** %v\n\n", ans.Correct)

						if ans.Text != "" {
							fmt.Fprintf(w, "**Answer:**\n\n")
							md := toMD(ans.Text)
							fmt.Fprint(w, md)
							fmt.Fprint(w, "\n\n")
						} else {
							fmt.Fprintf(w, "**Answer:** *(no answer provided)*\n\n")
						}
						fmt.Fprint(w, "---\n\n")
					}
				}
			}
		}
	}

	return nil
}

func writeSubmissionJSON(path string, quiz *models.Quiz, sub *models.QuizSubmission, student models.User, assignMap map[int]models.Submission) error {
	data := map[string]any{
		"quiz_title":    quiz.Title,
		"submission":    sub,
		"student_name":  studentName(student),
		"student_email": student.Email,
	}

	if assignMap != nil {
		if aSub, ok := assignMap[sub.UserID]; ok {
			if len(aSub.SubmissionHistory) > 0 {
				data["answers"] = aSub.SubmissionHistory[0].SubmissionData
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeSubmissionHTML(path string, quiz *models.Quiz, sub *models.QuizSubmission, student models.User, assignMap map[int]models.Submission) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := f
	title := quiz.Title
	if title == "" {
		title = quiz.Name
	}

	fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head><meta charset=\"UTF-8\"><title>%s - %s</title></head>\n<body>\n", title, studentName(student))
	fmt.Fprintf(w, "<h1>Quiz Submission: %s</h1>\n", title)
	fmt.Fprintf(w, "<h2>Student Information</h2>\n<ul>\n")
	fmt.Fprintf(w, "<li><strong>Name:</strong> %s</li>\n", studentName(student))
	fmt.Fprintf(w, "<li><strong>Email:</strong> %s</li>\n", student.Email)
	fmt.Fprintf(w, "<li><strong>User ID:</strong> %d</li>\n", sub.UserID)
	fmt.Fprintf(w, "<li><strong>Submission ID:</strong> %d</li>\n</ul>\n", sub.ID)

	fmt.Fprintf(w, "<h2>Submission Details</h2>\n<ul>\n")
	if sub.StartedAt != nil {
		fmt.Fprintf(w, "<li><strong>Started:</strong> %s</li>\n", *sub.StartedAt)
	}
	if sub.FinishedAt != nil {
		fmt.Fprintf(w, "<li><strong>Finished:</strong> %s</li>\n", *sub.FinishedAt)
	}
	fmt.Fprintf(w, "<li><strong>Time Spent:</strong> %s</li>\n", formatDuration(sub.TimeSpent))
	fmt.Fprintf(w, "<li><strong>Attempt:</strong> %d</li>\n", sub.Attempt)
	fmt.Fprintf(w, "<li><strong>Score:</strong> %s/%s</li>\n", formatPointsMD(sub.Score), formatPointsMD(sub.QuizPointsPossible))
	fmt.Fprintf(w, "<li><strong>Status:</strong> %s</li>\n", sub.WorkflowState)
	fmt.Fprintf(w, "</ul>\n")

	if assignMap != nil {
		if aSub, ok := assignMap[sub.UserID]; ok {
			if len(aSub.SubmissionHistory) > 0 {
				history := aSub.SubmissionHistory[0]
				if len(history.SubmissionData) > 0 {
					fmt.Fprintf(w, "<h2>Student Answers</h2>\n")
					for _, ans := range history.SubmissionData {
						qID := fmt.Sprintf("%v", ans.QuestionID)
						if qID == "" || qID == "<nil>" {
							qID = "Unknown"
						}
						fmt.Fprintf(w, "<h3>Question ID: %s</h3>\n", qID)
						fmt.Fprintf(w, "<p><strong>Points:</strong> %s | <strong>Correct:</strong> %v</p>\n", formatPointsMD(ans.Points), ans.Correct)
						if ans.Text != "" {
							fmt.Fprintf(w, "<p><strong>Answer:</strong></p>\n%s\n", ans.Text)
						} else {
							fmt.Fprintf(w, "<p><em>(no answer provided)</em></p>\n")
						}
						fmt.Fprintf(w, "<hr>\n")
					}
				}
			}
		}
	}
	fmt.Fprintf(w, "</body>\n</html>\n")
	return nil
}

// --- helpers ---

func studentName(s models.User) string {
	if s.Name != "" {
		return s.Name
	}
	return fmt.Sprintf("User %d", s.ID)
}

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
