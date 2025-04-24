package server

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandlePut(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Setup test data
	userID := "alice"
	calendarID := "work"
	objectID := "event1.ics"
	encodedPath := "/" + userID + "/cal/" + calendarID + "/" + objectID

	// Create handler with mock storage and URL converter
	urlConverter := &mockURLConverter{}
	// Add slog.Logger parameter with debug level for verbose logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, urlConverter, logger)

	// Create test event data
	now := time.Now()
	testEventData := `BEGIN:VCALENDAR
PRODID:-//libcaldora//NONSGML v1.0//EN
VERSION:2.0
BEGIN:VEVENT
UID:event-uid-1
SUMMARY:Test Event
DTSTART:` + now.Format("20060102T150405Z") + `
DTEND:` + now.Add(1*time.Hour).Format("20060102T150405Z") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`

	// Create an existing object for update tests
	existingEvent := &storage.CalendarObject{
		Path:         encodedPath,
		ETag:         "etag-event-123",
		LastModified: now,
		Component:    ical.NewComponent(ical.CompEvent),
	}
	existingEvent.Component.Props.SetText(ical.PropUID, "event-uid-1")

	// Test cases
	tests := []struct {
		name           string
		resourceType   storage.ResourceType
		setupMocks     func()
		body           string
		headers        map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:           "Non-object resource",
			resourceType:   storage.ResourceCollection,
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:         "Storage error on GetObject",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, errors.New("storage unavailable")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "If-Match does not match",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
			},
			headers: map[string]string{
				"If-Match": "wrong-etag",
			},
			expectedStatus: http.StatusPreconditionFailed,
		},
		{
			name:         "If-None-Match * on existing resource",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
			},
			headers: map[string]string{
				"If-None-Match": "*",
			},
			expectedStatus: http.StatusPreconditionFailed,
		},
		{
			name:         "If-Match on non-existent resource",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
			},
			headers: map[string]string{
				"If-Match": "any-etag",
			},
			expectedStatus: http.StatusPreconditionFailed,
		},
		{
			name:         "Unsupported Media Type",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
			},
			headers: map[string]string{
				"Content-Type": "application/json", // Non-calendar content type
			},
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:         "Invalid iCalendar data",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
			},
			headers: map[string]string{
				"Content-Type": "text/calendar",
			},
			body:           "Invalid iCalendar data",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Storage error on UpdateObject",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == objectID
				})).Return(encodedPath, nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.AnythingOfType("*storage.CalendarObject")).
					Return("", errors.New("storage unavailable")).Once()
			},
			headers: map[string]string{
				"Content-Type": "text/calendar",
			},
			body:           testEventData,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "Create new resource successfully",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == objectID
				})).Return(encodedPath, nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.AnythingOfType("*storage.CalendarObject")).
					Return("new-etag-123", nil).Once()
			},
			headers: map[string]string{
				"Content-Type": "text/calendar",
			},
			body:           testEventData,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, encodedPath, recorder.Header().Get("Location"))
				assert.Equal(t, "new-etag-123", recorder.Header().Get("ETag"))
			},
		},
		{
			name:         "Update existing resource successfully",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == objectID
				})).Return(encodedPath, nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.AnythingOfType("*storage.CalendarObject")).
					Return("updated-etag-123", nil).Once()
			},
			headers: map[string]string{
				"Content-Type": "text/calendar",
				"If-Match":     "etag-event-123", // Match existing ETag
			},
			body:           testEventData,
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, "updated-etag-123", recorder.Header().Get("ETag"))
				assert.Empty(t, recorder.Header().Get("Location")) // No Location on update
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockStorage.ExpectedCalls = nil
			urlConverter.ExpectedCalls = nil

			// Setup mocks
			tt.setupMocks()

			// Create request
			var reqBody string
			if tt.body != "" {
				reqBody = tt.body
			} else {
				reqBody = ""
			}

			req := httptest.NewRequest("PUT", "/caldav/"+userID+"/cal/"+calendarID+"/"+objectID,
				strings.NewReader(reqBody))

			// Add headers
			req.Header.Set("Content-Type", "text/calendar") // Default content type
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Create context
			ctx := &RequestContext{
				Resource: Resource{
					UserID:       userID,
					CalendarID:   calendarID,
					ObjectID:     objectID,
					ResourceType: tt.resourceType,
				},
				AuthUser: userID,
			}

			// Call handler
			handler.handlePut(recorder, req, ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			// Additional response checks if needed
			if tt.checkResponse != nil {
				tt.checkResponse(t, recorder)
			}

			// Verify all expectations were met
			mockStorage.AssertExpectations(t)
			urlConverter.AssertExpectations(t)
		})
	}
}

// Mock URL converter
type mockURLConverter struct {
	mock.Mock
}

func (c *mockURLConverter) EncodePath(resource Resource) (string, error) {
	args := c.Called(resource)
	return args.String(0), args.Error(1)
}

func (c *mockURLConverter) ParsePath(path string) (Resource, error) {
	args := c.Called(path)
	// Ensure the mock returns a Resource struct, even if empty
	res, ok := args.Get(0).(Resource)
	if !ok {
		// Return a default Resource if the type assertion fails
		// This might happen if the mock wasn't set up correctly for ParsePath
		return Resource{}, args.Error(1)
	}
	return res, args.Error(1)
}
