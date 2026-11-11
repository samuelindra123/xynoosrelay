package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xynoss/relay/services/api/internal/logger"
)

// Standard Error Codes per docs/05-api.md
const (
	CodeMalformedRequest   = "MALFORMED_REQUEST"
	CodeUnauthenticated    = "UNAUTHENTICATED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeCursorExpired      = "CURSOR_EXPIRED"
	CodeRevisionMismatch   = "REVISION_MISMATCH"
	CodePayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	CodeValidationError    = "VALIDATION_ERROR"
	CodeRateLimited        = "RATE_LIMITED"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ErrorDetail represents specific validation or field-level error info.
type ErrorDetail struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// ErrorBody matches the error schema defined in docs/05-api.md.
type ErrorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Details   []ErrorDetail `json:"details,omitempty"`
	RequestID string        `json:"request_id"`
}

// ErrorEnvelope wraps ErrorBody.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// SuccessMeta represents pagination metadata.
type SuccessMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// SuccessEnvelope wraps successful response data.
type SuccessEnvelope struct {
	Data      any          `json:"data"`
	Meta      *SuccessMeta `json:"meta,omitempty"`
	RequestID string       `json:"request_id"`
}

// APIError is a structured error containing HTTP status code and standardized error code.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    []ErrorDetail
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// Error constructors
func NewError(statusCode int, code, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

func NewValidationError(message string, details []ErrorDetail) *APIError {
	return &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       CodeValidationError,
		Message:    message,
		Details:    details,
	}
}

func NewNotFoundError(message string) *APIError {
	return NewError(http.StatusNotFound, CodeNotFound, message)
}

func NewUnauthorizedError(message string) *APIError {
	return NewError(http.StatusUnauthorized, CodeUnauthenticated, message)
}

func NewForbiddenError(message string) *APIError {
	return NewError(http.StatusForbidden, CodeForbidden, message)
}

func NewBadRequestError(message string) *APIError {
	return NewError(http.StatusBadRequest, CodeMalformedRequest, message)
}

func NewRateLimitedError(retryAfterSec int, message string) *APIError {
	return NewError(http.StatusTooManyRequests, CodeRateLimited, message)
}

func NewValidationErrorSingle(field, message string) *APIError {
	return NewValidationError(message, []ErrorDetail{
		{Field: field, Code: message},
	})
}

func NewConflictError(message string) *APIError {
	return NewError(http.StatusConflict, CodeConflict, message)
}

func NewInternalError(err error) *APIError {
	return &APIError{
		StatusCode: http.StatusInternalServerError,
		Code:       CodeInternalError,
		Message:    "an internal error occurred",
		Err:        err,
	}
}

func NewServiceUnavailableError(message string) *APIError {
	return NewError(http.StatusServiceUnavailable, CodeServiceUnavailable, message)
}

// JSON writes a JSON response with status and standard data envelope.
func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	JSONWithMeta(w, r, statusCode, data, nil)
}

// JSONWithMeta writes a JSON response with status, data, and optional pagination meta.
func JSONWithMeta(w http.ResponseWriter, r *http.Request, statusCode int, data any, meta *SuccessMeta) {
	reqID := logger.GetRequestID(r.Context())

	payload := SuccessEnvelope{
		Data:      data,
		Meta:      meta,
		RequestID: reqID,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// RenderError writes an error response conforming to docs/05-api.md.
func RenderError(w http.ResponseWriter, r *http.Request, err error) {
	reqID := logger.GetRequestID(r.Context())

	var apiErr *APIError
	if e, ok := err.(*APIError); ok {
		apiErr = e
	} else {
		apiErr = NewInternalError(err)
	}

	statusCode := apiErr.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	payload := ErrorEnvelope{
		Error: ErrorBody{
			Code:      apiErr.Code,
			Message:   apiErr.Message,
			Details:   apiErr.Details,
			RequestID: reqID,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
