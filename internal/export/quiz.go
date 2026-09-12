package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"

	"canvaslms-gui/internal/models"
)

// ExportQuizQuestions fetches quiz questions and saves them as Markdown or JSON.
// Returns the path of the saved file, or empty if the user cancelled.
func (e *canvasExporter) ExportQuizQuestions(courseID, quizID int, format string, filePath string) (string, error) {
	// Fetch the quiz
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

	questions, err := e.client.GetQuizQuestions(e.ctx, courseID, quizID)
	if err != nil {
		return "", fmt.Errorf("get questions: %w", err)
	}

	title := quiz.Name
	if title == "" {
		title = quiz.Title
	}
	if title == "" {
		title = "Untitled Quiz"
	}

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	switch {
	case strings.HasSuffix(filePath, ".json"):
		return filePath, writeQuizJSON(f, quiz, questions)
	case strings.HasSuffix(filePath, ".html"), strings.HasSuffix(filePath, ".htm"):
		return filePath, writeQuizHTML(f, quiz, questions)
	default: // .md or anything else
		return filePath, writeQuizMarkdown(f, quiz, questions)
	}
}

// --- quiz export helpers ---
func toMD(htmlText string) string {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(
				commonmark.WithListEndComment(false),
			),
		),
	)

	md, err := conv.ConvertString(htmlText)
	if err != nil {
		return htmlText
	}
	return md
}

func writeQuizMarkdown(w io.Writer, quiz *models.Quiz, questions []models.QuizQuestion) error {
	title := quiz.Name
	if title == "" {
		title = quiz.Title
	}

	fmt.Fprintf(w, "# %s\n\n", title)
	fmt.Fprintf(w, "**Quiz Details**\n\n")
	fmt.Fprintf(w, "- **ID:** %d\n", quiz.ID)
	fmt.Fprintf(w, "- **Type:** %s\n", quiz.QuizType)
	fmt.Fprintf(w, "- **Points:** %s\n", formatPointsMD(quiz.PointsPossible))
	fmt.Fprintf(w, "- **Questions:** %d\n", quiz.QuestionCount)

	if quiz.TimeLimit != nil {
		fmt.Fprintf(w, "- **Time Limit:** %d minutes\n", *quiz.TimeLimit)
	}
	if quiz.DueAt != nil {
		fmt.Fprintf(w, "- **Due:** %s\n", (*quiz.DueAt)[:min(16, len(*quiz.DueAt))])
	}
	if quiz.LockAt != nil {
		fmt.Fprintf(w, "- **Locked After:** %s\n", (*quiz.LockAt)[:min(16, len(*quiz.LockAt))])
	}
	if quiz.UnlockAt != nil {
		fmt.Fprintf(w, "- **Available From:** %s\n", (*quiz.UnlockAt)[:min(16, len(*quiz.UnlockAt))])
	}
	if quiz.AllowedAttempts != nil {
		if *quiz.AllowedAttempts == -1 {
			fmt.Fprintf(w, "- **Attempts:** Unlimited\n")
		} else {
			fmt.Fprintf(w, "- **Attempts:** %d\n", *quiz.AllowedAttempts)
		}
	}
	fmt.Fprintf(w, "- **Published:** %s\n", yesNo(quiz.Published))
	fmt.Fprintf(w, "- **Shuffle Answers:** %s\n", yesNo(quiz.ShuffleAnswers))
	fmt.Fprintf(w, "- **One Question at a Time:** %s\n", yesNo(quiz.OneQuestionAtATime))
	fmt.Fprintf(w, "- **Can't Go Back:** %s\n", yesNo(quiz.CantGoBack))
	if quiz.AccessCode != nil {
		fmt.Fprintf(w, "- **Access Code:** %s\n", *quiz.AccessCode)
	}
	fmt.Fprintf(w, "- **Scoring Policy:** %s\n", quiz.ScoringPolicy)
	fmt.Fprintf(w, "- **URL:** %s\n\n", quiz.HTMLURL)

	if quiz.Description != "" {
		fmt.Fprintf(w, "## Description\n\n")
		md := toMD(quiz.Description)
		fmt.Fprint(w, md)
		fmt.Fprint(w, "\n\n")
	}

	fmt.Fprintf(w, "## Questions\n\n")

	for i, q := range questions {
		qName := q.QuestionName
		if qName == "" {
			qName = fmt.Sprintf("Question %d", i+1)
		}
		fmt.Fprintf(w, "### Question %d: %s\n\n", i+1, qName)
		fmt.Fprintf(w, "**Type:** %s  \n", q.QuestionType)
		fmt.Fprintf(w, "**Points:** %s\n\n", formatPointsMD(q.PointsPossible))

		if q.QuestionText != "" {
			md := toMD(q.QuestionText)
			fmt.Fprint(w, md)
			fmt.Fprint(w, "\n\n")
		}

		if len(q.Answers) > 0 && q.QuestionType != "essay_question" {
			fmt.Fprintf(w, "**Answer Options:**\n\n")
			for _, ans := range q.Answers {
				if ans.Weight > 0 {
					fmt.Fprintf(w, "- [x] **%s** (correct)\n", ans.Text)
				} else {
					fmt.Fprintf(w, "- [ ] %s\n", ans.Text)
				}
			}
			fmt.Fprint(w, "\n")
		}

		if q.CorrectComments != nil && *q.CorrectComments != "" {
			fmt.Fprintf(w, "**Correct Feedback:**\n\n")
			md := toMD(*q.CorrectComments)
			fmt.Fprint(w, md)
			fmt.Fprint(w, "\n\n")
		}
		if q.IncorrectComments != nil && *q.IncorrectComments != "" {
			fmt.Fprintf(w, "**Incorrect Feedback:**\n\n")
			md := toMD(*q.IncorrectComments)
			fmt.Fprint(w, md)
			fmt.Fprint(w, "\n\n")
		}
		if q.NeutralComments != nil && *q.NeutralComments != "" {
			fmt.Fprintf(w, "**Neutral Feedback:**\n\n")
			md := toMD(*q.NeutralComments)
			fmt.Fprint(w, md)
			fmt.Fprint(w, "\n\n")
		}

		fmt.Fprint(w, "---\n\n")
	}
	return nil
}

func writeQuizHTML(w io.Writer, quiz *models.Quiz, questions []models.QuizQuestion) error {
	title := quiz.Name
	if title == "" {
		title = quiz.Title
	}

	fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head><meta charset=\"UTF-8\"><title>%s</title></head>\n<body>\n", title)
	fmt.Fprintf(w, "<h1>%s</h1>\n", title)
	fmt.Fprintf(w, "<h2>Quiz Details</h2>\n<ul>\n")
	fmt.Fprintf(w, "<li><strong>ID:</strong> %d</li>\n", quiz.ID)
	fmt.Fprintf(w, "<li><strong>Type:</strong> %s</li>\n", quiz.QuizType)
	fmt.Fprintf(w, "<li><strong>Points:</strong> %s</li>\n", formatPointsMD(quiz.PointsPossible))
	fmt.Fprintf(w, "<li><strong>Questions:</strong> %d</li>\n", quiz.QuestionCount)
	if quiz.TimeLimit != nil {
		fmt.Fprintf(w, "<li><strong>Time Limit:</strong> %d minutes</li>\n", *quiz.TimeLimit)
	}
	if quiz.DueAt != nil {
		fmt.Fprintf(w, "<li><strong>Due:</strong> %s</li>\n", *quiz.DueAt)
	}
	fmt.Fprintf(w, "</ul>\n")

	if quiz.Description != "" {
		fmt.Fprintf(w, "<h2>Description</h2>\n%s\n", quiz.Description)
	}

	fmt.Fprintf(w, "<h2>Questions</h2>\n")
	for i, q := range questions {
		qName := q.QuestionName
		if qName == "" {
			qName = fmt.Sprintf("Question %d", i+1)
		}
		fmt.Fprintf(w, "<h3>Question %d: %s</h3>\n", i+1, qName)
		fmt.Fprintf(w, "<p><strong>Type:</strong> %s | <strong>Points:</strong> %s</p>\n", q.QuestionType, formatPointsMD(q.PointsPossible))

		if q.QuestionText != "" {
			fmt.Fprintf(w, "%s\n", q.QuestionText)
		}

		if len(q.Answers) > 0 {
			fmt.Fprintf(w, "<h4>Answer Options:</h4>\n<ul>\n")
			for _, ans := range q.Answers {
				mark := ""
				if ans.Weight > 0 {
					mark = " (correct)"
				}
				fmt.Fprintf(w, "<li><strong>%s</strong>%s</li>\n", ans.Text, mark)
			}
			fmt.Fprintf(w, "</ul>\n")
		}

		if q.CorrectComments != nil && *q.CorrectComments != "" {
			fmt.Fprintf(w, "<p><strong>Correct Feedback:</strong></p>\n%s\n", *q.CorrectComments)
		}
		if q.IncorrectComments != nil && *q.IncorrectComments != "" {
			fmt.Fprintf(w, "<p><strong>Incorrect Feedback:</strong></p>\n%s\n", *q.IncorrectComments)
		}

		fmt.Fprintf(w, "<hr>\n")
	}
	fmt.Fprintf(w, "</body>\n</html>\n")
	return nil
}

func writeQuizJSON(w io.Writer, quiz *models.Quiz, questions []models.QuizQuestion) error {
	data := map[string]any{
		"quiz":      quiz,
		"questions": questions,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func formatPointsMD(p float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", p), "0"), ".")
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
