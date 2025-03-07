package server

import (
	"encoding/xml"
	"io"
	"net/http"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/internal/protocol"
)

// HandlePropFind processes PROPFIND requests
func (s *Server) HandlePropFind(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling PROPFIND request", "path", r.URL.Path)

	var propfind protocol.PropfindRequest

	// Try to parse request body if present
	if r.ContentLength > 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error("failed to read request body",
				"error", err,
				"path", r.URL.Path)
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
			return
		}

		if err := xml.Unmarshal(body, &propfind); err != nil {
			s.logger.Error("failed to parse XML request",
				"error", err,
				"path", r.URL.Path)
			s.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse XML request", Err: err})
			return
		}
	}
	// Empty body means allprop

	// Get resource properties
	resourcePath := s.stripPrefix(r.URL.Path)
	props, err := s.config.Provider.GetResourceProperties(r.Context(), resourcePath)
	if err != nil {
		s.logger.Error("failed to get resource properties",
			"error", err,
			"path", resourcePath)
		s.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found", Err: err})
		return
	}

	// Build response
	response := s.buildPropfindResponse(r.URL.Path, props)
	ms := protocol.NewMultistatusResponse([]protocol.Response{*response}...)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		s.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	s.logger.Debug("completed PROPFIND request",
		"path", r.URL.Path,
		"status", http.StatusMultiStatus)
}

func (s *Server) buildPropfindResponse(href string, props *interfaces.ResourceProperties) *protocol.Response {
	// Convert ResourceProperties to PropertySet
	propSet := protocol.PropertySet{
		ResourceType:  &protocol.ResourceType{},
		DisplayName:   props.DisplayName,
		CalendarColor: props.Color,
		GetCTag:       props.CTag,
		GetETag:       props.ETag,
	}

	if props.Type == interfaces.ResourceTypeCalendar {
		propSet.ResourceType.Collection = &xml.Name{Space: "DAV:", Local: "collection"}
		propSet.ResourceType.Calendar = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar"}
	}

	return &protocol.Response{
		Href: href,
		Propstat: &protocol.Propstat{
			Prop:   propSet,
			Status: "HTTP/1.1 200 OK",
		},
	}
}
