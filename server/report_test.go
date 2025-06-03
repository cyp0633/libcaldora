package server

import (
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleCalendarMultiget(t *testing.T) {
	// Add log capturing
	var logOutput strings.Builder
	log.SetOutput(&logOutput)
	defer log.SetOutput(nil)

	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	// Create a test logger that writes to a discard handler
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
		Logger:       logger,
	}

	// Common resource for context (usually the collection where REPORT is sent)
	ctxResource := Resource{
		UserID:       "user1",
		CalendarID:   "cal1",
		ResourceType: storage.ResourceCollection,
	}
	ctx := &RequestContext{
		Resource: ctxResource,
	}

	t.Run("Multiget Object and Collection", func(t *testing.T) {
		// Resources requested in the multiget body
		objectPath := "/calendars/user1/cal1/event1.ics"
		collectionPath := "/calendars/user1/cal1/"

		objectResource := Resource{
			UserID:       "user1",
			CalendarID:   "cal1",
			ObjectID:     "event1",
			ResourceType: storage.ResourceObject,
			URI:          objectPath, // URI is often set during propfind handling
		}
		collectionResource := Resource{
			UserID:       "user1",
			CalendarID:   "cal1",
			ResourceType: storage.ResourceCollection,
		}

		// Mock URLConverter ParsePath calls - allow multiple calls as needed
		mockURLConverter.On("ParsePath", objectPath).Return(objectResource, nil).Once()
		mockURLConverter.On("ParsePath", collectionPath).Return(collectionResource, nil).Once()

		// Mock Storage calls needed by handlePropfindObject for objectResource
		comp := &ical.Component{
			Name:  ical.CompEvent,
			Props: make(ical.Props),
		}
		mockStorage.On("GetObject", "user1", "cal1", "event1").Return(&storage.CalendarObject{
			ETag:      "etag1",
			Component: []*ical.Component{comp}, // Add a minimal non-nil component
		}, nil).Once()

		// Mock Storage calls needed by handlePropfindCollection for collectionResource
		mockStorage.On("GetCalendar", "user1", "cal1").Return(&storage.Calendar{
			ETag: "etagCal1",
			CalendarData: &ical.Calendar{ // Ensure CalendarData is not nil if needed by props
				Component: &ical.Component{
					Name:  ical.CompCalendar,
					Props: make(ical.Props),
				},
			},
		}, nil).Once()

		// Mock URLConverter EncodePath needed by handlePropfindCollection
		mockURLConverter.On("EncodePath", collectionResource).Return(collectionPath, nil).Once()

		// --- Request Body ---
		reqBody := `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <D:resourcetype/>
  </D:prop>
  <D:href>` + objectPath + `</D:href>
  <D:href>` + collectionPath + `</D:href>
</C:calendar-multiget>`

		req := httptest.NewRequest("REPORT", "/calendars/user1/cal1/", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/xml")
		rr := httptest.NewRecorder()

		// Call the handler
		h.handleCalendarMultiget(rr, req, ctx)

		// Check logs to debug the issue
		t.Logf("Log output: %s", logOutput.String())

		// If the test is failing, you can temporarily use this to pass
		if rr.Code != http.StatusMultiStatus {
			t.Logf("Expected status 207, got %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())
		}

		// Assertions
		assert.Equal(t, http.StatusMultiStatus, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "application/xml")

		respBody := rr.Body.String()
		// Check for responses for both hrefs
		assert.Contains(t, respBody, "<d:href>"+objectPath+"</d:href>")
		assert.Contains(t, respBody, "<d:href>"+collectionPath+"</d:href>")
		// Check for requested properties (getetag and resourcetype) in both responses
		assert.Contains(t, respBody, "<d:getetag>etag1</d:getetag>")    // From object
		assert.Contains(t, respBody, "<d:collection/>")                 // From collection resourcetype
		assert.Contains(t, respBody, "<d:getetag>etagCal1</d:getetag>") // From collection

		// Verify mocks
		mockURLConverter.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Multiget with Principal and HomeSet", func(t *testing.T) {
		// Reset mocks for a clean run
		mockURLConverter = new(MockURLConverter)
		mockStorage = new(storage.MockStorage)
		h.URLConverter = mockURLConverter
		h.Storage = mockStorage

		principalPath := "/principals/user1/"
		homeSetPath := "/calendars/user1/"

		principalResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}
		homeSetResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		// Mock URLConverter ParsePath calls
		mockURLConverter.On("ParsePath", principalPath).Return(principalResource, nil).Maybe()
		mockURLConverter.On("ParsePath", homeSetPath).Return(homeSetResource, nil).Maybe()

		// Mock Storage calls needed by handlePropfindPrincipal
		mockStorage.On("GetUser", "user1").Return(&storage.User{DisplayName: "User One"}, nil).Maybe()
		// Mock URLConverter EncodePath needed by handlePropfindPrincipal
		mockURLConverter.On("EncodePath", principalResource).Return(principalPath, nil).Maybe() // Called multiple times

		// Mock URLConverter EncodePath needed by handlePropfindHomeSet
		mockURLConverter.On("EncodePath", homeSetResource).Return(homeSetPath, nil).Maybe()
		// Principal path needed again inside handlePropfindHomeSet
		mockURLConverter.On("EncodePath", principalResource).Return(principalPath, nil).Maybe() // Called multiple times

		// --- Request Body ---
		reqBody := `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:displayname/>
    <D:principal-url/>
  </D:prop>
  <D:href>` + principalPath + `</D:href>
  <D:href>` + homeSetPath + `</D:href>
</C:calendar-multiget>`

		req := httptest.NewRequest("REPORT", "/calendars/user1/", strings.NewReader(reqBody)) // Target doesn't strictly matter for multiget logic itself
		req.Header.Set("Content-Type", "application/xml")
		rr := httptest.NewRecorder()

		// Call the handler
		h.handleCalendarMultiget(rr, req, ctx)

		// Check logs to debug the issue
		t.Logf("Log output: %s", logOutput.String())

		// If the test is failing, you can temporarily use this to pass
		if rr.Code != http.StatusMultiStatus {
			t.Logf("Expected status 207, got %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())
		}

		// Assertions
		assert.Equal(t, http.StatusMultiStatus, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "application/xml")

		respBody := rr.Body.String()
		// Check for responses for both hrefs
		assert.Contains(t, respBody, "<d:href>"+principalPath+"</d:href>")
		assert.Contains(t, respBody, "<d:href>"+homeSetPath+"</d:href>")
		// Check for requested properties
		assert.Contains(t, respBody, "<d:displayname>User One</d:displayname>")                               // From principal
		assert.Contains(t, respBody, "<d:displayname>Calendar Home</d:displayname>")                          // From home set
		assert.Contains(t, respBody, "<d:principal-url><d:href>"+principalPath+"</d:href></d:principal-url>") // Both should return this

		// Verify mocks
		mockURLConverter.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Error parsing resource path", func(t *testing.T) {
		// Reset mocks
		mockURLConverter = new(MockURLConverter)
		mockStorage = new(storage.MockStorage)
		h.URLConverter = mockURLConverter
		h.Storage = mockStorage

		invalidPath := "/invalid/path"

		// Mock URLConverter ParsePath to return an error
		mockURLConverter.On("ParsePath", invalidPath).Return(Resource{}, assert.AnError).Maybe()

		// --- Request Body ---
		reqBody := `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:displayname/>
  </D:prop>
  <D:href>` + invalidPath + `</D:href>
</C:calendar-multiget>`

		req := httptest.NewRequest("REPORT", "/calendars/user1/", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/xml")
		rr := httptest.NewRecorder()

		// Call the handler
		h.handleCalendarMultiget(rr, req, ctx)

		// Check logs to debug the issue
		t.Logf("Log output: %s", logOutput.String())

		// If the test is failing, you can temporarily use this to pass
		if rr.Code != http.StatusInternalServerError {
			t.Logf("Expected status 500, got %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())
		}

		// Assertions
		assert.Equal(t, http.StatusInternalServerError, rr.Code) // Expecting internal server error as ParsePath failed

		// Verify mocks
		mockURLConverter.AssertExpectations(t)
		mockStorage.AssertExpectations(t) // No storage calls should have been made
	})

}

func TestHandleCalendarQuery(t *testing.T) {
	// Add log capturing
	var logOutput strings.Builder
	log.SetOutput(&logOutput)
	defer log.SetOutput(os.Stdout)

	// Setup
	mockURL := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	// Create a test logger that writes to a discard handler
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := &CaldavHandler{
		URLConverter: mockURL,
		Storage:      mockStorage,
		Logger:       logger,
	}

	cases := []struct {
		name                   string
		ctxResource            Resource
		requestBody            string
		setupMocks             func()
		expectStatus           int
		expectResponseContains []string // strings that should be in the response
	}{
		{
			name: "single object query",
			ctxResource: Resource{
				UserID:       "user1",
				CalendarID:   "cal1",
				ObjectID:     "event1",
				ResourceType: storage.ResourceObject,
				URI:          "/calendars/user1/cal1/event1.ics",
			},
			requestBody: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT"/>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`,
			setupMocks: func() {
				// Create an object that will match the filter
				// The component needs to be a VCALENDAR containing a VEVENT child
				eventComp := ical.NewComponent(ical.CompEvent)
				eventComp.Props.SetText("SUMMARY", "Test Event")

				calComp := ical.NewComponent(ical.CompCalendar)
				calComp.Children = append(calComp.Children, eventComp)

				object := &storage.CalendarObject{
					ETag:      "event1-etag",
					Component: []*ical.Component{calComp},
				}

				mockStorage.On("GetObject", "user1", "cal1", "event1").Return(object, nil).Once()

				// Make the EncodePath mock more flexible to match how handleCalendarQuery calls it
				mockURL.On("EncodePath", mock.Anything).Return("/calendars/user1/cal1/event1.ics", nil).Maybe()
			},
			expectStatus: http.StatusMultiStatus,
			expectResponseContains: []string{
				"/calendars/user1/cal1/event1.ics",
				"event1-etag",
				"HTTP/1.1 200 OK",
			},
		},
		{
			name: "object doesn't match filter",
			ctxResource: Resource{
				UserID:       "user1",
				CalendarID:   "cal1",
				ObjectID:     "event2",
				ResourceType: storage.ResourceObject,
				URI:          "/calendars/user1/cal1/event2.ics",
			},
			requestBody: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:prop-filter name="SUMMARY">
          <C:text-match>DoesNotExist</C:text-match>
        </C:prop-filter>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`,
			setupMocks: func() {
				// Create an object that won't match the filter
				comp := &ical.Component{
					Name: ical.CompEvent,
					Props: ical.Props{
						"SUMMARY": []ical.Prop{{Value: "DifferentSummary"}},
					},
				}
				object := &storage.CalendarObject{
					ETag:      "event2-etag",
					Component: []*ical.Component{comp},
				}
				mockStorage.On("GetObject", "user1", "cal1", "event2").Return(object, nil).Once()
			},
			expectStatus: http.StatusNotFound,
			expectResponseContains: []string{
				"Object does not match filter",
			},
		},
		{
			name: "collection query with matching objects",
			ctxResource: Resource{
				UserID:       "user1",
				CalendarID:   "cal1",
				ResourceType: storage.ResourceCollection,
				URI:          "/calendars/user1/cal1/",
			},
			requestBody: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT"/>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`,
			setupMocks: func() {
				// Objects that match the filter
				comp := &ical.Component{
					Name:  ical.CompEvent,
					Props: make(ical.Props),
				}
				objects := []storage.CalendarObject{
					{
						ETag:      "event1-etag",
						Component: []*ical.Component{comp},
					},
				}
				mockStorage.On("GetObjectByFilter", "user1", "cal1", mock.Anything).Return(objects, nil).Once()

				// For collection header URLs
				mockURL.On("EncodePath", mock.MatchedBy(func(res Resource) bool {
					return res.ResourceType == storage.ResourceCollection
				})).Return("/calendars/user1/cal1/", nil).Maybe()

				// For object URLs in the collection
				mockURL.On("EncodePath", mock.MatchedBy(func(res Resource) bool {
					return res.ResourceType == storage.ResourceObject
				})).Return("/calendars/user1/cal1/object.ics", nil).Maybe()
			},
			expectStatus: http.StatusMultiStatus,
			expectResponseContains: []string{
				"HTTP/1.1 200 OK",
				"event1-etag",
			},
		},
		{
			name: "unsupported resource type",
			ctxResource: Resource{
				UserID:       "user1",
				ResourceType: storage.ResourceHomeSet,
				URI:          "/calendars/user1/",
			},
			requestBody: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT"/>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`,
			setupMocks: func() {
				// No mocks needed, should return error directly
			},
			expectStatus: http.StatusBadRequest,
			expectResponseContains: []string{
				"Unsupported resource type for calendar-query",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// reset mocks and logs
			mockStorage.ExpectedCalls = nil
			mockStorage.Calls = nil
			mockURL.ExpectedCalls = nil
			mockURL.Calls = nil
			logOutput.Reset()

			// Overwrite context
			ctx := &RequestContext{Resource: tc.ctxResource}

			tc.setupMocks()

			req := httptest.NewRequest("REPORT", tc.ctxResource.URI, strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/xml")
			rr := httptest.NewRecorder()

			h.handleCalendarQuery(rr, req, ctx)

			// status code
			assert.Equal(t, tc.expectStatus, rr.Code)

			respBody := rr.Body.String()
			t.Logf("Response for %s: %s", tc.name, respBody)

			// Check all expected response content
			for _, expected := range tc.expectResponseContains {
				assert.Contains(t, respBody, expected, "Response should contain '%s'", expected)
			}

			// verify expectations
			mockURL.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}
