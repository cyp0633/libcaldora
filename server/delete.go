package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handleDelete(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("DELETE received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)

	// DELETE is only valid for ResourceObject
	if ctx.Resource.ResourceType != storage.ResourceObject {
		log.Printf("DELETE not allowed on resource type: %s", ctx.Resource.ResourceType)
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}

	// Get the object to check if it exists and to get its ETag
	object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if errors.Is(err, storage.ErrNotFound) {
		log.Printf("Resource not found for deletion: %s", ctx.Resource.ObjectID)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error retrieving object for deletion: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check If-Match header for ETag validation
	ifMatch := r.Header.Get("If-Match")
	if ifMatch != "" && ifMatch != object.ETag {
		log.Printf("ETag mismatch. Client: %s, Server: %s", ifMatch, object.ETag)
		http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
		return
	}

	// Delete the object
	err = h.Storage.DeleteObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if err != nil {
		log.Printf("Failed to delete object: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return success with no content
	log.Printf("Successfully deleted object %s", ctx.Resource.ObjectID)
	w.WriteHeader(http.StatusNoContent)
}
