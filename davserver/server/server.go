package server

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
)

// Server provides a CalDAV server implementation
type Server struct {
	config interfaces.HandlerConfig
	logger *slog.Logger
}

// New creates a new Server with the given configuration
func New(config interfaces.HandlerConfig) *Server {
	// Set default logger if none provided
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// Normalize URL prefix
	if config.URLPrefix == "" {
		config.URLPrefix = "/"
	}
	if !strings.HasPrefix(config.URLPrefix, "/") {
		config.URLPrefix = "/" + config.URLPrefix
	}
	if !strings.HasSuffix(config.URLPrefix, "/") {
		config.URLPrefix = config.URLPrefix + "/"
	}

	return &Server{
		config: config,
		logger: config.Logger,
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("received request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)

	// Add standard CalDAV headers
	w.Header().Set("DAV", "1, 3, calendar-access")
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, POST, PUT, DELETE, PROPFIND, PROPPATCH, REPORT")

	// Add custom headers if configured
	for k, v := range s.config.CustomHeaders {
		w.Header().Set(k, v)
	}

	// Check if method is allowed
	if s.config.AllowedMethods != nil {
		allowed := false
		for _, m := range s.config.AllowedMethods {
			if r.Method == m {
				allowed = true
				break
			}
		}
		if !allowed {
			s.logger.Warn("method not allowed",
				"method", r.Method,
				"path", r.URL.Path)
			s.sendError(w, interfaces.ErrMethodNotAllowed)
			return
		}
	}

	// Route request to appropriate handler
	switch r.Method {
	case "PROPFIND":
		s.HandlePropFind(w, r)
	case "REPORT":
		s.HandleReport(w, r)
	case "GET":
		s.HandleGet(w, r)
	case "PUT":
		s.HandlePut(w, r)
	case "DELETE":
		s.HandleDelete(w, r)
	case "OPTIONS":
		s.HandleOptions(w, r)
	case "MKCOL":
		s.HandleMkCol(w, r)
	default:
		s.logger.Warn("unsupported method",
			"method", r.Method,
			"path", r.URL.Path)
		s.sendError(w, interfaces.ErrMethodNotAllowed)
	}
}

// Helper functions

func (s *Server) stripPrefix(urlPath string) string {
	// Remove URL prefix and make sure the path doesn't start with a slash
	return strings.TrimPrefix(strings.TrimPrefix(urlPath, s.config.URLPrefix), "/")
}
