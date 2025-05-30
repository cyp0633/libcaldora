package server

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/cyp0633/libcaldora/server/storage"
)

// RequestContext holds parsed information about the incoming CalDAV request.
type RequestContext struct {
	Resource Resource // Contains UserID, CalendarID, ObjectID, and ResourceType
	AuthUser string   // Authenticated user (from Basic Auth)
	Depth    int      // >3 is the same as infinity
	// Add other relevant context if needed
}

// CaldavHandler is the main HTTP handler for CalDAV requests under a specific prefix.
type CaldavHandler struct {
	Prefix       string // e.g., "/caldav/"
	Realm        string // Realm for Basic Auth
	Storage      storage.Storage
	MaxDepth     int // Optional: Max depth for PROPFIND requests, >3 for infinity
	URLConverter URLConverter
	Logger       *slog.Logger // Logger for structured logging
	// TODO: Add backend interface dependency here later
}

// NewCaldavHandler creates a new CaldavHandler.
func NewCaldavHandler(prefix, realm string, storage storage.Storage, maxDepth int, converter URLConverter, logger *slog.Logger) *CaldavHandler {
	// Ensure prefix starts and ends with a slash for consistent parsing
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	if converter == nil {
		converter = &DefaultURLConverter{Prefix: prefix}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &CaldavHandler{
		Prefix:       prefix,
		Realm:        realm,
		Storage:      storage,
		MaxDepth:     maxDepth,
		URLConverter: converter,
		Logger:       logger,
	}
}

// ServeHTTP handles incoming HTTP requests, performs authentication, parsing, and routing.
func (h *CaldavHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("received request",
		"method", r.Method,
		"path", r.URL.Path,
	)

	// 1. Basic Authentication Check
	userID, ok := h.checkAuth(w, r)
	if !ok {
		// checkAuth already sent the 401 response
		return
	}

	h.Logger.Info("authenticated user", "userID", userID)

	// 2. Path Parsing - now handled directly by the URL converter
	resource, err := h.URLConverter.ParsePath(r.URL.Path)
	if err != nil {
		h.Logger.Error("error parsing path",
			"path", r.URL.Path,
			"error", err,
		)
		http.Error(w, err.Error(), http.StatusNotFound) // Or BadRequest depending on error
		return
	}

	// Create request context with the parsed resource
	ctx := &RequestContext{
		Resource: resource,
		AuthUser: userID, // Use the user ID directly
	}

	h.Logger.Info("parsed path",
		"type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID,
	)

	// 3. --- TODO: User Access Control Check ---
	// After identifying the resource and the authenticated user (ctx.AuthUser),
	// check if ctx.AuthUser is allowed to access the resource identified by
	// ctx.UserID, ctx.CalendarID etc. For example, normally ctx.AuthUser must
	// be equal to ctx.UserID unless delegation or public calendars are involved.
	if ctx.Resource.UserID != "" && ctx.Resource.UserID != ctx.AuthUser {
		h.Logger.Warn("access control not implemented",
			"auth_user", ctx.AuthUser,
			"user_id", ctx.Resource.UserID,
		)
		// For now, let's assume users can only access their own resources
		http.Error(w, "Forbidden: Access denied to the requested resource", http.StatusForbidden)
		return
	}
	// --- End TODO ---

	depth := r.Header.Get("Depth")
	if depth == "" {
		ctx.Depth = 0 // Default depth
	} else if depth == "infinity" {
		ctx.Depth = 114514
	} else {
		// Parse depth as integer, default to 0 if invalid
		var err error
		ctx.Depth, err = strconv.Atoi(depth)
		if err != nil {
			h.Logger.Warn("invalid depth header",
				"value", depth,
				"default", 0,
			)
			ctx.Depth = 0
		}
		ctx.Depth = min(ctx.Depth, h.MaxDepth)
	}

	// 4. Routing based on HTTP Method (CalDAV methods)
	switch r.Method {
	case "PROPFIND":
		h.handlePropfind(w, r, ctx)
	case "REPORT":
		h.handleReport(w, r, ctx)
	case "PUT":
		h.handlePut(w, r, ctx)
	case "GET":
		h.handleGet(w, r, ctx)
	case "DELETE":
		h.handleDelete(w, r, ctx)
	case "MKCOL", "MKCALENDAR": // MKCALENDAR is often used instead of MKCOL for calendars
		h.handleMkCalendar(w, r, ctx)
	case "OPTIONS":
		h.handleOptions(w, r, ctx)
	// Add other CalDAV methods like COPY, MOVE if needed
	default:
		h.Logger.Error("method not allowed",
			"method", r.Method,
		)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// --- Placeholder CalDAV Method Handlers ---
// These functions will be called by ServeHTTP based on the request method.
// They currently just log and return a 501 Not Implemented status.

func (h *CaldavHandler) handleOptions(w http.ResponseWriter, _ *http.Request, ctx *RequestContext) {
	h.Logger.Info("options request",
		"type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID,
	)
	// TODO: Set correct Allow and DAV headers based on ctx.Resource.ResourceType and capabilities
	w.Header().Set("Allow", "OPTIONS, PROPFIND, REPORT, GET, PUT, DELETE, MKCALENDAR") // Example, tailor this
	w.Header().Set("DAV", "1, 3, calendar-access")                                     // Example CalDAV capabilities
	w.WriteHeader(http.StatusOK)
}

// --- Example Usage (in your main.go or server setup) ---
/*
func main() {
    caldavPrefix := "/caldav/"
    caldavHandler := caldav.NewCaldavHandler(caldavPrefix, "My CalDAV Server")

    // Need to wrap the handler with http.StripPrefix
    http.Handle(caldavPrefix, http.StripPrefix(strings.TrimSuffix(caldavPrefix, "/"), caldavHandler))

    log.Println("Starting CalDAV server on :8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
*/

// ServeWellKnown handles requests to the well-known CalDAV URL.
func (h *CaldavHandler) ServeWellKnown(w http.ResponseWriter, r *http.Request) {
	redirectURL := "//" + r.Host + h.Prefix

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusMovedPermanently)
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		w.Header().Set("DAV", "1, 3, calendar-access")
		w.WriteHeader(http.StatusOK)
	default:
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

	h.Logger.Info("well-known caldav request",
		"method", r.Method,
		"path", r.URL.Path,
		"redirect_to", redirectURL,
	)
}
