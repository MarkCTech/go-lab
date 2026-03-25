// Package api defines stable JSON response envelopes for the platform API.
package api

// Meta carries cross-cutting response metadata.
type Meta struct {
	RequestID string `json:"request_id"`
}

// OKEnvelope wraps successful responses.
type OKEnvelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

// ErrBody is the structured error payload.
type ErrBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorEnvelope wraps error responses.
type ErrorEnvelope struct {
	Error ErrBody `json:"error"`
	Meta  Meta    `json:"meta"`
}

// Common error codes (stable for clients).
const (
	CodeValidation     = "VALIDATION_ERROR"
	CodeNotFound       = "NOT_FOUND"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeConflict       = "CONFLICT"
	CodeInternal       = "INTERNAL_ERROR"
	CodeNotImplemented = "NOT_IMPLEMENTED"
	CodeRateLimited    = "RATE_LIMITED"
)
