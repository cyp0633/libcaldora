package caldav

import (
	"log"
	"net/http"
)

func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PROPFIND received for %s (User: %s, Calendar: %s, Object: %s)", ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	initialResource := ctx.Resource
	_ = fetchChildren(ctx.Depth, initialResource)
	// TODO: Implement PROPFIND logic based on ctx.ResourceType and request body/headers (Depth)
	http.Error(w, "Not Implemented: PROPFIND", http.StatusNotImplemented)
}

func fetchChildren(depth int, parent Resource) (resources []Resource) {
	switch parent.ResourceType {
	case ResourceObject:
		// object does not have children, return
		return
	case ResourcePrincipal:
		// no nested resources for principals, return empty
		return
	case ResourceCollection:
		// find objects (events) in the collection

	case ResourceHomeSet:
	}
	return
}
