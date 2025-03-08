package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/emersion/go-ical"
)

// HandleGet processes GET requests
func (s *Server) HandleGet(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling GET request", "path", r.URL.Path)

	resourcePath := s.stripPrefix(r.URL.Path)
	obj, err := s.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil {
		if err == interfaces.ErrNotFound {
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found"})
		} else {
			s.logger.Error("failed to get calendar object",
				"error", err,
				"path", resourcePath)
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to get calendar object", Err: err})
		}
		return
	}

	// Check If-None-Match for caching
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" && obj.Properties.ETag != "" {
		if ifNoneMatch == "*" || strings.Contains(ifNoneMatch, obj.Properties.ETag) {
			w.Header().Set("ETag", obj.Properties.ETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(obj.Data); err != nil {
		s.logger.Error("failed to encode calendar data",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to encode calendar data", Err: err})
		return
	}

	body := buf.Bytes()
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("ETag", obj.Properties.ETag)
	// Add cache control headers
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)

	s.logger.Debug("completed GET request",
		"path", r.URL.Path,
		"status", http.StatusOK)
}

// HandleOptions processes OPTIONS requests
func (s *Server) HandleOptions(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling OPTIONS request", "path", r.URL.Path)
	// Headers are already set in ServeHTTP
	w.WriteHeader(http.StatusOK)
}

// HandleMkCol processes MKCOL requests
func (s *Server) HandleMkCol(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling MKCOL request", "path", r.URL.Path)
	s.sendError(w, &interfaces.HTTPError{Status: http.StatusMethodNotAllowed, Message: "MKCOL not implemented"})
}
