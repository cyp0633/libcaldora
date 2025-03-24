package handlers

import (
	"net/http"
)

// handleOptions handles OPTIONS requests
func (r *Router) handleOptions(w http.ResponseWriter, req *http.Request) {
	r.logger.Debug("handling OPTIONS request", "path", req.URL.Path)

	// Set DAV headers
	w.Header().Set(HeaderDAV, DavCapabilities)
	w.Header().Set(HeaderAllow, AllowedMethods)
	w.WriteHeader(http.StatusOK)
}
