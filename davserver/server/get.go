package server

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/emersion/go-ical"
)

// HandleGet processes GET requests
func (s *Server) HandleGet(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling GET request", "path", r.URL.Path)

	resourcePath := s.stripPrefix(r.URL.Path)
	obj, err := s.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil {
		s.logger.Error("failed to get calendar object",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found", Err: err})
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	if obj.Properties.ETag != "" {
		w.Header().Set("ETag", obj.Properties.ETag)
	}
	w.WriteHeader(http.StatusOK)

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(obj.Data); err != nil {
		s.logger.Error("failed to encode calendar data",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to encode calendar data", Err: err})
		return
	}
	fmt.Fprintf(w, "%s", buf.String())

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
