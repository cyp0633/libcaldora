package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage/memory"
)

func setupTestServer(t *testing.T) *Server {
	store := memory.New()
	srv, err := New(store, "/caldav")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv
}

func TestServer_Options(t *testing.T) {
	srv := setupTestServer(t)

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/caldav/u/user123/cal", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", w.Code)
	}

	// Check headers
	if got := w.Header().Get(headerDAV); got != davCapabilities {
		t.Errorf("expected DAV header %q, got %q", davCapabilities, got)
	}
	if got := w.Header().Get(headerAllow); got != allowedMethods {
		t.Errorf("expected Allow header %q, got %q", allowedMethods, got)
	}
}

func TestServer_Get_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	// Test GET request for non-existent object
	req := httptest.NewRequest("GET", "/caldav/u/user123/evt/notfound", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %v", w.Code)
	}
}

func TestServer_InvalidPath(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "invalid root path",
			method: "GET",
			path:   "/caldav/invalid",
		},
		{
			name:   "missing user ID",
			method: "GET",
			path:   "/caldav/u//cal",
		},
		{
			name:   "invalid calendar path",
			method: "GET",
			path:   "/caldav/u/user123/invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status NotFound, got %v", w.Code)
			}
		})
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	srv := setupTestServer(t)

	// Test unsupported method
	req := httptest.NewRequest("PATCH", "/caldav/u/user123/cal", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status MethodNotAllowed, got %v", w.Code)
	}
}

func TestServer_CreateCalendar(t *testing.T) {
	srv := setupTestServer(t)

	// Test MKCOL request to create calendar
	req := httptest.NewRequest("MKCOL", "/caldav/u/user123/cal/personal", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %v", w.Code)
	}

	// Test creating the same calendar again
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status MethodNotAllowed, got %v", w.Code)
	}
}

func TestServer_DeleteCalendar(t *testing.T) {
	srv := setupTestServer(t)

	// First create a calendar
	req := httptest.NewRequest("MKCOL", "/caldav/u/user123/cal/personal", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %v", w.Code)
	}

	// Then delete it
	req = httptest.NewRequest("DELETE", "/caldav/u/user123/cal/personal", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status NoContent, got %v", w.Code)
	}

	// Try deleting it again
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %v", w.Code)
	}
}

func TestServer_BaseURIHandling(t *testing.T) {
	// Test with different base URIs
	baseURIs := []string{
		"/caldav",
		"/caldav/",
		"/dav/calendar",
		"/",
		"",
	}

	for _, baseURI := range baseURIs {
		t.Run(baseURI, func(t *testing.T) {
			store := memory.New()
			srv, err := New(store, baseURI)
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			// Test OPTIONS request with the base URI
			path := baseURI
			if path == "" {
				path = "/"
			}
			path += "u/user123/cal"

			req := httptest.NewRequest("OPTIONS", path, nil)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status OK with base URI %q, got %v", baseURI, w.Code)
			}
		})
	}
}
