package caldav

import (
	"log"
	"net/http"
)

func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PROPFIND received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement PROPFIND logic based on ctx.ResourceType and request body/headers (Depth)
	http.Error(w, "Not Implemented: PROPFIND", http.StatusNotImplemented)
}
