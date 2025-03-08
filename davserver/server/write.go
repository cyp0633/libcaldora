package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/emersion/go-ical"
)

// checkPreconditions checks If-Match and If-None-Match headers
func (s *Server) checkPreconditions(r *http.Request, currentETag string) bool {
	ifMatch := r.Header.Get("If-Match")
	ifNoneMatch := r.Header.Get("If-None-Match")

	if ifMatch != "" {
		if ifMatch != "*" && !strings.Contains(ifMatch, currentETag) {
			return false
		}
	}

	if ifNoneMatch != "" {
		if ifNoneMatch == "*" || strings.Contains(ifNoneMatch, currentETag) {
			return false
		}
	}

	return true
}

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

	// Check if resource exists
	existingObj, err := s.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil && err != interfaces.ErrNotFound {
		s.logger.Error("failed to check existing object",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to check existing object", Err: err})
		return
	}

	// If resource exists, check preconditions
	if existingObj != nil {
		if !s.checkPreconditions(r, existingObj.Properties.ETag) {
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusPreconditionFailed, Message: "ETag precondition failed"})
			return
		}
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

	status := http.StatusCreated
	if existingObj != nil {
		status = http.StatusNoContent
	}

	// Get the stored object to get its ETag
	storedObj, err := s.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil {
		s.logger.Error("failed to get stored object",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to get stored object", Err: err})
		return
	}

	w.Header().Set("ETag", storedObj.Properties.ETag)
	w.WriteHeader(status)
	s.logger.Info("saved calendar object",
		"path", resourcePath,
		"etag", storedObj.Properties.ETag)
}

// HandleDelete processes DELETE requests
func (s *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling DELETE request", "path", r.URL.Path)

	resourcePath := s.stripPrefix(r.URL.Path)

	// Get existing object to check ETag
	existingObj, err := s.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil {
		if err == interfaces.ErrNotFound {
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found"})
		} else {
			s.logger.Error("failed to get existing object",
				"error", err,
				"path", resourcePath)
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to get calendar object", Err: err})
		}
		return
	}

	// Check preconditions
	if !s.checkPreconditions(r, existingObj.Properties.ETag) {
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusPreconditionFailed, Message: "ETag precondition failed"})
		return
	}

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
