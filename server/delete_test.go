package server

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
)

func TestHandleDelete(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Setup test data
	userID := "alice"
	calendarID := "work"
	objectID := "event1.ics"

	// Create handler with mock storage and URL converter
	urlConverter := &mockURLConverter{}
	// Add slog.Logger parameter with debug level for verbose logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, urlConverter, logger)

	// Create an existing object for deletion tests
	now := time.Now()
	existingEvent := &storage.CalendarObject{
		Path:         "/" + userID + "/cal/" + calendarID + "/" + objectID,
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
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "Non-object resource",
			resourceType:   storage.ResourceCollection,
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:         "Resource not found",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, storage.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
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
			name:         "ETag mismatch",
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
			name:         "Storage error on DeleteObject",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
				mockStorage.On("DeleteObject", userID, calendarID, objectID).
					Return(errors.New("deletion failed")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "Successful deletion without If-Match",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
				mockStorage.On("DeleteObject", userID, calendarID, objectID).
					Return(nil).Once()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "Successful deletion with matching If-Match",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(existingEvent, nil).Once()
				mockStorage.On("DeleteObject", userID, calendarID, objectID).
					Return(nil).Once()
			},
			headers: map[string]string{
				"If-Match": "etag-event-123",
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockStorage.ExpectedCalls = nil

			// Setup mocks
			tt.setupMocks()

			// Create request
			req := httptest.NewRequest("DELETE", "/caldav/"+userID+"/cal/"+calendarID+"/"+objectID, nil)

			// Add headers
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
			handler.handleDelete(recorder, req, ctx)

			// Assert response code
			assert.Equal(t, tt.expectedStatus, recorder.Code, "Expected status %d, got %d for test: %s",
				tt.expectedStatus, recorder.Code, tt.name)

			// Verify all expectations were met
			mockStorage.AssertExpectations(t)
		})
	}
}
