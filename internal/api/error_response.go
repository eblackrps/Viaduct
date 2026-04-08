package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type apiFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type apiError struct {
	Code        string          `json:"code"`
	Message     string          `json:"message"`
	RequestID   string          `json:"request_id"`
	Retryable   bool            `json:"retryable"`
	Details     map[string]any  `json:"details"`
	FieldErrors []apiFieldError `json:"field_errors"`
}

type apiErrorEnvelope struct {
	Error apiError `json:"error"`
}

type apiErrorOptions struct {
	Retryable   bool
	Details     map[string]any
	FieldErrors []apiFieldError
}

type specValidationError struct {
	message     string
	fieldErrors []apiFieldError
}

func (e specValidationError) Error() string {
	return e.message
}

func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string, options apiErrorOptions) {
	requestID := responseRequestID(w, r)
	payload := apiErrorEnvelope{
		Error: apiError{
			Code:        strings.TrimSpace(code),
			Message:     strings.TrimSpace(message),
			RequestID:   requestID,
			Retryable:   options.Retryable,
			Details:     cloneMap(options.Details),
			FieldErrors: append([]apiFieldError(nil), options.FieldErrors...),
		},
	}
	if payload.Error.Code == "" {
		payload.Error.Code = "internal_error"
	}
	if payload.Error.Message == "" {
		payload.Error.Message = http.StatusText(status)
	}
	if payload.Error.Details == nil {
		payload.Error.Details = map[string]any{}
	}
	if payload.Error.FieldErrors == nil {
		payload.Error.FieldErrors = []apiFieldError{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func responseRequestID(w http.ResponseWriter, r *http.Request) string {
	if w != nil {
		if requestID := strings.TrimSpace(w.Header().Get(requestIDHeader)); requestID != "" {
			return requestID
		}
	}
	if r != nil {
		if requestID := strings.TrimSpace(RequestIDFromContext(r.Context())); requestID != "" {
			if w != nil {
				w.Header().Set(requestIDHeader, requestID)
			}
			return requestID
		}
		if requestID := strings.TrimSpace(r.Header.Get(requestIDHeader)); requestID != "" {
			if w != nil {
				w.Header().Set(requestIDHeader, requestID)
			}
			return requestID
		}
	}

	requestID := uuid.NewString()
	if w != nil {
		w.Header().Set(requestIDHeader, requestID)
	}
	return requestID
}

func fieldErrorsFromValidationErrors(errs []error) []apiFieldError {
	fieldErrors := make([]apiFieldError, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		message := strings.TrimSpace(err.Error())
		path, ok := fieldPathFromValidationMessage(message)
		if !ok {
			continue
		}
		fieldErrors = append(fieldErrors, apiFieldError{
			Path:    path,
			Message: message,
		})
	}
	return fieldErrors
}

func fieldPathFromValidationMessage(message string) (string, bool) {
	if message == "" || strings.HasPrefix(message, "at least ") || strings.HasPrefix(message, "spec ") {
		return "", false
	}
	firstSpace := strings.Index(message, " ")
	if firstSpace <= 0 {
		return "", false
	}
	path := strings.TrimSpace(message[:firstSpace])
	if path == "" {
		return "", false
	}
	switch {
	case path == "name":
		return path, true
	case strings.HasPrefix(path, "source."):
		return path, true
	case strings.HasPrefix(path, "target."):
		return path, true
	case strings.HasPrefix(path, "options."):
		return path, true
	case strings.HasPrefix(path, "workloads["):
		return path, true
	default:
		return "", false
	}
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func executionErrorCode(err error) string {
	if err == nil {
		return "conflict"
	}
	message := strings.TrimSpace(err.Error())
	switch {
	case strings.Contains(message, "requires approval"):
		return "approval_required"
	case strings.Contains(message, "window opens"):
		return "window_not_open"
	case strings.Contains(message, "window closed"):
		return "window_closed"
	default:
		return "conflict"
	}
}
