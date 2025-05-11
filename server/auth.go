package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// checkAuth enforces Basic Authentication. Returns the user ID and true if successful.
func (h *CaldavHandler) checkAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.Logger.Info("authentication required - no auth header")
		h.requireAuth(w)
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		h.Logger.Error("invalid authorization header format")
		http.Error(w, "Bad Request: Invalid Authorization header format", http.StatusBadRequest)
		return "", false
	}

	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		h.Logger.Error("failed to decode base64 credentials",
			"error", err)
		http.Error(w, "Bad Request: Invalid base64 encoding", http.StatusBadRequest)
		return "", false
	}

	credentials := string(decodedBytes)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		h.Logger.Error("invalid format for decoded credentials")
		http.Error(w, "Bad Request: Invalid credentials format", http.StatusBadRequest)
		return "", false
	}

	username := parts[0]
	password := parts[1]

	// Authenticate user
	userID, err := h.Storage.AuthUser(username, password)
	if err != nil {
		h.Logger.Warn("authentication failed",
			"username", username,
			"error", err)
		h.requireAuth(w)
		return "", false
	}

	if userID == "" {
		h.Logger.Warn("authentication failed - invalid credentials",
			"username", username)
		h.requireAuth(w)
		return "", false
	}

	h.Logger.Info("authentication successful",
		"username", username,
		"userID", userID)
	return userID, true
}

// requireAuth sends a 401 Unauthorized response asking for Basic Auth.
func (h *CaldavHandler) requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, h.Realm))
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
