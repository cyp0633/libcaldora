package caldav

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// checkAuth enforces Basic Authentication. Returns the username and true if successful.
func (h *CaldavHandler) checkAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.requireAuth(w)
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		log.Printf("Invalid Authorization header format")
		http.Error(w, "Bad Request: Invalid Authorization header format", http.StatusBadRequest)
		return "", false
	}

	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		log.Printf("Failed to decode base64 credentials: %v", err)
		http.Error(w, "Bad Request: Invalid base64 encoding", http.StatusBadRequest)
		return "", false
	}

	credentials := string(decodedBytes)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		log.Printf("Invalid format for decoded credentials")
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
		log.Printf("Empty username provided in Basic Auth")
		h.requireAuth(w) // Treat empty username as unauthorized
		return "", false
	}
	isValidUser := true // Placeholder: Assume valid user if username is not empty
	log.Printf("TODO: Implement real credential validation for user: %s", username)
	// --- End TODO ---

	if !isValidUser {
		log.Printf("Authentication failed for user: %s", username)
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
