package server

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/cyp0633/libcaldora/server/storage"
)

func (h *CaldavHandler) handlePut(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	if ctx.Resource.ResourceType != storage.ResourceObject {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1) Load existing object (or note that it doesn't exist)
	object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if errors.Is(err, storage.ErrNotFound) {
		object = nil
	} else if err != nil {
		log.Printf("storage error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 2) Validate preconditions
	ifMatch := r.Header.Get("If-Match")
	ifNone := r.Header.Get("If-None-Match")
	if object != nil {
		if ifMatch != "" && ifMatch != object.ETag {
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
		if ifNone == "*" {
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
	} else {
		// object==nil → creation
		if ifMatch != "" {
			// If-Match on a non-existent resource → 412
			http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
			return
		}
	}
	// (Optional) If-Unmodified-Since handling here…

	// 3) Check Content-Type
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/calendar") {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	// 4) Read & parse
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	comp, err := storage.ICSToICalComp(string(data))
	if err != nil {
		http.Error(w, "Invalid iCalendar data", http.StatusBadRequest)
		return
	}

	// 5) Persist
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		// that resource is from path decoding, should not fail
		log.Printf("UNEXPECTED ERROR - Error encoding path: %v", err)
		http.Error(w, "Failed to encode path", http.StatusInternalServerError)
		return
	}
	newObj := &storage.CalendarObject{Path: path, Component: comp}
	newETag, err := h.Storage.UpdateObject(ctx.Resource.UserID, ctx.Resource.CalendarID, newObj)
	if err != nil {
		http.Error(w, "Failed to save object", http.StatusInternalServerError)
		return
	}

	// 6) Respond
	w.Header().Set("ETag", newETag)
	if object == nil {
		w.Header().Set("Location", newObj.Path)
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
