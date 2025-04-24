package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// checkAuth enforces Basic Authentication. Returns the username and true if successful.
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
	_ = parts[1] // Password is intentionally unused for now

	// --- TODO: Implement actual user authentication ---
	// This is where you would typically look up the user `username` in your
	// user database and verify the `password`.
	// For now, we just check if a username was provided.
	if username == "" {
		h.Logger.Warn("empty username provided in basic auth")
		h.requireAuth(w) // Treat empty username as unauthorized
		return "", false
	}
	isValidUser := true // Placeholder: Assume valid user if username is not empty
	h.Logger.Debug("credential validation needed",
		"user", username,
		"message", "TODO: Implement real credential validation")
	// --- End TODO ---

	if !isValidUser {
		h.Logger.Warn("authentication failed",
			"user", username)
		h.requireAuth(w)
		return "", false
	}

	// Authentication successful (for now)
	return username, true
}

// requireAuth sends a 401 Unauthorized response asking for Basic Auth.
func (h *CaldavHandler) requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, h.Realm))
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
