package handlers

import (
	"net/http"

	"github.com/cyp0633/libcaldora/server/storage"
)

// handleReport handles REPORT requests
func (r *Router) handleReport(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)
	_, err := storage.ParseResourcePath(path)
	if err != nil {
		r.logger.Error("invalid resource path in REPORT request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.logger.Debug("handling REPORT request",
		"path", path)

	// TODO: Parse REPORT request and handle different report types
	w.Header().Set(HeaderContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
}
