package storage

// MockStorage is a mock implementation of the Storage interface for testing.
type MockStorage struct {
	// Track method calls for verification in tests
	GetEventsInCollectionCalls []string
	GetUserCalendarsCalls      []string

	// Mock responses
	MockEvents   CalendarObject
	MockCalendar Calendar

	// Optional errors to return
	EventsError   error
	CalendarError error
}

// GetEventsInCollection implements Storage.GetEventsInCollection for testing.
func (m *MockStorage) GetEventsInCollection(calendarID string) (CalendarObject, error) {
	m.GetEventsInCollectionCalls = append(m.GetEventsInCollectionCalls, calendarID)
	return m.MockEvents, m.EventsError
}

// GetUserCalendars implements Storage.GetUserCalendars for testing.
func (m *MockStorage) GetUserCalendars(userID string) (Calendar, error) {
	m.GetUserCalendarsCalls = append(m.GetUserCalendarsCalls, userID)
	return m.MockCalendar, m.CalendarError
}

// ClearCalls resets all tracked method calls.
func (m *MockStorage) ClearCalls() {
	m.GetEventsInCollectionCalls = nil
	m.GetUserCalendarsCalls = nil
}

// NewMockStorage creates a new MockStorage with default empty values.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		GetEventsInCollectionCalls: make([]string, 0),
		GetUserCalendarsCalls:      make([]string, 0),
	}
}
