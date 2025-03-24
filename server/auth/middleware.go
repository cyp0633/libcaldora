package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type contextKey string

const (
	// PrincipalContextKey is the context key for the authenticated principal
	PrincipalContextKey contextKey = "principal"
)

// GetPrincipalFromContext retrieves the authenticated principal from the context
func GetPrincipalFromContext(ctx context.Context) *Principal {
	if p, ok := ctx.Value(PrincipalContextKey).(*Principal); ok {
		return p
	}
	return nil
}

// Middleware creates HTTP middleware that enforces authentication
func Middleware(authenticator Authenticator, realm string) func(http.Handler) http.Handler {
	if realm == "" {
		realm = "CalDAV Server"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for well-known paths
			if strings.HasPrefix(r.URL.Path, "/.well-known/") {
				next.ServeHTTP(w, r)
				return
			}

			// Extract credentials from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				requestAuth(w, realm)
				return
			}

			creds, err := parseBasicAuth(authHeader)
			if err != nil {
				requestAuth(w, realm)
				return
			}

			// Authenticate user
			principal, err := authenticator.Authenticate(r.Context(), creds)
			if err != nil {
				requestAuth(w, realm)
				return
			}

			// Validate access to the requested path
			if err := authenticator.ValidateAccess(r.Context(), principal, r.URL.Path); err != nil {
				if err, ok := err.(*Error); ok && err.Type == ErrForbidden {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				requestAuth(w, realm)
				return
			}

			// Store principal in context
			ctx := context.WithValue(r.Context(), PrincipalContextKey, principal)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requestAuth sends WWW-Authenticate header
func requestAuth(w http.ResponseWriter, realm string) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// parseBasicAuth parses an HTTP Basic Authentication string
func parseBasicAuth(auth string) (Credentials, error) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return Credentials{}, &Error{
			Type:    ErrInvalidCredentials,
			Message: "invalid authorization header format",
		}
	}

	encoded := auth[len(prefix):]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Credentials{}, &Error{
			Type:    ErrInvalidCredentials,
			Message: "invalid base64 encoding",
			Err:     err,
		}
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return Credentials{}, &Error{
			Type:    ErrInvalidCredentials,
			Message: "invalid credentials format",
		}
	}

	return Credentials{
		Username: parts[0],
		Password: parts[1],
	}, nil
}
