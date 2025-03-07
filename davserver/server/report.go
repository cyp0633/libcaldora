package server

import (
	"encoding/xml"
	"io"
	"net/http"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/internal/protocol"
)

// HandleReport processes REPORT requests
func (s *Server) HandleReport(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling REPORT request", "path", r.URL.Path)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read request body",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
		return
	}

	// Determine report type from the XML
	var reportType struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(body, &reportType); err != nil {
		s.logger.Error("failed to parse XML request",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse XML request", Err: err})
		return
	}

	s.logger.Debug("processing report",
		"type", reportType.XMLName.Local,
		"path", r.URL.Path)

	switch reportType.XMLName.Local {
	case "calendar-query":
		s.handleCalendarQuery(w, r, body)
	case "calendar-multiget":
		s.handleCalendarMultiget(w, r, body)
	default:
		s.logger.Warn("unsupported report type",
			"type", reportType.XMLName.Local,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Unsupported report type"})
	}
}

// handleCalendarQuery processes calendar-query REPORT requests
func (s *Server) handleCalendarQuery(w http.ResponseWriter, r *http.Request, body []byte) {
	var query protocol.CalendarQueryReport
	if err := xml.Unmarshal(body, &query); err != nil {
		s.logger.Error("failed to parse calendar-query",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse calendar-query", Err: err})
		return
	}

	calendarPath := s.stripPrefix(r.URL.Path)
	filter := &interfaces.QueryFilter{
		CompFilter: query.Filter.CompFilter.Name,
	}

	s.logger.Debug("executing calendar query",
		"path", calendarPath,
		"filter", filter)

	objects, err := s.config.Provider.Query(r.Context(), calendarPath, filter)
	if err != nil {
		s.logger.Error("failed to execute query",
			"error", err,
			"path", calendarPath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to query calendar", Err: err})
		return
	}

	// Build response
	responses := make([]protocol.Response, len(objects))
	for i, obj := range objects {
		responses[i] = *s.buildPropfindResponse(obj.Properties.Path, obj.Properties)
	}

	ms := protocol.NewMultistatusResponse(responses...)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		s.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	s.logger.Debug("completed calendar query",
		"path", r.URL.Path,
		"results", len(objects))
}

// handleCalendarMultiget processes calendar-multiget REPORT requests
func (s *Server) handleCalendarMultiget(w http.ResponseWriter, r *http.Request, body []byte) {
	var multiget protocol.CalendarMultiGet
	if err := xml.Unmarshal(body, &multiget); err != nil {
		s.logger.Error("failed to parse calendar-multiget",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse calendar-multiget", Err: err})
		return
	}

	s.logger.Debug("executing calendar multiget",
		"path", r.URL.Path,
		"urls", len(multiget.Href))

	objects, err := s.config.Provider.MultiGet(r.Context(), multiget.Href)
	if err != nil {
		s.logger.Error("failed to execute multiget",
			"error", err,
			"path", r.URL.Path)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to fetch calendar objects", Err: err})
		return
	}

	// Build response
	responses := make([]protocol.Response, len(objects))
	for i, obj := range objects {
		responses[i] = *s.buildPropfindResponse(obj.Properties.Path, obj.Properties)
	}

	ms := protocol.NewMultistatusResponse(responses...)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		s.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	s.logger.Debug("completed calendar multiget",
		"path", r.URL.Path,
		"results", len(objects))
}
