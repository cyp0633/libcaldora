package handler

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	davxml "github.com/cyp0633/libcaldora/davserver/protocol/xml"
	"github.com/emersion/go-ical"
)

// DefaultHandler provides a basic implementation of the CalDAV server handler
type DefaultHandler struct {
	config interfaces.HandlerConfig
	logger *slog.Logger
}

// NewDefaultHandler creates a new DefaultHandler with the given configuration
func NewDefaultHandler(config interfaces.HandlerConfig) *DefaultHandler {
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

	return &DefaultHandler{
		config: config,
		logger: config.Logger,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("received request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)

	// Add standard CalDAV headers
	w.Header().Set("DAV", "1, 3, calendar-access")
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, POST, PUT, DELETE, PROPFIND, PROPPATCH, REPORT")

	// Add custom headers if configured
	for k, v := range h.config.CustomHeaders {
		w.Header().Set(k, v)
	}

	// Check if method is allowed
	if h.config.AllowedMethods != nil {
		allowed := false
		for _, m := range h.config.AllowedMethods {
			if r.Method == m {
				allowed = true
				break
			}
		}
		if !allowed {
			h.logger.Warn("method not allowed",
				"method", r.Method,
				"path", r.URL.Path)
			h.sendError(w, interfaces.ErrMethodNotAllowed)
			return
		}
	}

	// Route request to appropriate handler
	switch r.Method {
	case "PROPFIND":
		h.HandlePropFind(w, r)
	case "REPORT":
		h.HandleReport(w, r)
	case "GET":
		h.HandleGet(w, r)
	case "PUT":
		h.HandlePut(w, r)
	case "DELETE":
		h.HandleDelete(w, r)
	case "OPTIONS":
		h.HandleOptions(w, r)
	case "MKCOL":
		h.HandleMkCol(w, r)
	default:
		h.logger.Warn("unsupported method",
			"method", r.Method,
			"path", r.URL.Path)
		h.sendError(w, interfaces.ErrMethodNotAllowed)
	}
}

// HandlePropFind processes PROPFIND requests
func (h *DefaultHandler) HandlePropFind(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling PROPFIND request", "path", r.URL.Path)

	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
		return
	}

	var propfind davxml.PropfindRequest
	if err := xml.Unmarshal(body, &propfind); err != nil {
		h.logger.Error("failed to parse XML request",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse XML request", Err: err})
		return
	}

	// Get resource properties
	resourcePath := h.stripPrefix(r.URL.Path)
	props, err := h.config.Provider.GetResourceProperties(r.Context(), resourcePath)
	if err != nil {
		h.logger.Error("failed to get resource properties",
			"error", err,
			"path", resourcePath)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found", Err: err})
		return
	}

	// Build response
	response := h.buildPropfindResponse(r.URL.Path, props)
	ms := davxml.NewMultistatusResponse(*response)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	h.logger.Debug("completed PROPFIND request",
		"path", r.URL.Path,
		"status", http.StatusMultiStatus)
}

// HandleReport processes REPORT requests
func (h *DefaultHandler) HandleReport(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling REPORT request", "path", r.URL.Path)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
		return
	}

	// Determine report type from the XML
	var reportType struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(body, &reportType); err != nil {
		h.logger.Error("failed to parse XML request",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse XML request", Err: err})
		return
	}

	h.logger.Debug("processing report",
		"type", reportType.XMLName.Local,
		"path", r.URL.Path)

	switch reportType.XMLName.Local {
	case "calendar-query":
		h.handleCalendarQuery(w, r, body)
	case "calendar-multiget":
		h.handleCalendarMultiget(w, r, body)
	default:
		h.logger.Warn("unsupported report type",
			"type", reportType.XMLName.Local,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Unsupported report type"})
	}
}

// handleCalendarQuery processes calendar-query REPORT requests
func (h *DefaultHandler) handleCalendarQuery(w http.ResponseWriter, r *http.Request, body []byte) {
	var query davxml.CalendarQueryReport
	if err := xml.Unmarshal(body, &query); err != nil {
		h.logger.Error("failed to parse calendar-query",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse calendar-query", Err: err})
		return
	}

	calendarPath := h.stripPrefix(r.URL.Path)
	filter := &interfaces.QueryFilter{
		CompFilter: query.Filter.CompFilter.Name,
	}

	h.logger.Debug("executing calendar query",
		"path", calendarPath,
		"filter", filter)

	objects, err := h.config.Provider.Query(r.Context(), calendarPath, filter)
	if err != nil {
		h.logger.Error("failed to execute query",
			"error", err,
			"path", calendarPath)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to query calendar", Err: err})
		return
	}

	// Build response
	responses := make([]davxml.Response, len(objects))
	for i, obj := range objects {
		responses[i] = *h.buildPropfindResponse(obj.Properties.Path, obj.Properties)
	}

	ms := davxml.NewMultistatusResponse(responses...)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	h.logger.Debug("completed calendar query",
		"path", r.URL.Path,
		"results", len(objects))
}

// handleCalendarMultiget processes calendar-multiget REPORT requests
func (h *DefaultHandler) handleCalendarMultiget(w http.ResponseWriter, r *http.Request, body []byte) {
	var multiget davxml.CalendarMultiGet
	if err := xml.Unmarshal(body, &multiget); err != nil {
		h.logger.Error("failed to parse calendar-multiget",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to parse calendar-multiget", Err: err})
		return
	}

	h.logger.Debug("executing calendar multiget",
		"path", r.URL.Path,
		"urls", len(multiget.Href))

	objects, err := h.config.Provider.MultiGet(r.Context(), multiget.Href)
	if err != nil {
		h.logger.Error("failed to execute multiget",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to fetch calendar objects", Err: err})
		return
	}

	// Build response
	responses := make([]davxml.Response, len(objects))
	for i, obj := range objects {
		responses[i] = *h.buildPropfindResponse(obj.Properties.Path, obj.Properties)
	}

	ms := davxml.NewMultistatusResponse(responses...)

	// Send response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	enc := xml.NewEncoder(w)
	if err := enc.Encode(ms); err != nil {
		h.logger.Error("failed to encode response",
			"error", err,
			"path", r.URL.Path)
	}

	h.logger.Debug("completed calendar multiget",
		"path", r.URL.Path,
		"results", len(objects))
}

// HandleGet processes GET requests
func (h *DefaultHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling GET request", "path", r.URL.Path)

	resourcePath := h.stripPrefix(r.URL.Path)
	obj, err := h.config.Provider.GetCalendarObject(r.Context(), resourcePath)
	if err != nil {
		h.logger.Error("failed to get calendar object",
			"error", err,
			"path", resourcePath)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusNotFound, Message: "Resource not found", Err: err})
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
		h.logger.Error("failed to encode calendar data",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to encode calendar data", Err: err})
		return
	}
	fmt.Fprintf(w, "%s", buf.String())

	h.logger.Debug("completed GET request",
		"path", r.URL.Path,
		"status", http.StatusOK)
}

// HandlePut processes PUT requests
func (h *DefaultHandler) HandlePut(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling PUT request", "path", r.URL.Path)

	resourcePath := h.stripPrefix(r.URL.Path)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Failed to read request body", Err: err})
		return
	}

	// Parse iCalendar data
	dec := ical.NewDecoder(bytes.NewReader(body))
	cal, err := dec.Decode()
	if err != nil {
		h.logger.Error("failed to parse iCalendar data",
			"error", err,
			"path", r.URL.Path)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusBadRequest, Message: "Invalid iCalendar data", Err: err})
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

	if err := h.config.Provider.PutCalendarObject(r.Context(), resourcePath, calObj); err != nil {
		h.logger.Error("failed to store calendar object",
			"error", err,
			"path", resourcePath)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to store calendar object", Err: err})
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.logger.Info("created calendar object",
		"path", resourcePath)
}

// HandleDelete processes DELETE requests
func (h *DefaultHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling DELETE request", "path", r.URL.Path)

	resourcePath := h.stripPrefix(r.URL.Path)
	if err := h.config.Provider.DeleteCalendarObject(r.Context(), resourcePath); err != nil {
		h.logger.Error("failed to delete calendar object",
			"error", err,
			"path", resourcePath)
		h.sendError(w, &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: "Failed to delete calendar object", Err: err})
		return
	}

	w.WriteHeader(http.StatusNoContent)
	h.logger.Info("deleted calendar object",
		"path", resourcePath)
}

// HandleOptions processes OPTIONS requests
func (h *DefaultHandler) HandleOptions(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling OPTIONS request", "path", r.URL.Path)
	// Headers are already set in ServeHTTP
	w.WriteHeader(http.StatusOK)
}

// HandleMkCol processes MKCOL requests
func (h *DefaultHandler) HandleMkCol(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling MKCOL request", "path", r.URL.Path)
	h.sendError(w, &interfaces.HTTPError{Status: http.StatusMethodNotAllowed, Message: "MKCOL not implemented"})
}

// Helper functions

func (h *DefaultHandler) stripPrefix(urlPath string) string {
	return strings.TrimPrefix(urlPath, h.config.URLPrefix)
}

func (h *DefaultHandler) sendError(w http.ResponseWriter, err error) {
	var httpErr *interfaces.HTTPError
	if !errors.As(err, &httpErr) {
		httpErr = &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	h.logger.Error("error response",
		"status", httpErr.Status,
		"message", httpErr.Message,
		"error", httpErr.Err)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(httpErr.Status)

	errResp := &interfaces.ErrorResponse{
		Namespace: "DAV:",
		Message:   httpErr.Message,
	}

	enc := xml.NewEncoder(w)
	if err := enc.Encode(errResp); err != nil {
		h.logger.Error("failed to encode error response",
			"error", err)
	}
}

func (h *DefaultHandler) buildPropfindResponse(href string, props *interfaces.ResourceProperties) *davxml.Response {
	// Convert ResourceProperties to PropertySet
	propSet := davxml.PropertySet{
		ResourceType:  &davxml.ResourceType{},
		DisplayName:   props.DisplayName,
		CalendarColor: props.Color,
		GetCTag:       props.CTag,
		GetETag:       props.ETag,
	}

	if props.Type == interfaces.ResourceTypeCalendar {
		propSet.ResourceType.Collection = &xml.Name{Space: "DAV:", Local: "collection"}
		propSet.ResourceType.Calendar = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar"}
	}

	return &davxml.Response{
		Href: href,
		Propstat: &davxml.Propstat{
			Prop:   propSet,
			Status: "HTTP/1.1 200 OK",
		},
	}
}
