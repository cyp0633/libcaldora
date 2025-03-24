package auth

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
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

// MiddlewareOptions configures the authentication middleware
type MiddlewareOptions struct {
	Authenticator Authenticator
	Realm         string
	Logger        *slog.Logger
}

// Middleware creates HTTP middleware that enforces authentication
func Middleware(opts MiddlewareOptions) func(http.Handler) http.Handler {
	if opts.Realm == "" {
		opts.Realm = "CalDAV Server"
	}

	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for well-known paths
			if strings.HasPrefix(r.URL.Path, "/.well-known/") {
				opts.Logger.Debug("skipping authentication for well-known path",
					"path", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Extract credentials from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				opts.Logger.Info("missing authorization header",
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr)
				requestAuth(w, opts.Realm)
				return
			}

			creds, err := parseBasicAuth(authHeader)
			if err != nil {
				opts.Logger.Warn("invalid authorization header",
					"error", err,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr)
				requestAuth(w, opts.Realm)
				return
			}

			// Authenticate user
			principal, err := opts.Authenticator.Authenticate(r.Context(), creds)
			if err != nil {
				opts.Logger.Info("authentication failed",
					"username", creds.Username,
					"error", err,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr)
				requestAuth(w, opts.Realm)
				return
			}

			// Validate access to the requested path
			if err := opts.Authenticator.ValidateAccess(r.Context(), principal, r.URL.Path); err != nil {
				if err, ok := err.(*Error); ok && err.Type == ErrForbidden {
					opts.Logger.Warn("access forbidden",
						"username", creds.Username,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				opts.Logger.Info("access validation failed",
					"username", creds.Username,
					"error", err,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr)
				requestAuth(w, opts.Realm)
				return
			}

			opts.Logger.Debug("authentication successful",
				"username", creds.Username,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr)

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
