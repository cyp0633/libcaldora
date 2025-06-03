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
	comp := ical.NewComponent(ical.CompEvent)
	comp.Props.SetText(ical.PropUID, "event-uid-1")
	existingEvent := &storage.CalendarObject{
		Path:         encodedPath,
		ETag:         "etag-event-123",
		LastModified: now,
		Component:    []*ical.Component{comp},
	}

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

func TestHandlePutMultiComponentObjects(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Setup test data
	userID := "alice"
	calendarID := "work"
	encodedPath := "/" + userID + "/cal/" + calendarID

	// Create handler with mock storage and URL converter
	urlConverter := &mockURLConverter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, urlConverter, logger)

	now := time.Now()

	tests := []struct {
		name           string
		objectID       string
		body           string
		setupMocks     func()
		expectedStatus int
		checkResponse  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "PUT VEVENT with VTIMEZONE",
			objectID: "event-with-timezone.ics",
			body: `BEGIN:VCALENDAR
PRODID:-//libcaldora//NONSGML v1.0//EN
VERSION:2.0
BEGIN:VTIMEZONE
TZID:America/New_York
BEGIN:STANDARD
DTSTART:20071104T020000
TZOFFSETFROM:-0400
TZOFFSETTO:-0500
RRULE:FREQ=YEARLY;BYMONTH=11;BYDAY=1SU
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:20070311T020000
TZOFFSETFROM:-0500
TZOFFSETTO:-0400
RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=2SU
END:DAYLIGHT
END:VTIMEZONE
BEGIN:VEVENT
UID:event-with-tz-1
SUMMARY:Meeting in NY timezone
DESCRIPTION:A meeting that uses timezone information
LOCATION:New York Office
DTSTART;TZID=America/New_York:` + now.Format("20060102T150405") + `
DTEND;TZID=America/New_York:` + now.Add(1*time.Hour).Format("20060102T150405") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, "event-with-timezone.ics").
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == "event-with-timezone.ics"
				})).Return(encodedPath+"/event-with-timezone.ics", nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.MatchedBy(func(obj *storage.CalendarObject) bool {
					// Verify that the object contains both VTIMEZONE and VEVENT components
					if len(obj.Component) != 2 {
						return false
					}
					hasTimezone := false
					hasEvent := false
					for _, comp := range obj.Component {
						if comp.Name == "VTIMEZONE" {
							hasTimezone = true
							if tzid, err := comp.Props.Text("TZID"); err != nil || tzid != "America/New_York" {
								return false
							}
						}
						if comp.Name == ical.CompEvent {
							hasEvent = true
							if uid, err := comp.Props.Text(ical.PropUID); err != nil || uid != "event-with-tz-1" {
								return false
							}
						}
					}
					return hasTimezone && hasEvent
				})).Return("new-etag-tz-123", nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, encodedPath+"/event-with-timezone.ics", recorder.Header().Get("Location"))
				assert.Equal(t, "new-etag-tz-123", recorder.Header().Get("ETag"))
			},
		},
		{
			name:     "PUT VEVENT with override (master + exception)",
			objectID: "recurring-with-exception.ics",
			body: `BEGIN:VCALENDAR
PRODID:-//libcaldora//NONSGML v1.0//EN
VERSION:2.0
BEGIN:VEVENT
UID:recurring-event-123
SUMMARY:Weekly Team Meeting
DESCRIPTION:Regular weekly team sync
DTSTART:` + now.Format("20060102T150405Z") + `
DTEND:` + now.Add(1*time.Hour).Format("20060102T150405Z") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
RRULE:FREQ=WEEKLY;BYDAY=TU
END:VEVENT
BEGIN:VEVENT
UID:recurring-event-123
SUMMARY:Weekly Team Meeting - RESCHEDULED
DESCRIPTION:Rescheduled team meeting due to holiday
LOCATION:Conference Room B
RECURRENCE-ID:` + now.AddDate(0, 0, 7).Format("20060102T150405Z") + `
DTSTART:` + now.AddDate(0, 0, 7).Add(2*time.Hour).Format("20060102T150405Z") + `
DTEND:` + now.AddDate(0, 0, 7).Add(3*time.Hour).Format("20060102T150405Z") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, "recurring-with-exception.ics").
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == "recurring-with-exception.ics"
				})).Return(encodedPath+"/recurring-with-exception.ics", nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.MatchedBy(func(obj *storage.CalendarObject) bool {
					// Verify that the object contains both master and exception events
					if len(obj.Component) != 2 {
						return false
					}
					hasMaster := false
					hasException := false
					for _, comp := range obj.Component {
						if comp.Name == ical.CompEvent {
							if uid, err := comp.Props.Text(ical.PropUID); err != nil || uid != "recurring-event-123" {
								return false
							}
							// Check if this is the master (has RRULE) or exception (has RECURRENCE-ID)
							if comp.Props.Get(ical.PropRecurrenceRule) != nil {
								hasMaster = true
							}
							if comp.Props.Get("RECURRENCE-ID") != nil {
								hasException = true
							}
						}
					}
					return hasMaster && hasException
				})).Return("new-etag-recurring-123", nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, encodedPath+"/recurring-with-exception.ics", recorder.Header().Get("Location"))
				assert.Equal(t, "new-etag-recurring-123", recorder.Header().Get("ETag"))
			},
		},
		{
			name:     "PUT complex multi-component: VTIMEZONE + VEVENT + VTODO",
			objectID: "complex-multi.ics",
			body: `BEGIN:VCALENDAR
PRODID:-//libcaldora//NONSGML v1.0//EN
VERSION:2.0
BEGIN:VTIMEZONE
TZID:Europe/London
BEGIN:STANDARD
DTSTART:20071028T020000
TZOFFSETFROM:+0100
TZOFFSETTO:+0000
RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:complex-event-1
SUMMARY:Project Meeting
DTSTART;TZID=Europe/London:` + now.Format("20060102T150405") + `
DTEND;TZID=Europe/London:` + now.Add(1*time.Hour).Format("20060102T150405") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
BEGIN:VTODO
UID:complex-todo-1
SUMMARY:Follow up on meeting
DUE;TZID=Europe/London:` + now.Add(24*time.Hour).Format("20060102T150405") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VTODO
END:VCALENDAR`,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, "complex-multi.ics").
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == "complex-multi.ics"
				})).Return(encodedPath+"/complex-multi.ics", nil).Once()
				mockStorage.On("UpdateObject", userID, calendarID, mock.MatchedBy(func(obj *storage.CalendarObject) bool {
					// Verify that the object contains VTIMEZONE, VEVENT, and VTODO
					if len(obj.Component) != 3 {
						return false
					}
					hasTimezone := false
					hasEvent := false
					hasTodo := false
					for _, comp := range obj.Component {
						switch comp.Name {
						case "VTIMEZONE":
							hasTimezone = true
							if tzid, err := comp.Props.Text("TZID"); err != nil || tzid != "Europe/London" {
								return false
							}
						case ical.CompEvent:
							hasEvent = true
							if uid, err := comp.Props.Text(ical.PropUID); err != nil || uid != "complex-event-1" {
								return false
							}
						case ical.CompToDo:
							hasTodo = true
							if uid, err := comp.Props.Text(ical.PropUID); err != nil || uid != "complex-todo-1" {
								return false
							}
						}
					}
					return hasTimezone && hasEvent && hasTodo
				})).Return("new-etag-complex-123", nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, encodedPath+"/complex-multi.ics", recorder.Header().Get("Location"))
				assert.Equal(t, "new-etag-complex-123", recorder.Header().Get("ETag"))
			},
		},
		{
			name:     "PUT invalid multi-component: multiple VEVENTs with different UIDs",
			objectID: "invalid-multi.ics",
			body: `BEGIN:VCALENDAR
PRODID:-//libcaldora//NONSGML v1.0//EN
VERSION:2.0
BEGIN:VEVENT
UID:event-1
SUMMARY:First Event
DTSTART:` + now.Format("20060102T150405Z") + `
DTEND:` + now.Add(1*time.Hour).Format("20060102T150405Z") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
BEGIN:VEVENT
UID:event-2
SUMMARY:Second Event
DTSTART:` + now.Add(2*time.Hour).Format("20060102T150405Z") + `
DTEND:` + now.Add(3*time.Hour).Format("20060102T150405Z") + `
DTSTAMP:` + now.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`,
			setupMocks: func() {
				mockStorage.On("GetObject", userID, calendarID, "invalid-multi.ics").
					Return(nil, storage.ErrNotFound).Once()
				urlConverter.On("EncodePath", mock.MatchedBy(func(r Resource) bool {
					return r.UserID == userID && r.CalendarID == calendarID && r.ObjectID == "invalid-multi.ics"
				})).Return(encodedPath+"/invalid-multi.ics", nil).Once()
				// The storage should still accept it - CalDAV allows multiple components with different UIDs
				mockStorage.On("UpdateObject", userID, calendarID, mock.MatchedBy(func(obj *storage.CalendarObject) bool {
					return len(obj.Component) == 2 // Two VEVENT components
				})).Return("new-etag-multi-events-123", nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, encodedPath+"/invalid-multi.ics", recorder.Header().Get("Location"))
				assert.Equal(t, "new-etag-multi-events-123", recorder.Header().Get("ETag"))
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
			req := httptest.NewRequest("PUT", "/caldav/"+userID+"/cal/"+calendarID+"/"+tt.objectID,
				strings.NewReader(tt.body))

			// Add headers
			req.Header.Set("Content-Type", "text/calendar")

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Create context
			ctx := &RequestContext{
				Resource: Resource{
					UserID:       userID,
					CalendarID:   calendarID,
					ObjectID:     tt.objectID,
					ResourceType: storage.ResourceObject,
				},
				AuthUser: userID,
			}

			// Call handler
			handler.handlePut(recorder, req, ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			// Additional response checks if needed
			if tt.checkResponse != nil && recorder.Code == tt.expectedStatus {
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
