package server

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage"
)

type testAuthProvider struct {
	called bool
	input  BasicAuthInput
	userID string
	err    error
}

func (p *testAuthProvider) AuthenticateBasic(_ context.Context, input BasicAuthInput) (string, error) {
	p.called = true
	p.input = input
	return p.userID, p.err
}

func TestParsePath(t *testing.T) {
	// Create mock storage directly - no longer using NewMockStorage()
	mockStorage := &storage.MockStorage{}
	// Add slog.Logger parameter with debug level for verbose logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil, logger)

	testCases := []struct {
		name           string
		path           string
		wantErr        bool
		wantUserID     string
		wantCalendarID string
		wantObjectID   string
		wantResType    storage.ResourceType
	}{
		{"empty path", "", false, "", "", "", storage.ResourceServiceRoot},
		{"principal", "user1", false, "user1", "", "", storage.ResourcePrincipal},
		{"home set", "user1/cal", false, "user1", "", "", storage.ResourceHomeSet},
		{"invalid home set", "user1/calendar", true, "", "", "", storage.ResourceUnknown},
		{"collection", "user1/cal/personal", false, "user1", "personal", "", storage.ResourceCollection},
		{"object", "user1/cal/personal/event123.ics", false, "user1", "personal", "event123.ics", storage.ResourceObject},
		{"too many segments", "user1/cal/personal/event123/extra", true, "", "", "", storage.ResourceUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ParsePath now returns Resource, not RequestContext
			resource, err := h.URLConverter.ParsePath(tc.path)

			// Check error status
			if (err != nil) != tc.wantErr {
				t.Errorf("parsePath(%q) error = %v, wantErr %v", tc.path, err, tc.wantErr)
				return
			}

			if err != nil {
				// If we expected an error, no need to check the resource
				return
			}

			// Check resource values
			if resource.UserID != tc.wantUserID {
				t.Errorf("UserID = %q, want %q", resource.UserID, tc.wantUserID)
			}
			if resource.CalendarID != tc.wantCalendarID {
				t.Errorf("CalendarID = %q, want %q", resource.CalendarID, tc.wantCalendarID)
			}
			if resource.ObjectID != tc.wantObjectID {
				t.Errorf("ObjectID = %q, want %q", resource.ObjectID, tc.wantObjectID)
			}
			if resource.ResourceType != tc.wantResType {
				t.Errorf("ResourceType = %v, want %v", resource.ResourceType, tc.wantResType)
			}
		})
	}
}

func TestResourceTypeString(t *testing.T) {
	tests := []struct {
		rt   storage.ResourceType
		want string
	}{
		{storage.ResourceUnknown, "Unknown"},
		{storage.ResourcePrincipal, "Principal"},
		{storage.ResourceHomeSet, "HomeSet"},
		{storage.ResourceCollection, "Collection"},
		{storage.ResourceObject, "Object"},
		{storage.ResourceType(99), "Unknown"}, // Test invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.rt.String(); got != tt.want {
				t.Errorf("ResourceType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAuth(t *testing.T) {
	// Create mock storage directly
	mockStorage := &storage.MockStorage{}
	// Add slog.Logger parameter with debug level for verbose logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil, logger)

	tests := []struct {
		name           string
		authHeader     string
		wantStatusCode int
		wantUsername   string
		wantSuccess    bool
		setupMock      func()
	}{
		{
			name:           "no auth header",
			authHeader:     "",
			wantStatusCode: http.StatusUnauthorized,
			wantUsername:   "",
			wantSuccess:    false,
			setupMock:      func() {},
		},
		{
			name:           "invalid format",
			authHeader:     "NotBasic abcdef",
			wantStatusCode: http.StatusBadRequest,
			wantUsername:   "",
			wantSuccess:    false,
			setupMock:      func() {},
		},
		{
			name:           "invalid base64",
			authHeader:     "Basic !@#$%^",
			wantStatusCode: http.StatusBadRequest,
			wantUsername:   "",
			wantSuccess:    false,
			setupMock:      func() {},
		},
		{
			name:           "invalid credential format",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("username-without-colon")),
			wantStatusCode: http.StatusBadRequest,
			wantUsername:   "",
			wantSuccess:    false,
			setupMock:      func() {},
		},
		{
			name:           "empty username",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":password")),
			wantStatusCode: http.StatusUnauthorized,
			wantUsername:   "",
			wantSuccess:    false,
			setupMock: func() {
				mockStorage.On("AuthUser", "", "password").Return("", storage.ErrNotFound)
			},
		},
		{
			name:           "successful auth",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user1:password")),
			wantStatusCode: http.StatusOK,
			wantUsername:   "user1",
			wantSuccess:    true,
			setupMock: func() {
				mockStorage.On("AuthUser", "user1", "password").Return("user1", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock expectations
			tt.setupMock()

			req := httptest.NewRequest("GET", "http://example.com/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()

			username, ok := h.checkAuth(rr, req)

			if ok != tt.wantSuccess {
				t.Errorf("checkAuth() success = %v, want %v", ok, tt.wantSuccess)
			}

			if username != tt.wantUsername {
				t.Errorf("checkAuth() username = %v, want %v", username, tt.wantUsername)
			}

			if !tt.wantSuccess && rr.Code != tt.wantStatusCode {
				t.Errorf("checkAuth() status code = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			// Check WWW-Authenticate header is present when needed
			if tt.wantStatusCode == http.StatusUnauthorized {
				if authHeader := rr.Header().Get("WWW-Authenticate"); !strings.HasPrefix(authHeader, "Basic realm=") {
					t.Errorf("Expected WWW-Authenticate header with Basic realm, got %q", authHeader)
				}
			}

			// Verify that all expectations were met
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestCheckAuthWithCustomProvider(t *testing.T) {
	mockStorage := &storage.MockStorage{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil, logger)

	authProvider := &testAuthProvider{userID: "provider-user"}
	h.SetBasicAuthProvider(authProvider)

	credentials := base64.StdEncoding.EncodeToString([]byte("user1:password"))
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	req.Header.Set("User-Agent", "libcaldora-test/1.0")
	req.RemoteAddr = "10.0.0.2:1234"

	rr := httptest.NewRecorder()
	userID, ok := h.checkAuth(rr, req)

	if !ok {
		t.Fatalf("checkAuth() expected success, got failure")
	}
	if userID != "provider-user" {
		t.Fatalf("checkAuth() userID = %q, want %q", userID, "provider-user")
	}
	if !authProvider.called {
		t.Fatalf("expected custom auth provider to be called")
	}
	if authProvider.input.Username != "user1" {
		t.Fatalf("provider username = %q, want %q", authProvider.input.Username, "user1")
	}
	if authProvider.input.Password != "password" {
		t.Fatalf("provider password = %q, want %q", authProvider.input.Password, "password")
	}
	if authProvider.input.UserAgent != "libcaldora-test/1.0" {
		t.Fatalf("provider user-agent = %q, want %q", authProvider.input.UserAgent, "libcaldora-test/1.0")
	}
	if authProvider.input.RemoteAddr != "10.0.0.2:1234" {
		t.Fatalf("provider remote addr = %q, want %q", authProvider.input.RemoteAddr, "10.0.0.2:1234")
	}
}
