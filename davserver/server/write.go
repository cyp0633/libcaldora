package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/emersion/go-ical"
)

// HandlePut processes PUT requests
func (s *Server) HandlePut(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling PUT request", "path", r.URL.Path)

	resourcePath := s.stripPrefix(r.URL.Path)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read request body",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
		return
	}

	// Parse iCalendar data
	dec := ical.NewDecoder(bytes.NewReader(body))
	cal, err := dec.Decode()
	if err != nil {
		s.logger.Error("failed to parse iCalendar data",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Invalid iCalendar data", Err: err})
		return
	}

	calObj := &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        resourcePath,
			Type:        interfaces.ResourceTypeCalendarObject,
			ContentType: r.Header.Get("Content-Type"),
		},
		Data: cal,
	}

	if err := s.config.Provider.PutCalendarObject(r.Context(), resourcePath, calObj); err != nil {
		s.logger.Error("failed to store calendar object",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to store calendar object", Err: err})
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.logger.Info("created calendar object",
		"path", resourcePath)
}

// HandleDelete processes DELETE requests
func (s *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling DELETE request", "path", r.URL.Path)

	resourcePath := s.stripPrefix(r.URL.Path)
	if err := s.config.Provider.DeleteCalendarObject(r.Context(), resourcePath); err != nil {
		s.logger.Error("failed to delete calendar object",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to delete calendar object", Err: err})
		return
	}

	w.WriteHeader(http.StatusNoContent)
	s.logger.Info("deleted calendar object",
		"path", resourcePath)
}
