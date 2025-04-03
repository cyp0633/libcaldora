package caldav

import (
	"log"
	"net/http"
)

type resource struct {
	UserID       string
	CalendarID   string
	ObjectID     string
	ResourceType ResourceType
}

func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PROPFIND received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	initialResource := resource{
		UserID:       ctx.UserID,
		CalendarID:   ctx.CalendarID,
		ObjectID:     ctx.ObjectID,
		ResourceType: ctx.ResourceType,
	}
	_ = fetchChildren(ctx.Depth, initialResource)
	// TODO: Implement PROPFIND logic based on ctx.ResourceType and request body/headers (Depth)
	http.Error(w, "Not Implemented: PROPFIND", http.StatusNotImplemented)
}

func fetchChildren(depth int, parent resource) (resources []resource) {

	return
}
