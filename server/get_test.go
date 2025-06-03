package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestHandleGetMultiComponentObjects(t *testing.T) {
	// Create a mock storage
	mockStorage := &storage.MockStorage{}

	// Create handler with mock storage
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 1, nil, logger)

	// Setup test data
	userID := "alice"
	calendarID := "work"

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

	now := time.Now()

	tests := []struct {
		name           string
		objectID       string
		setupObject    func() *storage.CalendarObject
		expectedStatus int
		checkResponse  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "VEVENT with VTIMEZONE",
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

				// Add DAYLIGHT time definition
				daylight := ical.NewComponent("DAYLIGHT")
				daylight.Props.SetText("DTSTART", "20070311T020000")
				daylight.Props.SetText("TZOFFSETFROM", "-0500")
				daylight.Props.SetText("TZOFFSETTO", "-0400")
				daylight.Props.SetText("RRULE", "FREQ=YEARLY;BYMONTH=3;BYDAY=2SU")
				timezone.Children = append(timezone.Children, daylight)

				// Create VEVENT component
				eventComponent := ical.NewComponent(ical.CompEvent)
				eventComponent.Props.SetText(ical.PropUID, "event-with-tz-1")
				eventComponent.Props.SetText(ical.PropSummary, "Meeting in NY timezone")
				eventComponent.Props.SetText(ical.PropDescription, "A meeting that uses timezone information")
				eventComponent.Props.SetText(ical.PropLocation, "New York Office")

				// Set start/end times with timezone reference
				startProp := &ical.Prop{
					Name:   ical.PropDateTimeStart,
					Value:  now.Format("20060102T150405"),
					Params: ical.Params{"TZID": []string{"America/New_York"}},
				}
				eventComponent.Props[ical.PropDateTimeStart] = []ical.Prop{*startProp}

				endProp := &ical.Prop{
					Name:   ical.PropDateTimeEnd,
					Value:  now.Add(1 * time.Hour).Format("20060102T150405"),
					Params: ical.Params{"TZID": []string{"America/New_York"}},
				}
				eventComponent.Props[ical.PropDateTimeEnd] = []ical.Prop{*endProp}

				eventComponent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				return &storage.CalendarObject{
					Path:         "/" + userID + "/cal/" + calendarID + "/event-with-timezone.ics",
					ETag:         "etag-tz-event-123",
					LastModified: now,
					Component:    []*ical.Component{timezone, eventComponent}, // Both VTIMEZONE and VEVENT
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, "text/calendar; charset=utf-8", recorder.Header().Get("Content-Type"))
				assert.Equal(t, "etag-tz-event-123", recorder.Header().Get("ETag"))

				body := recorder.Body.String()
				assert.NotEmpty(t, body)

				// Check for both VTIMEZONE and VEVENT components
				assert.Contains(t, body, "BEGIN:VTIMEZONE")
				assert.Contains(t, body, "TZID:America/New_York")
				assert.Contains(t, body, "BEGIN:STANDARD")
				assert.Contains(t, body, "BEGIN:DAYLIGHT")
				assert.Contains(t, body, "END:VTIMEZONE")

				assert.Contains(t, body, "BEGIN:VEVENT")
				assert.Contains(t, body, "Meeting in NY timezone")
				assert.Contains(t, body, "TZID=America/New_York")
				assert.Contains(t, body, "UID:event-with-tz-1")
				assert.Contains(t, body, "END:VEVENT")
			},
		},
		{
			name:     "VEVENT with override (RECURRENCE-ID)",
			objectID: "recurring-event-with-exception.ics",
			setupObject: func() *storage.CalendarObject {
				// Create main recurring VEVENT
				masterEvent := ical.NewComponent(ical.CompEvent)
				masterEvent.Props.SetText(ical.PropUID, "recurring-event-123")
				masterEvent.Props.SetText(ical.PropSummary, "Weekly Team Meeting")
				masterEvent.Props.SetText(ical.PropDescription, "Regular weekly team sync")
				masterEvent.Props.SetDateTime(ical.PropDateTimeStart, now)
				masterEvent.Props.SetDateTime(ical.PropDateTimeEnd, now.Add(1*time.Hour))
				masterEvent.Props.SetDateTime(ical.PropDateTimeStamp, now)
				masterEvent.Props.SetText(ical.PropRecurrenceRule, "FREQ=WEEKLY;BYDAY=TU")

				// Create exception/override VEVENT with RECURRENCE-ID
				exceptionEvent := ical.NewComponent(ical.CompEvent)
				exceptionEvent.Props.SetText(ical.PropUID, "recurring-event-123") // Same UID
				exceptionEvent.Props.SetText(ical.PropSummary, "Weekly Team Meeting - RESCHEDULED")
				exceptionEvent.Props.SetText(ical.PropDescription, "Rescheduled team meeting due to holiday")
				exceptionEvent.Props.SetText(ical.PropLocation, "Conference Room B")

				// Set RECURRENCE-ID to indicate this is an exception
				exceptionDate := now.AddDate(0, 0, 7) // One week later
				exceptionEvent.Props.SetDateTime("RECURRENCE-ID", exceptionDate)

				// Different time for the exception
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeStart, exceptionDate.Add(2*time.Hour))
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeEnd, exceptionDate.Add(3*time.Hour))
				exceptionEvent.Props.SetDateTime(ical.PropDateTimeStamp, now)

				return &storage.CalendarObject{
					Path:         "/" + userID + "/cal/" + calendarID + "/recurring-event-with-exception.ics",
					ETag:         "etag-recurring-exception-123",
					LastModified: now,
					Component:    []*ical.Component{masterEvent, exceptionEvent}, // Master + Exception
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, "text/calendar; charset=utf-8", recorder.Header().Get("Content-Type"))
				assert.Equal(t, "etag-recurring-exception-123", recorder.Header().Get("ETag"))

				body := recorder.Body.String()
				assert.NotEmpty(t, body)

				// Check for two VEVENT components with same UID but different summaries
				assert.Contains(t, body, "BEGIN:VEVENT")
				assert.Contains(t, body, "Weekly Team Meeting")
				assert.Contains(t, body, "Weekly Team Meeting - RESCHEDULED")
				// The RRULE gets escaped in output, so look for the escaped version
				assert.Contains(t, body, "RRULE;VALUE=TEXT:FREQ=WEEKLY\\;BYDAY=TU")
				// RECURRENCE-ID gets its own line
				assert.Contains(t, body, "RECURRENCE-ID;TZID=Local:")
				assert.Contains(t, body, "UID:recurring-event-123")

				// Verify we have the master event and exception
				masterFound := false
				exceptionFound := false
				for _, line := range strings.Split(body, "\n") {
					if strings.Contains(line, "SUMMARY:Weekly Team Meeting") && !strings.Contains(line, "RESCHEDULED") {
						masterFound = true
					}
					if strings.Contains(line, "SUMMARY:Weekly Team Meeting - RESCHEDULED") {
						exceptionFound = true
					}
				}
				assert.True(t, masterFound, "Master event should be present")
				assert.True(t, exceptionFound, "Exception event should be present")
			},
		},
		{
			name:     "Complex multi-component: VEVENT + VTIMEZONE + VTODO",
			objectID: "complex-multi.ics",
			setupObject: func() *storage.CalendarObject {
				// Create VTIMEZONE
				timezone := ical.NewComponent("VTIMEZONE")
				timezone.Props.SetText("TZID", "Europe/London")
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
					Component:    []*ical.Component{timezone, eventComponent, todoComponent}, // All three types
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, "text/calendar; charset=utf-8", recorder.Header().Get("Content-Type"))
				assert.Equal(t, "etag-complex-123", recorder.Header().Get("ETag"))

				body := recorder.Body.String()
				assert.NotEmpty(t, body)

				// Check for all three component types
				assert.Contains(t, body, "BEGIN:VTIMEZONE")
				assert.Contains(t, body, "TZID:Europe/London")
				assert.Contains(t, body, "END:VTIMEZONE")

				assert.Contains(t, body, "BEGIN:VEVENT")
				assert.Contains(t, body, "Project Meeting")
				assert.Contains(t, body, "UID:complex-event-1")
				assert.Contains(t, body, "END:VEVENT")

				assert.Contains(t, body, "BEGIN:VTODO")
				assert.Contains(t, body, "Follow up on meeting")
				assert.Contains(t, body, "UID:complex-todo-1")
				assert.Contains(t, body, "END:VTODO")
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockStorage.ExpectedCalls = nil

			// Setup test object
			testObject := tt.setupObject()

			// Setup mocks
			mockStorage.On("GetObject", userID, calendarID, tt.objectID).
				Return(testObject, nil).Once()
			mockStorage.On("GetCalendar", userID, calendarID).
				Return(testCalendar, nil).Once()

			// Create request
			req := httptest.NewRequest("GET", "/caldav/"+userID+"/cal/"+calendarID+"/"+tt.objectID, nil)

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
