package api

import "canvaslms-gui/internal/errors"

// Error types are defined in the shared errors package and re-exported here
// for convenience. All CanvasClient methods return these types.
type (
	APIError        = errors.APIError
	AuthError       = errors.AuthError
	ValidationError = errors.ValidationError
)

// canvasErrorEnvelope matches Canvas's JSON error response format.
type canvasErrorEnvelope struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}
