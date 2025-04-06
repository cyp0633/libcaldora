package caldav

import (
	"log"
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
	// TODO: Add backend interface dependency here later
}

// NewCaldavHandler creates a new CaldavHandler.
func NewCaldavHandler(prefix, realm string, storage storage.Storage, maxDepth int, converter URLConverter) *CaldavHandler {
	// Ensure prefix starts and ends with a slash for consistent parsing
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	if converter == nil {
		converter = defaultURLConverter{}
	}
	return &CaldavHandler{
		Prefix:       prefix,
		Realm:        realm,
		Storage:      storage,
		MaxDepth:     maxDepth,
		URLConverter: converter,
	}
}

// ServeHTTP handles incoming HTTP requests, performs authentication, parsing, and routing.
func (h *CaldavHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)

	// 1. Basic Authentication Check
	authUser, ok := h.checkAuth(w, r)
	if !ok {
		// checkAuth already sent the 401 response
		return
	}
	log.Printf("Authenticated user: %s", authUser) // Log username, careful in production

	// 2. Path Parsing (relative to the prefix)
	relativePath := strings.TrimPrefix(r.URL.Path, h.Prefix)
	// Optional: Trim trailing slash for consistency, unless it's significant for collections
	// relativePath = strings.TrimSuffix(relativePath, "/") // Be careful if trailing slash matters for PROPFIND on collections

	resource, err := h.URLConverter.ParsePath(relativePath)
	if err != nil {
		log.Printf("Error parsing path '%s': %v", relativePath, err)
		http.Error(w, err.Error(), http.StatusNotFound) // Or BadRequest depending on error
		return
	}

	// Create request context with the parsed resource
	ctx := &RequestContext{
		Resource: resource,
		AuthUser: authUser,
	}

	log.Printf("Parsed path: Type=%s, UserID=%s, CalendarID=%s, ObjectID=%s",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)

	// 3. --- TODO: User Access Control Check ---
	// After identifying the resource and the authenticated user (ctx.AuthUser),
	// check if ctx.AuthUser is allowed to access the resource identified by
	// ctx.UserID, ctx.CalendarID etc. For example, normally ctx.AuthUser must
	// be equal to ctx.UserID unless delegation or public calendars are involved.
	if ctx.Resource.UserID != "" && ctx.Resource.UserID != ctx.AuthUser {
		log.Printf("TODO: Implement access control check: AuthUser '%s' accessing UserID '%s'", ctx.AuthUser, ctx.Resource.UserID)
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
			log.Printf("Invalid Depth header value: %s, defaulting to 0", depth)
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
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// --- Placeholder CalDAV Method Handlers ---
// These functions will be called by ServeHTTP based on the request method.
// They currently just log and return a 501 Not Implemented status.

func (h *CaldavHandler) handleReport(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("REPORT received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement REPORT logic (e.g., calendar-query, calendar-multiget) based on ctx.Resource.ResourceType and request body
	http.Error(w, "Not Implemented: REPORT", http.StatusNotImplemented)
}

func (h *CaldavHandler) handlePut(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PUT received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement PUT logic (creating/updating calendar objects) - only valid for storage.ResourceObject
	if ctx.Resource.ResourceType != storage.ResourceObject {
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: PUT", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleGet(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("GET received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement GET logic (retrieving calendar objects) - typically only valid for storage.ResourceObject
	if ctx.Resource.ResourceType != storage.ResourceObject {
		// Technically GET might be allowed on collections by some servers (listing?), but often not.
		// GET on Principal/HomeSet is unusual in CalDAV.
		http.Error(w, "Method Not Allowed on this resource type (or GET not implemented)", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: GET", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleDelete(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("DELETE received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement DELETE logic (deleting calendars or objects) - valid for storage.ResourceCollection and storage.ResourceObject
	if ctx.Resource.ResourceType != storage.ResourceCollection && ctx.Resource.ResourceType != storage.ResourceObject {
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: DELETE", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleMkCalendar(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("MKCALENDAR/MKCOL received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	// TODO: Implement MKCALENDAR logic (creating new calendars) - only valid for storage.ResourceCollection path structure
	if ctx.Resource.ResourceType != storage.ResourceCollection {
		http.Error(w, "Method Not Allowed: MKCALENDAR can only be used to create a calendar collection", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: MKCALENDAR", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleOptions(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("OPTIONS received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
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
