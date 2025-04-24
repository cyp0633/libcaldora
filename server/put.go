package server

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handlePut(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	h.Logger.Info("put request received",
		"resource_type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID)

	if ctx.Resource.ResourceType != storage.ResourceObject {
		h.Logger.Warn("put not allowed on resource type",
			"resource_type", ctx.Resource.ResourceType)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1) Load existing object (or note that it doesn't exist)
	object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if errors.Is(err, storage.ErrNotFound) {
		object = nil
		h.Logger.Debug("object does not exist, will create new")
	} else if err != nil {
		h.Logger.Error("storage error while retrieving object",
			"error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else {
		h.Logger.Debug("existing object found",
			"etag", object.ETag)
	}

	// 2) Validate preconditions
	ifMatch := r.Header.Get("If-Match")
	ifNone := r.Header.Get("If-None-Match")
	if object != nil {
		if ifMatch != "" && ifMatch != object.ETag {
			h.Logger.Warn("etag mismatch",
				"client_etag", ifMatch,
				"server_etag", object.ETag)
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
		if ifNone == "*" {
			h.Logger.Warn("if-none-match=* used but resource exists")
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
	} else {
		// object==nil → creation
		if ifMatch != "" {
			// If-Match on a non-existent resource → 412
			h.Logger.Warn("if-match used on non-existent resource",
				"etag", ifMatch)
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
	}
	// (Optional) If-Unmodified-Since handling here…

	// 3) Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/calendar") {
		h.Logger.Warn("unsupported media type",
			"content_type", contentType)
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	// 4) Read & parse
	data, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read request body",
			"error", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	comp, err := storage.ICSToICalComp(string(data))
	if err != nil {
		h.Logger.Warn("invalid iCalendar data",
			"error", err)
		http.Error(w, "Invalid iCalendar data", http.StatusBadRequest)
		return
	}

	// 5) Persist
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		// that resource is from path decoding, should not fail
		h.Logger.Error("unexpected error encoding path",
			"error", err,
			"resource", ctx.Resource)
		http.Error(w, "Failed to encode path", http.StatusInternalServerError)
		return
	}
	newObj := &storage.CalendarObject{Path: path, Component: comp}
	newETag, err := h.Storage.UpdateObject(ctx.Resource.UserID, ctx.Resource.CalendarID, newObj)
	if err != nil {
		h.Logger.Error("failed to save object",
			"error", err)
		http.Error(w, "Failed to save object", http.StatusInternalServerError)
		return
	}

	// 6) Respond
	w.Header().Set("ETag", newETag)
	if object == nil {
		h.Logger.Info("object created successfully",
			"path", newObj.Path,
			"etag", newETag)
		w.Header().Set("Location", newObj.Path)
		w.WriteHeader(http.StatusCreated)
	} else {
		h.Logger.Info("object updated successfully",
			"path", newObj.Path,
			"etag", newETag)
		w.WriteHeader(http.StatusNoContent)
	}
}
