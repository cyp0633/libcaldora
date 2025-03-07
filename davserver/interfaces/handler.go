package interfaces

import (
	"log/slog"
	"net/http"
)

// Handler defines the interface for handling CalDAV requests
type Handler interface {
	// ServeHTTP handles all CalDAV requests
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Individual method handlers that can be used directly with HTTP routers
	HandlePropFind(w http.ResponseWriter, r *http.Request)
	HandleReport(w http.ResponseWriter, r *http.Request)
	HandleGet(w http.ResponseWriter, r *http.Request)
	HandlePut(w http.ResponseWriter, r *http.Request)
	HandleDelete(w http.ResponseWriter, r *http.Request)
	HandleOptions(w http.ResponseWriter, r *http.Request)
	HandleMkCol(w http.ResponseWriter, r *http.Request)
}

// HandlerConfig contains configuration for the CalDAV handler
type HandlerConfig struct {
	// Provider is the CalendarProvider implementation
	Provider CalendarProvider

	// URLPrefix is the base path where the CalDAV server is mounted
	// For example: "/caldav/"
	URLPrefix string

	// AllowedMethods specifies which HTTP methods are allowed
	// If nil, all methods are allowed
	AllowedMethods []string

	// CustomHeaders allows adding custom headers to responses
	CustomHeaders map[string]string

	// Logger is the slog.Logger to use for logging
	// If nil, logging is disabled
	Logger *slog.Logger
}

// Option is a function that modifies HandlerConfig
type Option func(*HandlerConfig)

// WithURLPrefix sets the URL prefix for the handler
func WithURLPrefix(prefix string) Option {
	return func(c *HandlerConfig) {
		c.URLPrefix = prefix
	}
}

// WithAllowedMethods sets the allowed HTTP methods
func WithAllowedMethods(methods []string) Option {
	return func(c *HandlerConfig) {
		c.AllowedMethods = methods
	}
}

// WithCustomHeaders sets custom response headers
func WithCustomHeaders(headers map[string]string) Option {
	return func(c *HandlerConfig) {
		c.CustomHeaders = headers
	}
}

// WithLogger sets the logger for the handler
func WithLogger(logger *slog.Logger) Option {
	return func(c *HandlerConfig) {
		c.Logger = logger
	}
}

// ErrorResponse represents an error response in XML format
type ErrorResponse struct {
	XMLName   string `xml:"DAV: error"`
	Namespace string `xml:"xmlns,attr"`
	Message   string `xml:",innerxml"`
}

// HTTPError represents an HTTP error with status code and message
type HTTPError struct {
	Status  int
	Message string
	Err     error
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// Common HTTP errors
var (
	ErrMethodNotAllowed = &HTTPError{Status: http.StatusMethodNotAllowed, Message: "Method not allowed"}
	ErrNotFound         = &HTTPError{Status: http.StatusNotFound, Message: "Resource not found"}
	ErrForbidden        = &HTTPError{Status: http.StatusForbidden, Message: "Access denied"}
	ErrPrecondition     = &HTTPError{Status: http.StatusPreconditionFailed, Message: "Precondition failed"}
)
