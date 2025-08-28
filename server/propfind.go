package server

import (
	"io"
	"net/http"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/server/storage"
)

// Update the handlePropfind function to use MergeResponses
func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	// fetch all requested resources as Depth header
	initialResource := ctx.Resource
	children, err := h.fetchChildren(ctx.Depth, initialResource)
	if err != nil {
		h.Logger.Error("failed to fetch children for resource",
			"resource", initialResource,
			"error", err)
		http.Error(w, "Failed to fetch children", http.StatusInternalServerError)
		return
	}
	resources := append([]Resource{initialResource}, children...)

	// parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read request body",
			"error", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	req, _ := propfind.ParseRequest(string(bodyBytes))
	// TODO: PropName handling

	var docs []*etree.Document
	for _, resource := range resources {
		ctx1 := *ctx             // Create a copy of the context
		ctx1.Resource = resource // Update the context for the individual resource

		var doc *etree.Document
		var err error

		switch resource.ResourceType {
		case storage.ResourcePrincipal:
			doc, err = h.handlePropfindPrincipal(req, ctx1.Resource)
		case storage.ResourceHomeSet:
			doc, err = h.handlePropfindHomeSet(req, ctx1.Resource)
		case storage.ResourceCollection:
			doc, err = h.handlePropfindCollection(req, ctx1.Resource)
		case storage.ResourceObject:
			doc, err = h.handlePropfindObject(req, ctx1.Resource)
		case storage.ResourceServiceRoot:
			ctx1.Resource.UserID = ctx.AuthUser // Just a workaround
			doc, err = h.handlePropfindServiceRoot(req, ctx1.Resource)
		default:
			h.Logger.Error("unknown resource type",
				"type", resource.ResourceType,
				"resource", resource)
			http.Error(w, "Unknown resource type", http.StatusNotFound)
			return
		}

		if err != nil {
			h.Logger.Error("error handling PROPFIND",
				"resource_type", resource.ResourceType,
				"resource", resource,
				"error", err)
			continue // Skip this resource but continue with others
		}

		if doc != nil {
			docs = append(docs, doc)
		}
	}

	// Merge all responses
	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		h.Logger.Error("failed to merge PROPFIND responses",
			"error", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
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

	w.Write([]byte(xmlOutput))
}

// handles individual home set request
func (h *CaldavHandler) handlePropfindHomeSet(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	req = h.resolvePropfind(req, res, nil)
	return propfind.EncodeResponse(req, path), nil
}

// handles user principal request
func (h *CaldavHandler) handlePropfindPrincipal(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	req = h.resolvePropfind(req, res, nil)
	return propfind.EncodeResponse(req, path), nil
}

// handlePropfindObject is a wrapper that first fetches the object, then calls the inner function
func (h *CaldavHandler) handlePropfindObject(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	if res.URI == "" {
		path, err := h.URLConverter.EncodePath(res)
		if err != nil {
			h.Logger.Error("failed to encode path for resource",
				"resource", res,
				"error", err)
			return nil, err
		}
		res.URI = path
	}

	object, err := h.Storage.GetObject(res.UserID, res.CalendarID, res.ObjectID)
	if err != nil {
		h.Logger.Error("failed to get object for resource",
			"resource", res,
			"error", err)
		return nil, err
	}
	if object == nil || len(object.Component) == 0 {
		h.Logger.Error("no object found for resource",
			"resource", res)
		return nil, propfind.ErrNotFound
	}

	return h.handlePropfindObjectWithObject(req, res, *object)
}

// handlePropfindObjectWithObject processes a PROPFIND request for a calendar object
// when the calendar object has already been fetched
func (h *CaldavHandler) handlePropfindObjectWithObject(req propfind.ResponseMap, res Resource, object storage.CalendarObject) (*etree.Document, error) {
	// Use resolver with preloaded object
	req = h.resolvePropfind(req, res, &object)
	return propfind.EncodeResponse(req, res.URI), nil
}

func (h *CaldavHandler) handlePropfindCollection(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	h.Logger.Debug("handling PROPFIND for collection",
		"path", path,
		"user_id", res.UserID,
		"calendar_id", res.CalendarID,
		"resource_type", res.ResourceType)

	// Resolve via resolvers
	req = h.resolvePropfind(req, res, nil)
	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) handlePropfindServiceRoot(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}
	req = h.resolvePropfind(req, res, nil)
	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) fetchChildren(depth int, parent Resource) (resources []Resource, err error) {
	if depth <= 0 {
		return
	}

	h.Logger.Debug("fetching children",
		"depth", depth,
		"parent_type", parent.ResourceType,
		"user_id", parent.UserID,
		"calendar_id", parent.CalendarID)

	switch parent.ResourceType {
	case storage.ResourceObject, storage.ResourcePrincipal:
		// These types don't have children, return empty slice
		h.Logger.Debug("resource type has no children",
			"resource_type", parent.ResourceType)
		return []Resource{}, nil

	case storage.ResourceCollection:
		// find object (event) paths in the collection
		paths, err := h.Storage.GetObjectPathsInCollection(parent.CalendarID)
		if err != nil {
			h.Logger.Error("failed to fetch event paths in collection",
				"calendar_id", parent.CalendarID,
				"error", err)
			return nil, err
		}

		h.Logger.Debug("found event paths in collection",
			"calendar_id", parent.CalendarID,
			"path_count", len(paths))

		for _, path := range paths {
			h.Logger.Debug("parsing event path",
				"path", path,
				"calendar_id", parent.CalendarID)

			resource, err := h.URLConverter.ParsePath(path)
			if err != nil {
				h.Logger.Error("failed to parse path",
					"path", path,
					"error", err)
				return nil, err
			}

			h.Logger.Debug("parsed event resource",
				"path", path,
				"resource_type", resource.ResourceType,
				"object_id", resource.ObjectID)

			resources = append(resources, resource)
			children, err := h.fetchChildren(depth-1, resource) // Recursively fetch children for the object
			if err != nil {
				h.Logger.Error("failed to fetch children for resource",
					"resource", resource,
					"error", err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	case storage.ResourceHomeSet:
		// find collections in the home set
		calendars, err := h.Storage.GetUserCalendars(parent.UserID)
		if err != nil {
			h.Logger.Error("failed to fetch calendars for user",
				"user_id", parent.UserID,
				"error", err)
			return nil, err
		}
		for _, cal := range calendars {
			resource, err := h.URLConverter.ParsePath(cal.Path)
			if err != nil {
				h.Logger.Error("failed to parse calendar path",
					"path", cal.Path,
					"error", err)
				return nil, err
			}
			resources = append(resources, resource)
			// Recursively fetch children for the collection
			children, err := h.fetchChildren(depth-1, resource)
			if err != nil {
				h.Logger.Error("failed to fetch children for resource",
					"resource", resource,
					"error", err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	}
	return
}
