package errors

import "fmt"

// APIError is an HTTP-level failure (4xx/5xx from Canvas).
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Canvas API error %d: %s", e.StatusCode, e.Message)
}

// AuthError indicates the token is missing or rejected (401).
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error: %s", e.Message)
}

// ValidationError indicates a CSV structure or file problem.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
}
