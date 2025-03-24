package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml"
	"github.com/cyp0633/libcaldora/server/auth"
	"github.com/cyp0633/libcaldora/server/storage"
)

const (
	// HTTP headers
	headerContentType = "Content-Type"
	headerETag        = "ETag"
	headerDAV         = "DAV"
	headerAllow       = "Allow"

	// MIME types
	mimeTypeCalendar = "text/calendar; charset=utf-8"

	// DAV capability values
	davCapabilities = "1, calendar-access"
	allowedMethods  = "OPTIONS, PROPFIND, REPORT, GET, PUT, DELETE, MKCOL"
)

// stripPrefix removes the baseURI prefix from the path
func stripPrefix(path, baseURI string) string {
	return strings.TrimPrefix(path, baseURI)
}

// Server represents a CalDAV server
type Server struct {
	storage  storage.Storage
	baseURI  string
	handlers map[string]http.HandlerFunc
	handler  http.Handler
	logger   *slog.Logger
}

// Options configures a CalDAV server
type Options struct {
	Storage storage.Storage
	BaseURI string
	Auth    auth.Authenticator
	Realm   string
	Logger  *slog.Logger // Optional logger, defaults to slog.Default()
}

// New creates a new CalDAV server
func New(opts Options) (*Server, error) {
	if opts.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	s := &Server{
		storage:  opts.Storage,
		baseURI:  opts.BaseURI,
		handlers: make(map[string]http.HandlerFunc),
		logger:   logger,
	}

	// Register method handlers
	s.handlers["OPTIONS"] = s.handleOptions
	s.handlers["PROPFIND"] = s.handlePropfind
	s.handlers["REPORT"] = s.handleReport
	s.handlers["GET"] = s.handleGet
	s.handlers["PUT"] = s.handlePut
	s.handlers["DELETE"] = s.handleDelete
	s.handlers["MKCOL"] = s.handleMkcol

	// Create base handler
	var handler http.Handler = http.HandlerFunc(s.serveHTTP)

	// Apply authentication middleware if configured
	if opts.Auth != nil {
		handler = auth.Middleware(auth.MiddlewareOptions{
			Authenticator: opts.Auth,
			Realm:         opts.Realm,
			Logger:        s.logger,
		})(handler)
	}

	s.handler = handler
	return s, nil
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// serveHTTP is the internal handler that processes CalDAV methods
func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("received request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)

	handler, ok := s.handlers[r.Method]
	if !ok {
		s.logger.Warn("method not allowed",
			"method", r.Method,
			"path", r.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handler(w, r)
}

// Method handlers

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("handling OPTIONS request", "path", r.URL.Path)
	// Set DAV headers
	w.Header().Set(headerDAV, davCapabilities)
	w.Header().Set(headerAllow, allowedMethods)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePropfind(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)

	// Read request body
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(r.Body); err != nil {
		s.logger.Error("failed to read PROPFIND request body",
			"error", err,
			"path", path)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse PROPFIND request
	var propfind xml.PropfindRequest
	if err := propfind.Parse(doc); err != nil {
		s.logger.Error("failed to parse PROPFIND request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build multistatus response
	response := &xml.MultistatusResponse{
		Responses: []xml.Response{
			{
				Href:      s.baseURI + path,
				PropStats: []xml.PropStat{},
			},
		},
	}

	// Get requested properties
	props := propfind.Prop
	if propfind.AllProp {
		// Standard properties to include for allprop
		props = []string{
			xml.TagResourcetype,
			"getcontenttype",
			"displayname",
		}
		props = append(props, propfind.Include...)
	}

	// Handle root path specially
	if path == "" || path == "/" {
		// Split properties into found and not found
		foundProps := []xml.Property{}
		notFoundProps := []xml.Property{}

		for _, prop := range props {
			switch prop {
			case xml.TagResourcetype:
				// Root is a collection
				foundProps = append(foundProps, xml.Property{
					Name:      xml.TagResourcetype,
					Namespace: xml.DAV,
					Children: []xml.Property{
						{Name: xml.TagCollection, Namespace: xml.DAV},
					},
				})
			default:
				// Other properties not found on root
				notFoundProps = append(notFoundProps, xml.Property{
					Name:      prop,
					Namespace: xml.DAV,
				})
			}
		}

		// Add found properties
		if len(foundProps) > 0 {
			response.Responses[0].PropStats = append(response.Responses[0].PropStats, xml.PropStat{
				Props:  foundProps,
				Status: "HTTP/1.1 200 OK",
			})
		}

		// Add not found properties
		if len(notFoundProps) > 0 {
			response.Responses[0].PropStats = append(response.Responses[0].PropStats, xml.PropStat{
				Props:  notFoundProps,
				Status: "HTTP/1.1 404 Not Found",
			})
		}
	} else {
		// Non-root paths
		_, err := storage.ParseResourcePath(path)
		if err != nil {
			s.logger.Error("invalid resource path in PROPFIND request",
				"error", err,
				"path", path)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// TODO: Handle non-root paths
	}

	// Convert response to XML and send
	respDoc := response.ToXML()
	w.Header().Set(headerContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	respDoc.WriteTo(w)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	_, err := storage.ParseResourcePath(path)
	if err != nil {
		s.logger.Error("invalid resource path in REPORT request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Debug("handling REPORT request",
		"path", path)

	// TODO: Parse REPORT request and handle different report types
	w.Header().Set(headerContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		s.logger.Error("invalid resource path in GET request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeObject:
		obj, err := s.storage.GetObject(r.Context(), resourcePath.UserID, resourcePath.ObjectID)
		if err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				s.logger.Info("object not found",
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				s.logger.Error("failed to get object",
					"error", err,
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set(headerContentType, mimeTypeCalendar)
		w.Header().Set(headerETag, obj.ETag)
		// TODO: Encode calendar object to iCalendar format
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Resource type not supported for GET", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		s.logger.Error("invalid resource path in PUT request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Info("handling PUT request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeObject:
		// TODO: Parse iCalendar data and store object
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Resource type not supported for PUT", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		s.logger.Error("invalid resource path in DELETE request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Info("handling DELETE request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeCalendar:
		if err := s.storage.DeleteCalendar(r.Context(), resourcePath.UserID, resourcePath.CalendarID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				s.logger.Info("calendar not found for deletion",
					"user_id", resourcePath.UserID,
					"calendar_id", resourcePath.CalendarID)
				http.Error(w, "Calendar not found", http.StatusNotFound)
			} else {
				s.logger.Error("failed to delete calendar",
					"error", err,
					"user_id", resourcePath.UserID,
					"calendar_id", resourcePath.CalendarID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		s.logger.Info("calendar deleted successfully",
			"user_id", resourcePath.UserID,
			"calendar_id", resourcePath.CalendarID)
		w.WriteHeader(http.StatusNoContent)

	case storage.ResourceTypeObject:
		if err := s.storage.DeleteObject(r.Context(), resourcePath.UserID, resourcePath.ObjectID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				s.logger.Info("object not found for deletion",
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				s.logger.Error("failed to delete object",
					"error", err,
					"user_id", resourcePath.UserID,
					"object_id", resourcePath.ObjectID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		s.logger.Info("object deleted successfully",
			"user_id", resourcePath.UserID,
			"object_id", resourcePath.ObjectID)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Resource type not supported for DELETE", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMkcol(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		s.logger.Error("invalid resource path in MKCOL request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Info("handling MKCOL request",
		"path", path,
		"user_id", resourcePath.UserID,
		"resource_type", resourcePath.Type)

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeCalendar:
		cal := &storage.Calendar{
			ID:     resourcePath.CalendarID,
			UserID: resourcePath.UserID,
			// TODO: Parse calendar properties from request
		}

		if err := s.storage.CreateCalendar(r.Context(), cal); err != nil {
			if e, ok := err.(*storage.Error); ok {
				switch e.Type {
				case storage.ErrAlreadyExists:
					s.logger.Info("calendar already exists",
						"user_id", resourcePath.UserID,
						"calendar_id", resourcePath.CalendarID)
					http.Error(w, "Calendar already exists", http.StatusMethodNotAllowed)
				default:
					s.logger.Error("failed to create calendar",
						"error", err,
						"user_id", resourcePath.UserID,
						"calendar_id", resourcePath.CalendarID)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}
		s.logger.Info("calendar created successfully",
			"user_id", resourcePath.UserID,
			"calendar_id", resourcePath.CalendarID)
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Resource type not supported for MKCOL", http.StatusMethodNotAllowed)
	}
}
