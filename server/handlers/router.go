package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/server/storage"
)

const (
	// HTTP headers
	HeaderContentType = "Content-Type"
	HeaderETag        = "ETag"
	HeaderDAV         = "DAV"
	HeaderAllow       = "Allow"

	// MIME types
	MimeTypeCalendar = "text/calendar; charset=utf-8"

	// DAV capability values
	DavCapabilities = "1, calendar-access"
	AllowedMethods  = "OPTIONS, PROPFIND, REPORT, GET, PUT, DELETE, MKCOL"
)

// Router handles CalDAV request routing
type Router struct {
	storage  storage.Storage
	baseURI  string
	handlers map[string]http.HandlerFunc
	logger   *slog.Logger
}

// NewRouter creates a new CalDAV router
func NewRouter(storage storage.Storage, baseURI string, logger *slog.Logger) *Router {
	r := &Router{
		storage:  storage,
		baseURI:  baseURI,
		handlers: make(map[string]http.HandlerFunc),
		logger:   logger,
	}

	// Register method handlers
	r.handlers["OPTIONS"] = r.handleOptions
	r.handlers["PROPFIND"] = r.handlePropfind
	r.handlers["REPORT"] = r.handleReport
	r.handlers["GET"] = r.handleGet
	r.handlers["PUT"] = r.handlePut
	r.handlers["DELETE"] = r.handleDelete
	r.handlers["MKCOL"] = r.handleMkcol

	return r
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.logger.Info("received request",
		"method", req.Method,
		"path", req.URL.Path,
		"remote_addr", req.RemoteAddr)

	handler, ok := r.handlers[req.Method]
	if !ok {
		r.logger.Warn("method not allowed",
			"method", req.Method,
			"path", req.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handler(w, req)
}

// StripPrefix removes the baseURI prefix from the path
func StripPrefix(path, baseURI string) string {
	return strings.TrimPrefix(path, baseURI)
}
