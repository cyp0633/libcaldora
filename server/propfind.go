package caldav

import (
	"log"
	"net/http"
)

func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PROPFIND received for %s (User: %s, Calendar: %s, Object: %s)", ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	initialResource := ctx.Resource
	_, _ = h.fetchChildren(ctx.Depth, initialResource)
	// TODO: Implement PROPFIND logic based on ctx.ResourceType and request body/headers (Depth)
	http.Error(w, "Not Implemented: PROPFIND", http.StatusNotImplemented)
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
