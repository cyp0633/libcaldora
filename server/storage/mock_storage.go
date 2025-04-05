package storage

import (
	"time"

	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/mock"
)

// MockStorage implements the Storage interface for testing
type MockStorage struct {
	mock.Mock
}

// GetObjectsInCollection implements the Storage interface
func (m *MockStorage) GetObjectsInCollection(calendarID string) ([]CalendarObject, error) {
	args := m.Called(calendarID)
	return args.Get(0).([]CalendarObject), args.Error(1)
}

// GetObjectPathsInCollection implements the Storage interface
func (m *MockStorage) GetObjectPathsInCollection(calendarID string) ([]string, error) {
	args := m.Called(calendarID)
	return args.Get(0).([]string), args.Error(1)
}

// GetUserCalendars implements the Storage interface
func (m *MockStorage) GetUserCalendars(userID string) ([]Calendar, error) {
	args := m.Called(userID)
	return args.Get(0).([]Calendar), args.Error(1)
}

func (m *MockStorage) GetUser(userID string) (*User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	user := args.Get(0).(*User)
	if user == nil {
		return nil, args.Error(1)
	}
	return user, args.Error(1)
}

func (m *MockStorage) GetCalendar(userID, calendarID string) (*Calendar, error) {
	args := m.Called(userID, calendarID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cal := args.Get(0).(*Calendar)
	if cal == nil {
		return nil, args.Error(1)
	}
	return cal, args.Error(1)
}

// --- Helper methods for creating test data ---

// NewMockCalendar creates a test Calendar with basic properties
func NewMockCalendar(path, name, description string) Calendar {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropName, name)
	cal.Props.SetText(ical.PropDescription, description)
	cal.Props.SetText("X-APPLE-CALENDAR-COLOR", "#FF9500") // Example color

	return Calendar{
		Path:         path,
		CTag:         "ctag-" + path + "-1",
		ETag:         "etag-" + path + "-1",
		CalendarData: cal,
	}
}

// NewMockEvent creates a test VEVENT calendar object
func NewMockEvent(path, uid, summary string, start, end time.Time) CalendarObject {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetText(ical.PropSummary, summary)
	event.Props.SetDateTime(ical.PropDateTimeStart, start)
	event.Props.SetDateTime(ical.PropDateTimeEnd, end)

	return CalendarObject{
		Path:         path,
		ETag:         "etag-" + uid + "-1",
		LastModified: time.Now(),
		Component:    event,
	}
}

// NewMockTodo creates a test VTODO calendar object
func NewMockTodo(path, uid, summary string, due time.Time) CalendarObject {
	todo := ical.NewComponent(ical.CompToDo)
	todo.Props.SetText(ical.PropUID, uid)
	todo.Props.SetText(ical.PropSummary, summary)
	todo.Props.SetDateTime(ical.PropDue, due)

	return CalendarObject{
		Path:         path,
		ETag:         "etag-" + uid + "-1",
		LastModified: time.Now(),
		Component:    todo,
	}
}

// --- Convenience methods for setting up common test scenarios ---

// SetupBasicUserWithCalendars populates the mock with a user who has multiple calendars
func (m *MockStorage) SetupBasicUserWithCalendars(userID string) {
	// Create calendars
	calendars := []Calendar{
		NewMockCalendar("/"+userID+"/cal/default", "Default", "Default Calendar"),
		NewMockCalendar("/"+userID+"/cal/work", "Work", "Work Calendar"),
	}

	// Setup GetUserCalendars expectation
	m.On("GetUserCalendars", userID).Return(calendars, nil)

	// Setup empty calendars by default
	m.On("GetObjectsInCollection", "default").Return([]CalendarObject{}, nil)
	m.On("GetObjectPathsInCollection", "default").Return([]string{}, nil)
	m.On("GetObjectsInCollection", "work").Return([]CalendarObject{}, nil)
	m.On("GetObjectPathsInCollection", "work").Return([]string{}, nil)
}

// AddEvents adds mock events to a calendar
func (m *MockStorage) AddEvents(calendarID string, events []CalendarObject) {
	paths := make([]string, len(events))
	for i, event := range events {
		paths[i] = event.Path
	}

	// Override any existing expectations
	m.ExpectedCalls = removeMatchingCalls(m.ExpectedCalls, "GetObjectsInCollection", calendarID)
	m.ExpectedCalls = removeMatchingCalls(m.ExpectedCalls, "GetObjectPathsInCollection", calendarID)

	// Set new expectations
	m.On("GetObjectsInCollection", calendarID).Return(events, nil)
	m.On("GetObjectPathsInCollection", calendarID).Return(paths, nil)
}

// Helper to remove existing mock calls that match a method and first argument
func removeMatchingCalls(calls []*mock.Call, method string, firstArg interface{}) []*mock.Call {
	result := make([]*mock.Call, 0, len(calls))
	for _, call := range calls {
		if call.Method == method && len(call.Arguments) > 0 && call.Arguments[0] == firstArg {
			continue
		}
		result = append(result, call)
	}
	return result
}

/*
// Example usage:
func TestCalendarHandler(t *testing.T) {
    mockStorage := &MockStorage{}

    // Setup a test user with calendars
    mockStorage.SetupBasicUserWithCalendars("alice")

    // Add some events to the work calendar
    now := time.Now()
    events := []CalendarObject{
        NewMockEvent("/alice/cal/work/event1.ics", "event-uid-1", "Team Meeting",
            now, now.Add(1*time.Hour)),
        NewMockEvent("/alice/cal/work/event2.ics", "event-uid-2", "Project Review",
            now.Add(2*time.Hour), now.Add(3*time.Hour)),
    }
    mockStorage.AddEvents("work", events)

    // Create your handler with the mock storage
    handler := caldav.NewCaldavHandler("/caldav/", "Test Realm", mockStorage, 3, nil)

    // Now you can test your handler with HTTP requests
    // ...
}
*/
