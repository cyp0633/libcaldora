package handlers

import (
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

// handleMkcol handles MKCOL requests
func (r *Router) handleMkcol(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		r.logger.Error("invalid resource path in MKCOL request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.logger.Info("handling MKCOL request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeCalendar:
		cal := &storage.Calendar{
			ID:     resourcePath.CalendarID,
			UserID: resourcePath.UserID,
			// TODO: Parse calendar properties from request
		}

		if err := r.storage.CreateCalendar(req.Context(), cal); err != nil {
			if e, ok := err.(*storage.Error); ok {
				switch e.Type {
				case storage.ErrAlreadyExists:
					r.logger.Info("calendar already exists",
						"user_id", resourcePath.UserID,
						"calendar_id", resourcePath.CalendarID)
					http.Error(w, "Calendar already exists", http.StatusMethodNotAllowed)
				default:
					r.logger.Error("failed to create calendar",
						"error", err,
						"user_id", resourcePath.UserID,
						"calendar_id", resourcePath.CalendarID)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}
		r.logger.Info("calendar created successfully",
			"user_id", resourcePath.UserID,
			"calendar_id", resourcePath.CalendarID)
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Resource type not supported for MKCOL", http.StatusMethodNotAllowed)
	}
}
