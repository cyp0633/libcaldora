package caldav

import (
	"io"
	"log"
	"net/http"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
)

// Update the handlePropfind function to use MergeResponses
func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	// fetch all requested resources as Depth header
	initialResource := ctx.Resource
	children, err := h.fetchChildren(ctx.Depth, initialResource)
	if err != nil {
		log.Printf("Failed to fetch children for resource %s: %v", initialResource, err)
		http.Error(w, "Failed to fetch children", http.StatusInternalServerError)
		return
	}
	resources := append([]Resource{initialResource}, children...)

	// parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
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
		case ResourcePrincipal:
			doc, err = h.handlePropfindPrincipal(req, &ctx1)
		case ResourceHomeSet:
			doc, err = h.handlePropfindHomeSet(req, &ctx1)
		case ResourceCollection:
			doc, err = h.handlePropfindCollection(req, &ctx1)
		case ResourceObject:
			doc, err = h.handlePropfindObject(req, &ctx1)
		}

		if err != nil {
			log.Printf("Error handling PROPFIND for resource %v: %v", resource, err)
			continue // Skip this resource but continue with others
		}

		if doc != nil {
			docs = append(docs, doc)
		}
	}

	// Merge all responses
	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		log.Printf("Failed to merge PROPFIND responses: %v", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status

	// Serialize and write the XML document
	xmlOutput, err := mergedDoc.WriteToString()
	if err != nil {
		log.Printf("Failed to serialize XML response: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xmlOutput))
}

// handles individual home set request
func (h *CaldavHandler) handlePropfindHomeSet(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

// handles individual resource
func (h *CaldavHandler) handlePropfindPrincipal(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

// handles individual resource
func (h *CaldavHandler) handlePropfindObject(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) handlePropfindCollection(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) fetchChildren(depth int, parent Resource) (resources []Resource, err error) {
	if depth <= 0 {
		return
	}
	switch parent.ResourceType {
	case ResourceObject:
		// object does not have children, return
		return
	case ResourcePrincipal:
		// no nested resources for principals, return empty
		return
	case ResourceCollection:
		// find object (event) paths in the collection
		paths, err := h.Storage.GetObjectPathsInCollection(parent.CalendarID)
		if err != nil {
			log.Printf("Failed to fetch event paths in collection %s: %v", parent.CalendarID, err)
			return nil, err
		}
		for _, path := range paths {
			resource, err := h.URLConverter.ParsePath(path)
			if err != nil {
				log.Printf("Failed to parse path %s: %v", path, err)
				return nil, err
			}
			resources = append(resources, resource)
			children, err := h.fetchChildren(depth-1, resource) // Recursively fetch children for the object
			if err != nil {
				log.Printf("Failed to fetch children for resource %s: %v", resource, err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	case ResourceHomeSet:
		// find collections in the home set
		calendars, err := h.Storage.GetUserCalendars(parent.UserID)
		if err != nil {
			log.Printf("Failed to fetch calendars for user %s: %v", parent.UserID, err)
			return nil, err
		}
		for _, cal := range calendars {
			resource, err := h.URLConverter.ParsePath(cal.Path)
			if err != nil {
				log.Printf("Failed to parse calendar path %s: %v", cal.Path, err)
				return nil, err
			}
			resources = append(resources, resource)
			// Recursively fetch children for the collection
			children, err := h.fetchChildren(depth-1, resource)
			if err != nil {
				log.Printf("Failed to fetch children for resource %s: %v", resource, err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	}
	return
}
