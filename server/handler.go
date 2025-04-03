package caldav

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// ResourceType indicates the type of CalDAV resource identified by the URL path.
type ResourceType int

const (
	ResourceUnknown ResourceType = iota
	ResourcePrincipal
	ResourceHomeSet
	ResourceCollection
	ResourceObject
)

// String provides a human-readable representation of the ResourceType.
func (rt ResourceType) String() string {
	switch rt {
	case ResourcePrincipal:
		return "Principal"
	case ResourceHomeSet:
		return "HomeSet"
	case ResourceCollection:
		return "Collection"
	case ResourceObject:
		return "Object"
	default:
		return "Unknown"
	}
}

// RequestContext holds parsed information about the incoming CalDAV request.
type RequestContext struct {
	UserID       string
	CalendarID   string
	ObjectID     string
	ResourceType ResourceType
	AuthUser     string // Authenticated user (from Basic Auth)
	// Add other relevant context if needed, e.g., Depth header
}

// CaldavHandler is the main HTTP handler for CalDAV requests under a specific prefix.
type CaldavHandler struct {
	Prefix string // e.g., "/caldav/"
	Realm  string // Realm for Basic Auth
	// TODO: Add backend interface dependency here later
}

// NewCaldavHandler creates a new CaldavHandler.
func NewCaldavHandler(prefix, realm string) *CaldavHandler {
	// Ensure prefix starts and ends with a slash for consistent parsing
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	return &CaldavHandler{
		Prefix: prefix,
		Realm:  realm,
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

	ctx, err := h.parsePath(relativePath)
	if err != nil {
		log.Printf("Error parsing path '%s': %v", relativePath, err)
		http.Error(w, err.Error(), http.StatusNotFound) // Or BadRequest depending on error
		return
	}
	ctx.AuthUser = authUser // Store authenticated user in context

	log.Printf("Parsed path: Type=%s, UserID=%s, CalendarID=%s, ObjectID=%s",
		ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)

	// 3. Routing based on HTTP Method (CalDAV methods)
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

// parsePath analyzes the path relative to the handler's prefix and extracts resource info.
func (h *CaldavHandler) parsePath(path string) (*RequestContext, error) {
	ctx := &RequestContext{ResourceType: ResourceUnknown}
	parts := strings.Split(path, "/")

	// Filter out empty segments caused by leading/trailing/double slashes
	var segments []string
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}

	numSegments := len(segments)

	switch numSegments {
	case 0: // Path was just the prefix itself (e.g., /caldav/) - Usually not a valid CalDAV resource
		return nil, fmt.Errorf("invalid path: root path not directly addressable")

	case 1: // /<userid>
		ctx.UserID = segments[0]
		ctx.ResourceType = ResourcePrincipal
		// TODO: Check if ctx.UserID is a valid user identifier format/exists
		log.Printf("TODO: Validate principal UserID: %s", ctx.UserID)

	case 2: // /<userid>/cal
		if segments[1] == "cal" {
			ctx.UserID = segments[0]
			ctx.ResourceType = ResourceHomeSet
			// TODO: Check if ctx.UserID is valid and has a calendar home set
			log.Printf("TODO: Validate homeset UserID: %s", ctx.UserID)
		} else {
			return nil, fmt.Errorf("invalid path: expected '/<userid>/cal', got '/%s/%s'", segments[0], segments[1])
		}

	case 3: // /<userid>/cal/<calendarid>
		if segments[1] == "cal" {
			ctx.UserID = segments[0]
			ctx.CalendarID = segments[2]
			ctx.ResourceType = ResourceCollection
			// TODO: Check if UserID and CalendarID are valid/exist
			log.Printf("TODO: Validate collection UserID: %s, CalendarID: %s", ctx.UserID, ctx.CalendarID)
		} else {
			return nil, fmt.Errorf("invalid path: expected '/<userid>/cal/<calendarid>', got '/%s/%s/%s'", segments[0], segments[1], segments[2])
		}

	case 4: // /<userid>/cal/<calendarid>/<objectid>
		if segments[1] == "cal" {
			ctx.UserID = segments[0]
			ctx.CalendarID = segments[2]
			ctx.ObjectID = segments[3] // Object ID might contain ".ics"
			ctx.ResourceType = ResourceObject
			// TODO: Check if UserID, CalendarID, and ObjectID are valid/exist
			log.Printf("TODO: Validate object UserID: %s, CalendarID: %s, ObjectID: %s", ctx.UserID, ctx.CalendarID, ctx.ObjectID)
		} else {
			return nil, fmt.Errorf("invalid path: expected '/<userid>/cal/<calendarid>/<objectid>', got '/%s/%s/%s/%s'", segments[0], segments[1], segments[2], segments[3])
		}

	default: // More than 4 segments - not defined by our convention
		return nil, fmt.Errorf("invalid path: too many segments (%d)", numSegments)
	}

	// --- TODO: User Access Control Check ---
	// After identifying the resource and the authenticated user (ctx.AuthUser),
	// check if ctx.AuthUser is allowed to access the resource identified by
	// ctx.UserID, ctx.CalendarID etc. For example, normally ctx.AuthUser must
	// be equal to ctx.UserID unless delegation or public calendars are involved.
	if ctx.UserID != "" && ctx.UserID != ctx.AuthUser {
		log.Printf("TODO: Implement access control check: AuthUser '%s' accessing UserID '%s'", ctx.AuthUser, ctx.UserID)
		// For now, let's assume users can only access their own resources
		// return nil, fmt.Errorf("forbidden: user %s cannot access resources for %s", ctx.AuthUser, ctx.UserID) // Return 403 Forbidden
	}
	// --- End TODO ---

	return ctx, nil
}

// --- Placeholder CalDAV Method Handlers ---
// These functions will be called by ServeHTTP based on the request method.
// They currently just log and return a 501 Not Implemented status.

func (h *CaldavHandler) handleReport(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("REPORT received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement REPORT logic (e.g., calendar-query, calendar-multiget) based on ctx.ResourceType and request body
	http.Error(w, "Not Implemented: REPORT", http.StatusNotImplemented)
}

func (h *CaldavHandler) handlePut(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("PUT received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement PUT logic (creating/updating calendar objects) - only valid for ResourceObject
	if ctx.ResourceType != ResourceObject {
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: PUT", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleGet(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("GET received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement GET logic (retrieving calendar objects) - typically only valid for ResourceObject
	if ctx.ResourceType != ResourceObject {
		// Technically GET might be allowed on collections by some servers (listing?), but often not.
		// GET on Principal/HomeSet is unusual in CalDAV.
		http.Error(w, "Method Not Allowed on this resource type (or GET not implemented)", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: GET", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleDelete(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("DELETE received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement DELETE logic (deleting calendars or objects) - valid for ResourceCollection and ResourceObject
	if ctx.ResourceType != ResourceCollection && ctx.ResourceType != ResourceObject {
		http.Error(w, "Method Not Allowed on this resource type", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: DELETE", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleMkCalendar(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("MKCALENDAR/MKCOL received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Implement MKCALENDAR logic (creating new calendars) - only valid for ResourceCollection path structure
	if ctx.ResourceType != ResourceCollection {
		http.Error(w, "Method Not Allowed: MKCALENDAR can only be used to create a calendar collection", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not Implemented: MKCALENDAR", http.StatusNotImplemented)
}

func (h *CaldavHandler) handleOptions(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("OPTIONS received for %s (User: %s, Calendar: %s, Object: %s)", ctx.ResourceType, ctx.UserID, ctx.CalendarID, ctx.ObjectID)
	// TODO: Set correct Allow and DAV headers based on ctx.ResourceType and capabilities
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
