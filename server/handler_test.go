package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage"
)

func TestParsePath(t *testing.T) {
	// Create mock storage directly - no longer using NewMockStorage()
	mockStorage := &storage.MockStorage{}
	h := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil)

	testCases := []struct {
		name           string
		path           string
		wantErr        bool
		wantUserID     string
		wantCalendarID string
		wantObjectID   string
		wantResType    storage.ResourceType
	}{
		{"empty path", "", true, "", "", "", storage.ResourceUnknown},
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
	h := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil)

	tests := []struct {
		name           string
		authHeader     string
		wantStatusCode int
		wantUsername   string
		wantSuccess    bool
	}{
		{"no auth header", "", http.StatusUnauthorized, "", false},
		{"invalid format", "NotBasic abcdef", http.StatusBadRequest, "", false},
		{"invalid base64", "Basic !@#$%^", http.StatusBadRequest, "", false},
		{"invalid credential format", "Basic " + base64.StdEncoding.EncodeToString([]byte("username-without-colon")), http.StatusBadRequest, "", false},
		{"empty username", "Basic " + base64.StdEncoding.EncodeToString([]byte(":password")), http.StatusUnauthorized, "", false},
		{"successful auth", "Basic " + base64.StdEncoding.EncodeToString([]byte("user1:password")), http.StatusOK, "user1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
