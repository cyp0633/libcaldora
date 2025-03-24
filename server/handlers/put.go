package handlers

import (
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

// handlePut handles PUT requests
func (r *Router) handlePut(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		r.logger.Error("invalid resource path in PUT request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.logger.Info("handling PUT request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeObject:
		// TODO: Parse iCalendar data and store object
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Resource type not supported for PUT", http.StatusMethodNotAllowed)
	}
}
