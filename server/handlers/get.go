package handlers

import (
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

// handleGet handles GET requests
func (r *Router) handleGet(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		r.logger.Error("invalid resource path in GET request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeObject:
		obj, err := r.storage.GetObject(req.Context(), resourcePath.UserID, resourcePath.ObjectID)
		if err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				r.logger.Info("object not found",
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				r.logger.Error("failed to get object",
					"error", err,
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set(HeaderContentType, MimeTypeCalendar)
		w.Header().Set(HeaderETag, obj.ETag)
		// TODO: Encode calendar object to iCalendar format
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Resource type not supported for GET", http.StatusMethodNotAllowed)
	}
}
