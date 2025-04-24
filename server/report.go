package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/beevik/etree"
	cmg "github.com/cyp0633/libcaldora/internal/xml/calendar-multiget"
	cq "github.com/cyp0633/libcaldora/internal/xml/calendar-query"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handleReport(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	h.Logger.Info("report request received",
		"resource_type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read report request body",
			"error", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse XML
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		h.Logger.Error("failed to parse report XML",
			"error", err)
		http.Error(w, "Error parsing XML request body", http.StatusBadRequest)
		return
	}

	// Get the root element
	root := doc.Root()
	if root == nil {
		h.Logger.Error("invalid XML: no root element")
		http.Error(w, "Invalid XML: no root element", http.StatusBadRequest)
		return
	}

	// Extract local name (removing namespace prefix if present)
	tagName := root.Tag
	if idx := strings.Index(tagName, ":"); idx != -1 {
		tagName = tagName[idx+1:]
	}

	h.Logger.Debug("report type identified",
		"tag", tagName)

	// Clone the request for handlers to re-read the body
	reqClone := r.Clone(r.Context())
	reqClone.Body = io.NopCloser(strings.NewReader(string(body)))

	// Route to appropriate handler based on report type
	switch tagName {
	case "calendar-multiget":
		h.handleCalendarMultiget(w, reqClone, ctx)
	case "calendar-query":
		h.handleCalendarQuery(w, reqClone, ctx)
	case "freebusy-query":
		h.handleFreebusyQuery(w, reqClone, ctx)
	case "schedule-query":
		h.handleScheduleQuery(w, reqClone, ctx)
	case "availability-query":
		h.handleAvailabilityQuery(w, reqClone, ctx)
	default:
		h.Logger.Warn("unsupported report type",
			"tag", tagName)
		http.Error(w, "Unsupported report type", http.StatusBadRequest)
	}
}

func (h *CaldavHandler) handleCalendarMultiget(w http.ResponseWriter, r *http.Request, _ *RequestContext) {
	// get resources and requested properties
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read request body",
			"error", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	bodyStr := string(bodyBytes)
	h.Logger.Info("calendar-multiget request body",
		"body", bodyStr)

	req, resourceLinks := cmg.ParseRequest(bodyStr)

	h.Logger.Info("parsed resource links from request",
		"count", len(resourceLinks))

	// use PROPFIND handler to get properties
	var docs []*etree.Document
	for _, resourceLink := range resourceLinks {
		h.Logger.Info("processing resource link",
			"link", resourceLink)
		resource, err := h.URLConverter.ParsePath(resourceLink)
		if err != nil {
			h.Logger.Error("error parsing path",
				"path", resourceLink,
				"error", err)
			http.Error(w, "Error retrieving resource", http.StatusInternalServerError)
			return
		}

		var doc *etree.Document
		switch resource.ResourceType {
		case storage.ResourceObject:
			doc, err = h.handlePropfindObject(req, resource)
		case storage.ResourceCollection:
			doc, err = h.handlePropfindCollection(req, resource)
		case storage.ResourceHomeSet:
			doc, err = h.handlePropfindHomeSet(req, resource)
		case storage.ResourcePrincipal:
			doc, err = h.handlePropfindPrincipal(req, resource)
		default:
			h.Logger.Error("unsupported resource type",
				"type", resource.ResourceType)
			http.Error(w, "Unsupported resource type", http.StatusBadRequest)
			return
		}

		if err != nil {
			h.Logger.Error("error handling propfind",
				"type", resource.ResourceType,
				"error", err)
			http.Error(w, "Error retrieving resource", http.StatusInternalServerError)
			return
		}
		docs = append(docs, doc)
	}

	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		h.Logger.Error("error merging responses",
			"error", err)
		http.Error(w, "Error merging responses", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status

	// Serialize and write the XML document
	xmlOutput, err := mergedDoc.WriteToString()
	if err != nil {
		h.Logger.Error("failed to serialize XML response",
			"error", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("sending calendar-multiget response",
		"response_size", len(xmlOutput))
	w.Write([]byte(xmlOutput))
}

func (h *CaldavHandler) handleCalendarQuery(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read request body",
			"error", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	bodyStr := string(bodyBytes)
	h.Logger.Info("calendar-query request body",
		"body", bodyStr)

	req, filter, err := cq.ParseRequest(bodyStr)
	if err != nil {
		h.Logger.Error("error parsing request",
			"error", err)
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	docs := []*etree.Document{}
	switch ctx.Resource.ResourceType {
	case storage.ResourceObject:
		object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
		if err != nil {
			h.Logger.Error("error getting object",
				"error", err)
			http.Error(w, "Error retrieving object", http.StatusInternalServerError)
			return
		}
		if !filter.Validate(object) {
			h.Logger.Warn("object does not match filter",
				"object_id", ctx.Resource.ObjectID)
			http.Error(w, "Object does not match filter", http.StatusNotFound)
			return
		}
		doc, err := h.handlePropfindObjectWithObject(req, ctx.Resource, *object)
		if err != nil {
			h.Logger.Error("error handling propfind for object",
				"error", err)
			http.Error(w, "Error retrieving object", http.StatusInternalServerError)
			return
		}
		docs = append(docs, doc)
	case storage.ResourceCollection:
		objects, err := h.Storage.GetObjectByFilter(ctx.Resource.UserID, ctx.Resource.CalendarID, filter)
		if err != nil {
			h.Logger.Error("error getting objects by filter",
				"error", err)
			http.Error(w, "Error retrieving objects", http.StatusInternalServerError)
			return
		}
		for _, object := range objects {
			doc, err := h.handlePropfindObjectWithObject(req, ctx.Resource, object)
			if err != nil {
				h.Logger.Error("error handling propfind for object",
					"error", err)
				http.Error(w, "Error retrieving object", http.StatusInternalServerError)
				return
			}
			docs = append(docs, doc)
		}
	default:
		// bad request, only collection & object
		h.Logger.Error("unsupported resource type for calendar-query",
			"type", ctx.Resource.ResourceType)
		http.Error(w, "Unsupported resource type for calendar-query", http.StatusBadRequest)
		return
	}

	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		h.Logger.Error("error merging responses",
			"error", err)
		http.Error(w, "Error merging responses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status

	xmlOutput, err := mergedDoc.WriteToString()
	if err != nil {
		h.Logger.Error("failed to serialize XML response",
			"error", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("sending calendar-query response",
		"response_size", len(xmlOutput))
	w.Write([]byte(xmlOutput))
}

func (h *CaldavHandler) handleFreebusyQuery(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
}

func (h *CaldavHandler) handleScheduleQuery(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
}

func (h *CaldavHandler) handleAvailabilityQuery(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
}
