package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	auth "github.com/cyp0633/libcaldora/server/auth/memory"
	"github.com/cyp0633/libcaldora/server/handlers"
	store "github.com/cyp0633/libcaldora/server/storage/memory"
)

func setupTestServer(t *testing.T) (*Server, *auth.Store) {
	storage := store.New()
	authStore := auth.New()
	if err := authStore.AddUser("testuser", "password"); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	srv, err := New(Options{
		Storage: storage,
		BaseURI: "/caldav",
		Auth:    authStore,
		Realm:   "Test Realm",
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv, authStore
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestServer_Options(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/caldav/u/testuser/cal", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", w.Code)
	}

	// Check headers
	if got := w.Header().Get(handlers.HeaderDAV); got != handlers.DavCapabilities {
		t.Errorf("expected DAV header %q, got %q", handlers.DavCapabilities, got)
	}
	if got := w.Header().Get(handlers.HeaderAllow); got != handlers.AllowedMethods {
		t.Errorf("expected Allow header %q, got %q", handlers.AllowedMethods, got)
	}
}

func TestServer_Authentication(t *testing.T) {
	srv, _ := setupTestServer(t)

	tests := []struct {
		name     string
		auth     string
		path     string
		wantCode int
	}{
		{
			name:     "no auth",
			auth:     "",
			path:     "/caldav/u/testuser/cal",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid auth format",
			auth:     "Basic invalid",
			path:     "/caldav/u/testuser/cal",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "wrong password",
			auth:     basicAuth("testuser", "wrongpass"),
			path:     "/caldav/u/testuser/cal",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "valid auth",
			auth:     basicAuth("testuser", "password"),
			path:     "/caldav/u/testuser/cal",
			wantCode: http.StatusOK,
		},
		{
			name:     "wrong user path",
			auth:     basicAuth("testuser", "password"),
			path:     "/caldav/u/otheruser/cal",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "well-known no auth",
			auth:     "",
			path:     "/.well-known/caldav",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("running test %s\n", tt.name)
			req := httptest.NewRequest("OPTIONS", tt.path, nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %v, got %v", tt.wantCode, w.Code)
			}
		})
	}
}

func TestServer_Get_NotFound(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Test GET request for non-existent object
	req := httptest.NewRequest("GET", "/caldav/u/testuser/evt/notfound", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %v", w.Code)
	}
}

func TestServer_InvalidPath(t *testing.T) {
	srv, _ := setupTestServer(t)

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
			path:   "/caldav/u/testuser/invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", basicAuth("testuser", "password"))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status NotFound, got %v", w.Code)
			}
		})
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Test unsupported method
	req := httptest.NewRequest("PATCH", "/caldav/u/testuser/cal", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status MethodNotAllowed, got %v", w.Code)
	}
}

func TestServer_CreateCalendar(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Test MKCOL request to create calendar
	req := httptest.NewRequest("MKCOL", "/caldav/u/testuser/cal/personal", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
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
	srv, _ := setupTestServer(t)

	// First create a calendar
	req := httptest.NewRequest("MKCOL", "/caldav/u/testuser/cal/personal", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status Created, got %v", w.Code)
	}

	// Then delete it
	req = httptest.NewRequest("DELETE", "/caldav/u/testuser/cal/personal", nil)
	req.Header.Set("Authorization", basicAuth("testuser", "password"))
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
			storage := store.New()
			authStore := auth.New()
			if err := authStore.AddUser("testuser", "password"); err != nil {
				t.Fatalf("failed to create test user: %v", err)
			}

			srv, err := New(Options{
				Storage: storage,
				BaseURI: baseURI,
				Auth:    authStore,
				Realm:   "Test Realm",
			})
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			// Test OPTIONS request with the base URI
			path := baseURI
			if path == "" {
				path = "/"
			}
			path += "u/testuser/cal"

			req := httptest.NewRequest("OPTIONS", path, nil)
			req.Header.Set("Authorization", basicAuth("testuser", "password"))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status OK with base URI %q, got %v", baseURI, w.Code)
			}
		})
	}
}
