package handlers

import (
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

// handleDelete handles DELETE requests
func (r *Router) handleDelete(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		r.logger.Error("invalid resource path in DELETE request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.logger.Info("handling DELETE request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeCalendar:
		if err := r.storage.DeleteCalendar(req.Context(), resourcePath.UserID, resourcePath.CalendarID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				r.logger.Info("calendar not found for deletion",
					"user_id", resourcePath.UserID,
					"calendar_id", resourcePath.CalendarID)
				http.Error(w, "Calendar not found", http.StatusNotFound)
			} else {
				r.logger.Error("failed to delete calendar",
					"error", err,
					"user_id", resourcePath.UserID,
					"calendar_id", resourcePath.CalendarID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		r.logger.Info("calendar deleted successfully",
			"user_id", resourcePath.UserID,
			"calendar_id", resourcePath.CalendarID)
		w.WriteHeader(http.StatusNoContent)

	case storage.ResourceTypeObject:
		if err := r.storage.DeleteObject(req.Context(), resourcePath.UserID, resourcePath.ObjectID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				r.logger.Info("object not found for deletion",
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				r.logger.Error("failed to delete object",
					"error", err,
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		r.logger.Info("object deleted successfully",
			"user_id", resourcePath.UserID,
			"object_id", resourcePath.ObjectID)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Resource type not supported for DELETE", http.StatusMethodNotAllowed)
	}
}
