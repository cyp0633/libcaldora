package server

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Use storage types for testing
type Calendar = storage.Calendar
type CalendarObject = storage.CalendarObject

// Mock URL Converter
type MockURLConverter struct {
	mock.Mock
}

func (m *MockURLConverter) EncodePath(r Resource) (string, error) {
	args := m.Called(r)
	return args.String(0), args.Error(1)
}

func (m *MockURLConverter) ParsePath(path string) (Resource, error) {
	args := m.Called(path)
	return args.Get(0).(Resource), args.Error(1)
}

func TestHandlePropfindHomeSet(t *testing.T) {
	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	// Add a logger that writes to a discard handler
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
		Logger:       logger,
	}

	resource := Resource{
		UserID:       "user1",
		ResourceType: storage.ResourceHomeSet,
	}

	ctx := &RequestContext{
		Resource: resource,
	}

	// Test case 1: Basic properties
	t.Run("Basic properties", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		// Create request map with some basic properties using mo.Ok instead of mo.None
		req := propfind.ResponseMap{
			"displayname":                      mo.Ok[props.Property](nil),
			"calendar-home-set":                mo.Ok[props.Property](nil),
			"supported-calendar-component-set": mo.Ok[props.Property](nil),
			"calendar-user-type":               mo.Ok[props.Property](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Verify response has expected elements
		root := doc.Root()
		assert.NotNil(t, root)

		// Check if properties were properly set
		response := root.FindElement("//d:response")
		assert.NotNil(t, response)

		// Check href
		href := response.FindElement("//d:href")
		assert.NotNil(t, href)
		assert.Equal(t, "/calendars/user1/", href.Text())

		// Check calendar user type
		calUserType := response.FindElement("//d:prop/cal:calendar-user-type")
		assert.NotNil(t, calUserType)
		assert.Equal(t, "individual", calUserType.Text())

		// Check displayname
		displayName := response.FindElement("//d:prop/d:displayname")
		assert.NotNil(t, displayName)
		assert.Equal(t, "Calendar Home", displayName.Text())

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 2: Principal URL property
	t.Run("Principal URL property", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		principalResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}
		mockURLConverter.On("EncodePath", principalResource).Return("/principals/user1/", nil)

		// Create request map with principal-url property
		req := propfind.ResponseMap{
			"principal-url": mo.Ok[props.Property](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if principal-url was properly set
		root := doc.Root()
		principalURL := root.FindElement("//d:propstat/d:prop/d:principal-url/d:href")
		assert.NotNil(t, principalURL)
		assert.Equal(t, "/principals/user1/", principalURL.Text())

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 3: ACL property
	t.Run("ACL property", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		principalResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}
		mockURLConverter.On("EncodePath", principalResource).Return("/principals/user1/", nil)

		// Create request map with acl property
		req := propfind.ResponseMap{
			"acl": mo.Ok[props.Property](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if acl was properly set
		root := doc.Root()

		// Verify ACL contains read and write privileges
		ace := root.FindElement("//d:propstat/d:prop/d:acl/d:ace")
		assert.NotNil(t, ace)

		// Verify principal href
		principal := ace.FindElement("//d:principal/d:href")
		assert.NotNil(t, principal)
		assert.Equal(t, "/principals/user1/", principal.Text())

		// Verify grant privileges
		readPriv := ace.FindElement("//d:grant/d:privilege/d:read")
		assert.NotNil(t, readPriv)

		writePriv := ace.FindElement("//d:grant/d:privilege/d:write")
		assert.NotNil(t, writePriv)

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 4: Calendar limits properties
	t.Run("Calendar limits properties", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		// Create request map with calendar limits properties
		req := propfind.ResponseMap{
			"max-resource-size":          mo.Ok[props.Property](nil),
			"min-date-time":              mo.Ok[props.Property](nil),
			"max-date-time":              mo.Ok[props.Property](nil),
			"max-instances":              mo.Ok[props.Property](nil),
			"max-attendees-per-instance": mo.Ok[props.Property](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if properties were properly set
		root := doc.Root()

		maxSize := root.FindElement("//d:propstat/d:prop/cal:max-resource-size")
		assert.NotNil(t, maxSize)
		assert.Equal(t, "10485760", maxSize.Text())

		maxInstances := root.FindElement("//d:propstat/d:prop/cal:max-instances")
		assert.NotNil(t, maxInstances)
		assert.Equal(t, "100000", maxInstances.Text())

		maxAttendees := root.FindElement("//d:propstat/d:prop/cal:max-attendees-per-instance")
		assert.NotNil(t, maxAttendees)
		assert.Equal(t, "100", maxAttendees.Text())

		mockURLConverter.AssertExpectations(t)
	})
}

func TestFetchChildren(t *testing.T) {
	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	// Add a logger that writes to a discard handler
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
		Logger:       logger,
	}

	// Test case 1: Depth 0 returns empty
	t.Run("Depth 0", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		resources, err := h.fetchChildren(0, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 2: storage.ResourceObject has no children
	t.Run("Resource Object", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 3: storage.ResourcePrincipal has no children
	t.Run("Resource Principal", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 4: storage.ResourceCollection with children
	t.Run("Resource Collection", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		// Mock storage response
		mockStorage.On("GetObjectPathsInCollection", "cal1").Return([]string{
			"/calendars/user1/cal1/event1.ics",
			"/calendars/user1/cal1/event2.ics",
		}, nil)

		// Mock URL converter for each path
		event1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		event2Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event2",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event1.ics").Return(event1Resource, nil)
		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event2.ics").Return(event2Resource, nil)

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "event1", resources[0].ObjectID)
		assert.Equal(t, "event2", resources[1].ObjectID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})

	// Test case 5: storage.ResourceHomeSet with children
	t.Run("Resource HomeSet", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		// Mock storage response - using proper Calendar struct
		mockStorage.On("GetUserCalendars", "user1").Return([]Calendar{
			{Path: "/calendars/user1/cal1/"},
			{Path: "/calendars/user1/cal2/"},
		}, nil)

		// Mock URL converter for each path
		cal1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		cal2Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal2",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/").Return(cal1Resource, nil)
		mockURLConverter.On("ParsePath", "/calendars/user1/cal2/").Return(cal2Resource, nil)

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "cal1", resources[0].CalendarID)
		assert.Equal(t, "cal2", resources[1].CalendarID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})

	// Test case 6: Recursive fetching with depth > 1
	t.Run("Recursive fetching", func(t *testing.T) {
		// Setup fresh mocks for this specific test
		mockURLConverter := new(MockURLConverter)
		mockStorage := new(storage.MockStorage)

		h := &CaldavHandler{
			URLConverter: mockURLConverter,
			Storage:      mockStorage,
			Logger:       logger, // Add the logger here as well
		}

		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		// Mock HomeSet -> Calendar responses
		mockStorage.On("GetUserCalendars", "user1").Return([]Calendar{
			{Path: "/calendars/user1/cal1/"},
		}, nil)

		cal1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/").Return(cal1Resource, nil)

		// Mock Calendar -> Events responses
		mockStorage.On("GetObjectPathsInCollection", "cal1").Return([]string{
			"/calendars/user1/cal1/event1.ics",
		}, nil)

		event1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event1.ics").Return(event1Resource, nil)

		resources, err := h.fetchChildren(2, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "cal1", resources[0].CalendarID)
		assert.Equal(t, "event1", resources[1].ObjectID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})
}

func TestHandlePropfindMultiComponentCalendarData(t *testing.T) {
	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
		Logger:       logger,
	}

	userID := "alice"
	calendarID := "work"
	now := time.Now()

	// Create test calendar
	testCalendar := &storage.Calendar{
		Path:         "/" + userID + "/cal/" + calendarID,
		CTag:         "ctag-123",
		ETag:         "etag-cal-123",
		CalendarData: ical.NewCalendar(),
	}
	testCalendar.CalendarData.Props.SetText(ical.PropName, "Work Calendar")
	testCalendar.CalendarData.Props.SetText(ical.PropProductID, "-//libcaldora//NONSGML v1.0//EN")
	testCalendar.CalendarData.Props.SetText(ical.PropVersion, "2.0")

	tests := []struct {
		name          string
		objectID      string
		setupObject   func() *storage.CalendarObject
		checkResponse func(t *testing.T, doc *etree.Document)
	}{
		{
			name:     "PROPFIND calendar-data for VEVENT with VTIMEZONE",
			objectID: "event-with-timezone.ics",
			setupObject: func() *storage.CalendarObject {
				// Create VTIMEZONE component
				timezone := ical.NewComponent("VTIMEZONE")
				timezone.Props.SetText("TZID", "America/New_York")

				// Add STANDARD time definition
				standard := ical.NewComponent("STANDARD")
				standard.Props.SetText("DTSTART", "20071104T020000")
				standard.Props.SetText("TZOFFSETFROM", "-0400")
				standard.Props.SetText("TZOFFSETTO", "-0500")
				standard.Props.SetText("RRULE", "FREQ=YEARLY;BYMONTH=11;BYDAY=1SU")
				timezone.Children = append(timezone.Children, standard)

				// Create VEVENT component
				eventComponent := ical.NewComponent(ical.CompEvent)
				eventComponent.Props.SetText(ical.PropUID, "event-with-tz-1")
				eventComponent.Props.SetText(ical.PropSummary, "Meeting in NY timezone")
				eventComponent.Props.SetDateTime(ical.PropDateTimeStart, now)
				eventComponent.Props.SetDateTime(ical.PropDateTimeEnd, now.Add(1*time.Hour))
				eventComponent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				return &storage.CalendarObject{
					Path:         "/" + userID + "/cal/" + calendarID + "/event-with-timezone.ics",
					ETag:         "etag-tz-event-123",
					LastModified: now,
					Component:    []*ical.Component{timezone, eventComponent},
				}
			},
			checkResponse: func(t *testing.T, doc *etree.Document) {
				// Check that calendar-data is present and contains both components
				calData := doc.Root().FindElement("//cal:calendar-data")
				assert.NotNil(t, calData, "calendar-data element should be present")

				icsContent := calData.Text()
				assert.NotEmpty(t, icsContent, "calendar-data should not be empty")

				// Check for VTIMEZONE and VEVENT components
				assert.Contains(t, icsContent, "BEGIN:VTIMEZONE")
				assert.Contains(t, icsContent, "TZID:America/New_York")
				assert.Contains(t, icsContent, "BEGIN:VEVENT")
				assert.Contains(t, icsContent, "Meeting in NY timezone")
				assert.Contains(t, icsContent, "UID:event-with-tz-1")
			},
		},
		{
			name:     "PROPFIND calendar-data for VEVENT with override",
			objectID: "recurring-with-exception.ics",
			setupObject: func() *storage.CalendarObject {
				// Create main recurring VEVENT
				masterEvent := ical.NewComponent(ical.CompEvent)
				masterEvent.Props.SetText(ical.PropUID, "recurring-event-123")
				masterEvent.Props.SetText(ical.PropSummary, "Weekly Team Meeting")
				masterEvent.Props.SetDateTime(ical.PropDateTimeStart, now)
				masterEvent.Props.SetDateTime(ical.PropDateTimeEnd, now.Add(1*time.Hour))
				masterEvent.Props.SetDateTime(ical.PropDateTimeStamp, now)
				masterEvent.Props.SetText(ical.PropRecurrenceRule, "FREQ=WEEKLY;BYDAY=TU")

				// Create exception/override VEVENT
				exceptionEvent := ical.NewComponent(ical.CompEvent)
				exceptionEvent.Props.SetText(ical.PropUID, "recurring-event-123")
				exceptionEvent.Props.SetText(ical.PropSummary, "Weekly Team Meeting - RESCHEDULED")
				exceptionDate := now.AddDate(0, 0, 7)
				exceptionEvent.Props.SetDateTime("RECURRENCE-ID", exceptionDate)
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeStart, exceptionDate.Add(2*time.Hour))
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeEnd, exceptionDate.Add(3*time.Hour))
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				return &storage.CalendarObject{
					Path:         "/" + userID + "/cal/" + calendarID + "/recurring-with-exception.ics",
					ETag:         "etag-recurring-exception-123",
					LastModified: now,
					Component:    []*ical.Component{masterEvent, exceptionEvent},
				}
			},
			checkResponse: func(t *testing.T, doc *etree.Document) {
				calData := doc.Root().FindElement("//cal:calendar-data")
				assert.NotNil(t, calData, "calendar-data element should be present")

				icsContent := calData.Text()
				assert.NotEmpty(t, icsContent, "calendar-data should not be empty")

				// Check for both master and exception events
				assert.Contains(t, icsContent, "UID:recurring-event-123")
				assert.Contains(t, icsContent, "Weekly Team Meeting")
				assert.Contains(t, icsContent, "Weekly Team Meeting - RESCHEDULED")
				assert.Contains(t, icsContent, "RRULE")
				assert.Contains(t, icsContent, "RECURRENCE-ID")

				// Verify we have two VEVENT sections
				eventCount := strings.Count(icsContent, "BEGIN:VEVENT")
				assert.Equal(t, 2, eventCount, "Should have 2 VEVENT components")
			},
		},
		{
			name:     "PROPFIND calendar-data for complex multi-component",
			objectID: "complex-multi.ics",
			setupObject: func() *storage.CalendarObject {
				// Create VTIMEZONE with proper structure
				timezone := ical.NewComponent("VTIMEZONE")
				timezone.Props.SetText("TZID", "Europe/London")

				// Add STANDARD time definition (required by go-ical)
				standard := ical.NewComponent("STANDARD")
				standard.Props.SetText("DTSTART", "20071028T020000")
				standard.Props.SetText("TZOFFSETFROM", "+0100")
				standard.Props.SetText("TZOFFSETTO", "+0000")
				standard.Props.SetText("RRULE", "FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU")
				timezone.Children = append(timezone.Children, standard)

				// Create VEVENT
				eventComponent := ical.NewComponent(ical.CompEvent)
				eventComponent.Props.SetText(ical.PropUID, "complex-event-1")
				eventComponent.Props.SetText(ical.PropSummary, "Project Meeting")
				eventComponent.Props.SetDateTime(ical.PropDateTimeStart, now)
				eventComponent.Props.SetDateTime(ical.PropDateTimeEnd, now.Add(1*time.Hour))
				eventComponent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				// Create VTODO
				todoComponent := ical.NewComponent(ical.CompToDo)
				todoComponent.Props.SetText(ical.PropUID, "complex-todo-1")
				todoComponent.Props.SetText(ical.PropSummary, "Follow up on meeting")
				todoComponent.Props.SetDateTime(ical.PropDue, now.Add(24*time.Hour))
				todoComponent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				return &storage.CalendarObject{
					Path:         "/" + userID + "/cal/" + calendarID + "/complex-multi.ics",
					ETag:         "etag-complex-123",
					LastModified: now,
					Component:    []*ical.Component{timezone, eventComponent, todoComponent},
				}
			},
			checkResponse: func(t *testing.T, doc *etree.Document) {
				calData := doc.Root().FindElement("//cal:calendar-data")
				assert.NotNil(t, calData, "calendar-data element should be present")

				icsContent := calData.Text()
				assert.NotEmpty(t, icsContent, "calendar-data should not be empty")

				// Check for all three component types
				assert.Contains(t, icsContent, "BEGIN:VTIMEZONE")
				assert.Contains(t, icsContent, "TZID:Europe/London")
				assert.Contains(t, icsContent, "BEGIN:VEVENT")
				assert.Contains(t, icsContent, "Project Meeting")
				assert.Contains(t, icsContent, "UID:complex-event-1")
				assert.Contains(t, icsContent, "BEGIN:VTODO")
				assert.Contains(t, icsContent, "Follow up on meeting")
				assert.Contains(t, icsContent, "UID:complex-todo-1")
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockStorage.ExpectedCalls = nil
			mockURLConverter.ExpectedCalls = nil

			// Setup test object
			testObject := tt.setupObject()

			// Setup mocks
			objectResource := Resource{
				UserID:       userID,
				CalendarID:   calendarID,
				ObjectID:     tt.objectID,
				ResourceType: storage.ResourceObject,
			}

			mockStorage.On("GetObject", userID, calendarID, tt.objectID).
				Return(testObject, nil).Once()
			mockURLConverter.On("EncodePath", objectResource).
				Return(testObject.Path, nil).Once()

			// Create request map for calendar-data property
			req := propfind.ResponseMap{
				"calendar-data": mo.Ok[props.Property](nil),
				"getetag":       mo.Ok[props.Property](nil),
			}

			// Call function
			doc, err := h.handlePropfindObject(req, objectResource)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, doc)

			// Check basic response structure
			root := doc.Root()
			assert.NotNil(t, root)

			response := root.FindElement("//d:response")
			assert.NotNil(t, response)

			href := response.FindElement("//d:href")
			assert.NotNil(t, href)
			assert.Equal(t, testObject.Path, href.Text())

			// Check ETag
			etag := response.FindElement("//d:getetag")
			assert.NotNil(t, etag)
			assert.Equal(t, testObject.ETag, etag.Text())

			// Additional response checks
			tt.checkResponse(t, doc)

			// Verify all expectations were met
			mockStorage.AssertExpectations(t)
			mockURLConverter.AssertExpectations(t)
		})
	}
}
