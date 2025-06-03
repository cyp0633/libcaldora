package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
)

func TestHandleGet(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Create handler with mock storage
	// Add slog.Logger parameter with debug level for verbose logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil, logger)

	// Setup test data
	userID := "alice"
	calendarID := "work"
	objectID := "event1.ics"

	// Create test calendar
	testCalendar := &storage.Calendar{
		Path:         "/" + userID + "/cal/" + calendarID,
		CTag:         "ctag-123",
		ETag:         "etag-cal-123",
		CalendarData: ical.NewCalendar(),
	}
	testCalendar.CalendarData.Props.SetText(ical.PropName, "Work Calendar")
	// Add required PRODID property
	testCalendar.CalendarData.Props.SetText(ical.PropProductID, "-//libcaldora//NONSGML v1.0//EN")
	// Add required VERSION property
	testCalendar.CalendarData.Props.SetText(ical.PropVersion, "2.0")

	// Create test event
	now := time.Now()
	eventComponent := ical.NewComponent(ical.CompEvent)
	eventComponent.Props.SetText(ical.PropUID, "event-uid-1")
	eventComponent.Props.SetText(ical.PropSummary, "Test Event")
	eventComponent.Props.SetDateTime(ical.PropDateTimeStart, now)
	eventComponent.Props.SetDateTime(ical.PropDateTimeEnd, now.Add(1*time.Hour))
	eventComponent.Props.SetDateTime(ical.PropDateTimeStamp, now)

	testEvent := &storage.CalendarObject{
		Path:         "/" + userID + "/cal/" + calendarID + "/" + objectID,
		ETag:         "etag-event-123",
		LastModified: now,
		Component:    []*ical.Component{eventComponent},
	}

	// Test cases
	tests := []struct {
		name           string
		resourceType   storage.ResourceType
		setupMocks     func()
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
			name:         "Object not found",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(nil, propfind.ErrNotFound).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "ETag match - Not modified",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(testEvent, nil).Once()
			},
			headers: map[string]string{
				"If-None-Match": "etag-event-123",
			},
			expectedStatus: http.StatusNotModified,
		},
		{
			name:         "Calendar collection not found",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(testEvent, nil).Once()
				mockStorage.On("GetCalendar", userID, calendarID).
					Return(nil, propfind.ErrNotFound).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "Successful GET",
			resourceType: storage.ResourceObject,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, objectID).
					Return(testEvent, nil).Once()
				mockStorage.On("GetCalendar", userID, calendarID).
					Return(testCalendar, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, "text/calendar; charset=utf-8", recorder.Header().Get("Content-Type"))
				assert.Equal(t, "etag-event-123", recorder.Header().Get("ETag"))
				assert.NotEmpty(t, recorder.Body.String())
				assert.Contains(t, recorder.Body.String(), "VEVENT")
				assert.Contains(t, recorder.Body.String(), "Test Event")
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockStorage.ExpectedCalls = nil

			// Setup mocks
			tt.setupMocks()

			// Create request
			req := httptest.NewRequest("GET", "/caldav/"+userID+"/cal/"+calendarID+"/"+objectID, nil)

			// Add headers if needed
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
			handler.handleGet(recorder, req, ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			// Additional response checks if needed
			if tt.checkResponse != nil && recorder.Code == http.StatusOK {
				tt.checkResponse(t, recorder)
			}

			// Verify all expectations were met
			mockStorage.AssertExpectations(t)
		})
	}
}
