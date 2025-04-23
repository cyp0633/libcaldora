package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleMkCalendar(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Create handler with mock storage
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil)

	// Setup test data
	userID := "alice"
	calendarID := "work"
	calendarPath := "/alice/cal/work/"

	// Test cases
	tests := []struct {
		name           string
		resourceType   storage.ResourceType
		xmlBody        string
		setupMocks     func()
		expectedStatus int
		expectedETag   string
		expectedPath   string
	}{
		{
			name:           "Non-collection resource",
			resourceType:   storage.ResourceObject,
			xmlBody:        "",
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:         "Basic calendar creation",
			resourceType: storage.ResourceCollection,
			xmlBody: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Work Calendar</D:displayname>
      <C:calendar-description>Calendar for work-related events</C:calendar-description>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
        <C:comp name="VTODO"/>
      </C:supported-calendar-component-set>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			setupMocks: func() {
				// Storage should modify the Calendar by setting ETag and Path
				mockStorage.On("CreateCalendar", userID, mock.AnythingOfType("*storage.Calendar")).
					Run(func(args mock.Arguments) {
						cal := args.Get(1).(*storage.Calendar)
						cal.ETag = "etag-cal-123"
						cal.Path = calendarPath
					}).
					Return(nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedETag:   "etag-cal-123",
			expectedPath:   calendarPath,
		},
		{
			name:         "Calendar with Apple extensions",
			resourceType: storage.ResourceCollection,
			xmlBody: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/">
  <D:set>
    <D:prop>
      <D:displayname>Personal</D:displayname>
      <CS:calendar-color>#FF0000</CS:calendar-color>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			setupMocks: func() {
				mockStorage.On("CreateCalendar", userID, mock.AnythingOfType("*storage.Calendar")).
					Run(func(args mock.Arguments) {
						cal := args.Get(1).(*storage.Calendar)
						cal.ETag = "etag-cal-apple"
						cal.Path = calendarPath
					}).
					Return(nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedETag:   "etag-cal-apple",
			expectedPath:   calendarPath,
		},
		{
			name:         "Storage error",
			resourceType: storage.ResourceCollection,
			xmlBody: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Error Calendar</D:displayname>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			setupMocks: func() {
				mockStorage.On("CreateCalendar", userID, mock.AnythingOfType("*storage.Calendar")).
					Return(storage.ErrInvalidInput).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "Path not set by storage",
			resourceType: storage.ResourceCollection,
			xmlBody: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Incomplete Calendar</D:displayname>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			setupMocks: func() {
				// Storage sets ETag but not Path (which is incorrect behavior)
				mockStorage.On("CreateCalendar", userID, mock.AnythingOfType("*storage.Calendar")).
					Run(func(args mock.Arguments) {
						cal := args.Get(1).(*storage.Calendar)
						cal.ETag = "etag-incomplete"
						// Path intentionally not set
					}).
					Return(nil).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "ETag not set by storage",
			resourceType: storage.ResourceCollection,
			xmlBody: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Incomplete Calendar</D:displayname>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			setupMocks: func() {
				// Storage sets Path but not ETag (which is incorrect behavior)
				mockStorage.On("CreateCalendar", userID, mock.AnythingOfType("*storage.Calendar")).
					Run(func(args mock.Arguments) {
						cal := args.Get(1).(*storage.Calendar)
						// ETag intentionally not set
						cal.Path = calendarPath
					}).
					Return(nil).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Invalid XML",
			resourceType:   storage.ResourceCollection,
			xmlBody:        "This is not XML",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
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
			req := httptest.NewRequest("MKCALENDAR", "/caldav/"+userID+"/cal/"+calendarID+"/", bytes.NewBufferString(tt.xmlBody))
			req.Header.Set("Content-Type", "application/xml")

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Create context
			ctx := &RequestContext{
				Resource: Resource{
					UserID:       userID,
					CalendarID:   calendarID,
					ResourceType: tt.resourceType,
				},
				AuthUser: userID,
			}

			// Call handler
			handler.handleMkCalendar(recorder, req, ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			// Check headers for successful responses
			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, tt.expectedETag, recorder.Header().Get("ETag"))
				assert.Equal(t, tt.expectedPath, recorder.Header().Get("Location"))
			}

			// Verify all expectations were met
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestHandleMkCalendarPropertyHandling(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Create handler with mock storage
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil)

	// Test calendar with all supported properties
	mockStorage.On("CreateCalendar", "alice", mock.AnythingOfType("*storage.Calendar")).
		Run(func(args mock.Arguments) {
			cal := args.Get(1).(*storage.Calendar)

			// Verify calendar properties were set correctly
			displayName, _ := cal.CalendarData.Props.Text(ical.PropName)
			assert.Equal(t, "Full Test Calendar", displayName)

			description, _ := cal.CalendarData.Props.Text(ical.PropDescription)
			assert.Equal(t, "Calendar with all properties", description)

			color, _ := cal.CalendarData.Props.Text(ical.PropColor)
			assert.Equal(t, "#4A86E8", color)

			// Verify supported components
			assert.Contains(t, cal.SupportedComponents, "VEVENT")
			assert.Contains(t, cal.SupportedComponents, "VTODO")
			assert.Equal(t, 2, len(cal.SupportedComponents))

			// Verify timezone component exists
			hasTimezone := false
			for _, child := range cal.CalendarData.Children {
				if child.Name == ical.CompTimezone {
					hasTimezone = true
					break
				}
			}
			assert.True(t, hasTimezone, "Calendar should have VTIMEZONE component")

			// Set required fields
			cal.ETag = "etag-full-test"
			cal.Path = "/alice/cal/full/"
		}).
		Return(nil).Once()

	// Create request with all properties
	fullXML := `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" 
              xmlns:D="DAV:"
              xmlns:CS="http://calendarserver.org/ns/"
              xmlns:G="http://schemas.google.com/gCal/2005">
  <D:set>
    <D:prop>
      <D:displayname>Full Test Calendar</D:displayname>
      <C:calendar-description>Calendar with all properties</C:calendar-description>
      <C:calendar-timezone>BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Example Corp.//CalDAV Client//EN\r\nBEGIN:VTIMEZONE\r\nTZID:America/New_York\r\nEND:VTIMEZONE\r\nEND:VCALENDAR</C:calendar-timezone>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
        <C:comp name="VTODO"/>
      </C:supported-calendar-component-set>
      <CS:calendar-color>#4A86E8</CS:calendar-color>
      <G:timezone>America/New_York</G:timezone>
    </D:prop>
  </D:set>
</C:mkcalendar>`

	req := httptest.NewRequest("MKCALENDAR", "/caldav/alice/cal/full/", strings.NewReader(fullXML))
	req.Header.Set("Content-Type", "application/xml")
	recorder := httptest.NewRecorder()

	ctx := &RequestContext{
		Resource: Resource{
			UserID:       "alice",
			CalendarID:   "full",
			ResourceType: storage.ResourceCollection,
		},
		AuthUser: "alice",
	}

	// Call handler
	handler.handleMkCalendar(recorder, req, ctx)

	// Assert response
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "etag-full-test", recorder.Header().Get("ETag"))
	assert.Equal(t, "/alice/cal/full/", recorder.Header().Get("Location"))

	// Verify all expectations were met
	mockStorage.AssertExpectations(t)
}
