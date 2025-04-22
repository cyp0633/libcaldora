package server

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical" // Import go-ical
	"github.com/stretchr/testify/assert"
)

func TestHandleCalendarMultiget(t *testing.T) {
	// Add log capturing
	var logOutput strings.Builder
	log.SetOutput(&logOutput)
	defer log.SetOutput(nil)

	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
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
		mockStorage.On("GetObject", "user1", "cal1", "event1").Return(&storage.CalendarObject{
			ETag: "etag1",
			Component: &ical.Component{ // Add a minimal non-nil component
				Name:  ical.CompEvent,
				Props: make(ical.Props),
			},
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
