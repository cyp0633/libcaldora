package server

import (
	"errors"
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handleDelete(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	h.Logger.Info("delete request received",
		"resource_type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID)

	// DELETE is only valid for ResourceObject
	if ctx.Resource.ResourceType != storage.ResourceObject {
		h.Logger.Warn("delete not allowed on resource type",
			"resource_type", ctx.Resource.ResourceType)
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}

	// Get the object to check if it exists and to get its ETag
	object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if errors.Is(err, storage.ErrNotFound) {
		h.Logger.Warn("resource not found for deletion",
			"object_id", ctx.Resource.ObjectID)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	} else if err != nil {
		h.Logger.Error("error retrieving object for deletion",
			"error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check If-Match header for ETag validation
	ifMatch := r.Header.Get("If-Match")
	if ifMatch != "" && ifMatch != object.ETag {
		h.Logger.Warn("etag mismatch",
			"client_etag", ifMatch,
			"server_etag", object.ETag)
		http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
		return
	}

	// Delete the object
	err = h.Storage.DeleteObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if err != nil {
		h.Logger.Error("failed to delete object",
			"error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return success with no content
	h.Logger.Info("object deleted successfully",
		"object_id", ctx.Resource.ObjectID)
	w.WriteHeader(http.StatusNoContent)
}
