package server

import (
	"log"
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handleMkCalendar(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("MKCALENDAR/MKCOL received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement MKCALENDAR logic (creating new calendars) - only valid for storage.ResourceCollection path structure
	if ctx.Resource.ResourceType != storage.ResourceCollection {
		http.Error(w, "Method Not Allowed: MKCALENDAR can only be used to create a calendar collection", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: MKCALENDAR", http.StatusNotImplemented)
}
