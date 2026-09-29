package grades

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"canvaslms-gui/internal/models"
)

// StudentIssue identifies a CSV row or a Canvas-reported failure.
type StudentIssue struct {
	StudentID string `json:"student_id"`
	Row       int    `json:"row,omitempty"`
	Reason    string `json:"reason"`
}

// MatchStudents follows the CSV convention: numeric Canvas user IDs, not SIS
// IDs, login IDs or email. Row numbers include the header.
func MatchStudents(rows []StudentGrade, roster []models.User) ([]StudentGrade, []StudentIssue) {
	ids := make(map[int]bool, len(roster))
	for _, student := range roster {
		ids[student.ID] = true
	}
	matched := make([]StudentGrade, 0, len(rows))
	unmatched := make([]StudentIssue, 0)
	for i, row := range rows {
		id, _ := strconv.Atoi(row.StudentID)
		if !ids[id] {
			unmatched = append(unmatched, StudentIssue{row.StudentID, i + 2, "not found in course roster"})
			continue
		}
		row.StudentID = strconv.Itoa(id)
		matched = append(matched, row)
	}
	return matched, unmatched
}

// ConfirmStudents pauses the worker, not the GUI, until a user decision arrives.
func ConfirmStudents(ctx context.Context, emitter EventEmitter, decisions <-chan string, students []StudentIssue) error {
	emitter.Emit("upload:unmatched", map[string]any{"students": students})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case action := <-decisions:
		if action == "ignore" {
			return nil
		}
		return context.Canceled
	}
}

var missingUserIDs = regexp.MustCompile(`'([0-9]+)'`)

// JobIssues reads optional Progress results. Canvas upstream currently reports
// missing users in the failed job's message; some deployments also supply
// structured errors/results. Preserve unknown result content as a warning,
// rather than silently presenting it as a clean success.
func JobIssues(p models.BatchProgress) []StudentIssue {
	issues := make([]StudentIssue, 0)
	if strings.Contains(p.Message, "Couldn't find User(s)") {
		for _, match := range missingUserIDs.FindAllStringSubmatch(p.Message, -1) {
			issues = append(issues, StudentIssue{StudentID: match[1], Reason: p.Message})
		}
	}
	if len(p.Results) == 0 || string(p.Results) == "null" || string(p.Results) == "{}" {
		return issues
	}
	var results map[string]json.RawMessage
	if json.Unmarshal(p.Results, &results) != nil {
		return append(issues, StudentIssue{Reason: "Canvas job results: " + string(p.Results)})
	}
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := results[key]
		switch key {
		case "errors", "failed_students":
			var byID map[string]any
			if json.Unmarshal(value, &byID) == nil {
				ids := make([]string, 0, len(byID))
				for id := range byID {
					ids = append(ids, id)
				}
				sort.Strings(ids)
				for _, id := range ids {
					issues = append(issues, StudentIssue{StudentID: id, Reason: fmt.Sprint(byID[id])})
				}
				continue
			}
			var list []struct {
				StudentID json.RawMessage `json:"student_id"`
				UserID    json.RawMessage `json:"user_id"`
				Message   string          `json:"message"`
				Error     string          `json:"error"`
			}
			if json.Unmarshal(value, &list) == nil {
				for _, item := range list {
					id := item.StudentID
					if len(id) == 0 {
						id = item.UserID
					}
					reason := item.Message
					if reason == "" {
						reason = item.Error
					}
					if reason == "" {
						reason = string(value)
					}
					issues = append(issues, StudentIssue{StudentID: strings.Trim(string(id), `"`), Reason: reason})
				}
				continue
			}
		case "missing_user_ids":
			var ids []json.RawMessage
			if json.Unmarshal(value, &ids) == nil {
				for _, id := range ids {
					issues = append(issues, StudentIssue{StudentID: strings.Trim(string(id), `"`), Reason: "Canvas could not find student"})
				}
				continue
			}
		}
		issues = append(issues, StudentIssue{Reason: "Canvas result " + key + ": " + string(value)})
	}
	return issues
}
