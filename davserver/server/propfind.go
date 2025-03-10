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

	// Parse Depth header (0 or 1)
	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "0" // Default to 0 if not specified
	}

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

	// Build response for the requested resource
	responses := []protocol.Response{*s.buildPropfindResponse(r.URL.Path, props, &propfind)}

	// If Depth=1 and it's a collection, get child resources
	if depth == "1" && (props.Type == interfaces.ResourceTypeCollection || props.Type == interfaces.ResourceTypeCalendar) {
		if listable, ok := s.config.Provider.(interfaces.ListableProvider); ok {
			children, err := listable.ListResources(r.Context(), resourcePath)
			if err == nil { // Ignore errors, just don't include children
				for _, child := range children {
					responses = append(responses, *s.buildPropfindResponse(
						r.URL.Path+child.Path,
						child,
						&propfind,
					))
				}
			}
		}
	}

	// Create multistatus response with namespaces
	ms := &protocol.MultistatusResponse{
		XMLName: xml.Name{
			Space: "DAV:",
			Local: "multistatus",
		},
		Response: responses,
	}

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

func (s *Server) buildPropfindResponse(href string, props *interfaces.ResourceProperties, propfind *protocol.PropfindRequest) *protocol.Response {
	// Initialize empty PropertySet
	propSet := protocol.PropertySet{}

	// If allprop is requested or no specific props are requested, include all properties
	if propfind == nil || propfind.AllProp != nil || (propfind.Props == nil && propfind.PropName == nil) {
		propSet = protocol.PropertySet{
			ResourceType:  &protocol.ResourceType{},
			DisplayName:   props.DisplayName,
			CalendarColor: props.Color,
			GetCTag:       props.CTag,
			GetETag:       props.ETag,
		}
	} else if propfind.Props != nil {
		// Only include requested properties
		if propfind.Props.ResourceType != nil {
			propSet.ResourceType = &protocol.ResourceType{}
		}
		if propfind.Props.DisplayName != nil {
			propSet.DisplayName = props.DisplayName
		}
		if propfind.Props.CalendarColor != nil {
			propSet.CalendarColor = props.Color
		}
		if propfind.Props.GetCTag != nil {
			propSet.GetCTag = props.CTag
		}
		if propfind.Props.GetETag != nil {
			propSet.GetETag = props.ETag
		}
	}

	// Add CurrentUserPrivSet if present
	if len(props.CurrentUserPrivSet) > 0 {
		privSet := &protocol.CurrentUserPrivSet{
			Privilege: make([]protocol.Privilege, 0),
		}
		for _, priv := range props.CurrentUserPrivSet {
			switch priv {
			case "read":
				privSet.Privilege = append(privSet.Privilege, protocol.Privilege{Read: &xml.Name{Space: "DAV:", Local: "read"}})
			case "write":
				privSet.Privilege = append(privSet.Privilege, protocol.Privilege{Write: &xml.Name{Space: "DAV:", Local: "write"}})
			case "write-content":
				privSet.Privilege = append(privSet.Privilege, protocol.Privilege{WriteContent: &xml.Name{Space: "DAV:", Local: "write-content"}})
			}
		}
		propSet.CurrentUserPrivSet = privSet
	}

	// Set resource type
	if props.Type == interfaces.ResourceTypeCalendar {
		propSet.ResourceType.Collection = &xml.Name{Space: "DAV:", Local: "collection"}
		propSet.ResourceType.Calendar = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar"}
	} else if props.Type == interfaces.ResourceTypeCollection {
		propSet.ResourceType.Collection = &xml.Name{Space: "DAV:", Local: "collection"}
	}

	// Add CurrentUserPrincipal if present
	if props.PrincipalURL != "" {
		propSet.CurrentUserPrincipal = &protocol.CurrentUserPrincipal{
			Href: props.PrincipalURL,
		}
	}

	// Add CalendarHomeSet if present
	if props.CalendarHomeURL != "" {
		props.CalendarHomeSet = &interfaces.CalendarHome{
			Href: props.CalendarHomeURL,
		}
		propSet.CalendarHomeSet = &protocol.CalendarHomeSet{
			Href: props.CalendarHomeURL,
		}
	}

	return &protocol.Response{
		Href: href,
		Propstat: &protocol.Propstat{
			Prop:   propSet,
			Status: "HTTP/1.1 200 OK",
		},
	}
}
