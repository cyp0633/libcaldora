package server

import (
	"fmt"
	"net/http"
	"strings"

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
}

// New creates a new CalDAV server
func New(store storage.Storage, baseURI string) (*Server, error) {
	if store == nil {
		return nil, fmt.Errorf("storage is required")
	}

	s := &Server{
		storage:  store,
		baseURI:  baseURI,
		handlers: make(map[string]http.HandlerFunc),
	}

	// Register method handlers
	s.handlers["OPTIONS"] = s.handleOptions
	s.handlers["PROPFIND"] = s.handlePropfind
	s.handlers["REPORT"] = s.handleReport
	s.handlers["GET"] = s.handleGet
	s.handlers["PUT"] = s.handlePut
	s.handlers["DELETE"] = s.handleDelete
	s.handlers["MKCOL"] = s.handleMkcol

	return s, nil
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, ok := s.handlers[r.Method]
	if !ok {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handler(w, r)
}

// Method handlers

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	// Set DAV headers
	w.Header().Set(headerDAV, davCapabilities)
	w.Header().Set(headerAllow, allowedMethods)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePropfind(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	_, err := storage.ParseResourcePath(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// TODO: Parse PROPFIND request and build response
	w.Header().Set(headerContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	_, err := storage.ParseResourcePath(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// TODO: Parse REPORT request and handle different report types
	w.Header().Set(headerContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	// Parse resource path
	path := stripPrefix(r.URL.Path, s.baseURI)
	resourcePath, err := storage.ParseResourcePath(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeObject:
		obj, err := s.storage.GetObject(r.Context(), resourcePath.UserID, resourcePath.ObjectID)
		if err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Handle different resource types
	switch resourcePath.Type {
	case storage.ResourceTypeCalendar:
		if err := s.storage.DeleteCalendar(r.Context(), resourcePath.UserID, resourcePath.CalendarID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				http.Error(w, "Calendar not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case storage.ResourceTypeObject:
		if err := s.storage.DeleteObject(r.Context(), resourcePath.UserID, resourcePath.ObjectID); err != nil {
			if e, ok := err.(*storage.Error); ok && e.Type == storage.ErrNotFound {
				http.Error(w, "Object not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

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
					http.Error(w, "Calendar already exists", http.StatusMethodNotAllowed)
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Resource type not supported for MKCOL", http.StatusMethodNotAllowed)
	}
}
