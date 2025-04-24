package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/google/uuid"
)

// MemoryStorage implements storage.Storage interface using in-memory maps
type MemoryStorage struct {
	// Mutex to protect concurrent access
	mu sync.RWMutex

	// Users map: userID -> User
	users map[string]storage.User

	// Calendars map: userID -> calendarID -> Calendar
	calendars map[string]map[string]storage.Calendar

	// Objects map: userID -> calendarID -> objectID -> CalendarObject
	objects map[string]map[string]map[string]storage.CalendarObject
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		users:     make(map[string]storage.User),
		calendars: make(map[string]map[string]storage.Calendar),
		objects:   make(map[string]map[string]map[string]storage.CalendarObject),
	}
}

// RegisterUser adds a new user to the storage
func (m *MemoryStorage) RegisterUser(userID string, displayName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[userID] = storage.User{
		DisplayName:       displayName,
		UserAddress:       fmt.Sprintf("mailto:%s@example.com", userID),
		PreferredColor:    "#4285F4", // Default blue color
		PreferredTimezone: "UTC",
	}

	// Initialize maps for the user
	m.calendars[userID] = make(map[string]storage.Calendar)
	m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
}

// GetUser gets user information
func (m *MemoryStorage) GetUser(userID string) (*storage.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, storage.ErrNotFound
	}
	return &user, nil
}

// GetUserCalendars retrieves all calendar collections for a user
func (m *MemoryStorage) GetUserCalendars(userID string) ([]storage.Calendar, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		return nil, storage.ErrNotFound
	}

	// Check if user has any calendars
	userCalendars, exists := m.calendars[userID]
	if !exists {
		return []storage.Calendar{}, nil
	}

	// Convert map to slice
	calendars := make([]storage.Calendar, 0, len(userCalendars))
	for _, cal := range userCalendars {
		// Create a copy to avoid modifying the original
		calCopy := cal
		calendars = append(calendars, calCopy)
	}

	return calendars, nil
}

// GetCalendar retrieves a specific calendar by user id and calendar id
func (m *MemoryStorage) GetCalendar(userID, calendarID string) (*storage.Calendar, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userCalendars, exists := m.calendars[userID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calendar, exists := userCalendars[calendarID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return &calendar, nil
}

// CreateCalendar creates a new calendar collection
func (m *MemoryStorage) CreateCalendar(userID string, calendar *storage.Calendar) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		return storage.ErrNotFound
	}

	// Ensure the user's calendar map exists
	if _, exists := m.calendars[userID]; !exists {
		m.calendars[userID] = make(map[string]storage.Calendar)
	}

	// Extract calendar ID from path (this is a simplification)
	pathParts := pathSplit(calendar.Path)
	if len(pathParts) < 3 {
		return storage.ErrInvalidInput
	}

	calendarID := pathParts[2] // Assuming path format: "/userID/cal/calendarID/"

	// Check if calendar already exists
	if _, exists := m.calendars[userID][calendarID]; exists {
		return storage.ErrConflict
	}

	// Set ETag if not already set
	if calendar.ETag == "" {
		calendar.ETag = fmt.Sprintf("etag-calendar-%s", uuid.New().String())
	}

	// Set path if not already set
	if calendar.Path == "" {
		calendar.Path = fmt.Sprintf("/%s/cal/%s/", userID, calendarID)
	}

	// Create calendar
	m.calendars[userID][calendarID] = *calendar

	// Initialize map for calendar objects
	if _, exists := m.objects[userID]; !exists {
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)

	return nil
}

// GetObjectsInCollection retrieves all calendar objects in a given calendar collection
func (m *MemoryStorage) GetObjectsInCollection(calendarID string) ([]storage.CalendarObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for userID, userCals := range m.calendars {
		for calID := range userCals {
			if calID == calendarID {
				// Check if objects exist for this calendar
				userObjs, exists := m.objects[userID]
				if !exists {
					return []storage.CalendarObject{}, nil
				}

				calObjs, exists := userObjs[calendarID]
				if !exists {
					return []storage.CalendarObject{}, nil
				}

				// Convert map to slice
				objects := make([]storage.CalendarObject, 0, len(calObjs))
				for _, obj := range calObjs {
					objects = append(objects, obj)
				}

				return objects, nil
			}
		}
	}

	return nil, storage.ErrNotFound
}

// GetObjectPathsInCollection retrieves paths of all calendar objects in a given calendar collection
func (m *MemoryStorage) GetObjectPathsInCollection(calendarID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for userID, userCals := range m.calendars {
		for calID := range userCals {
			if calID == calendarID {
				// Check if objects exist for this calendar
				userObjs, exists := m.objects[userID]
				if !exists {
					return []string{}, nil
				}

				calObjs, exists := userObjs[calendarID]
				if !exists {
					return []string{}, nil
				}

				// Extract paths
				paths := make([]string, 0, len(calObjs))
				for _, obj := range calObjs {
					paths = append(paths, obj.Path)
				}

				return paths, nil
			}
		}
	}

	return nil, storage.ErrNotFound
}

// GetObject finds a calendar object by user id, calendar id and object id
func (m *MemoryStorage) GetObject(userID, calendarID, objectID string) (*storage.CalendarObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	// Check if object exists
	obj, exists := calObjs[objectID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return &obj, nil
}

// GetObjectByFilter finds calendar objects by filter
func (m *MemoryStorage) GetObjectByFilter(userID, calendarID string, filter *storage.Filter) ([]storage.CalendarObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		return nil, storage.ErrNotFound
	}

	// Convert map to slice
	objects := make([]storage.CalendarObject, 0, len(calObjs))
	for _, obj := range calObjs {
		// Skip if object doesn't match the filter
		if filter != nil && !filter.Validate(&obj) {
			continue
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

// UpdateObject updates a calendar object, or creates one if it doesn't exist
func (m *MemoryStorage) UpdateObject(userID, calendarID string, object *storage.CalendarObject) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		return "", storage.ErrNotFound
	}

	// Check if calendar exists
	userCals, exists := m.calendars[userID]
	if !exists || userCals[calendarID].Path == "" {
		return "", storage.ErrNotFound
	}

	// Ensure the user's objects map hierarchy exists
	if _, exists := m.objects[userID]; !exists {
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	if _, exists := m.objects[userID][calendarID]; !exists {
		m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)
	}

	// Extract object ID from path
	pathParts := pathSplit(object.Path)
	if len(pathParts) < 4 {
		return "", storage.ErrInvalidInput
	}

	objectID := pathParts[3] // Assuming path format: "/userID/cal/calendarID/objectID.ics"

	// Generate new ETag if not provided
	if object.ETag == "" {
		object.ETag = fmt.Sprintf("etag-%s-%d", uuid.New().String(), time.Now().Unix())
	}

	// Update LastModified
	object.LastModified = time.Now()

	// Store the object
	m.objects[userID][calendarID][objectID] = *object

	// Update the calendar's CTag
	cal := userCals[calendarID]
	cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
	m.calendars[userID][calendarID] = cal

	return object.ETag, nil
}

// DeleteObject removes a calendar object
func (m *MemoryStorage) DeleteObject(userID, calendarID, objectID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		return storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		return storage.ErrNotFound
	}

	// Check if object exists
	if _, exists := calObjs[objectID]; !exists {
		return storage.ErrNotFound
	}

	// Delete the object
	delete(m.objects[userID][calendarID], objectID)

	// Update the calendar's CTag
	userCals := m.calendars[userID]
	cal := userCals[calendarID]
	cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
	m.calendars[userID][calendarID] = cal

	return nil
}

// Helper function to split a path
func pathSplit(path string) []string {
	var parts []string
	current := ""

	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// AddEvent adds a calendar object to a specific calendar
func (m *MemoryStorage) AddEvent(userID, calendarID string, event storage.CalendarObject) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract object ID from path
	pathParts := pathSplit(event.Path)
	if len(pathParts) < 4 {
		return
	}

	objectID := pathParts[3] // Assuming path format: "/userID/cal/calendarID/objectID.ics"

	// Ensure the user's objects map hierarchy exists
	if _, exists := m.objects[userID]; !exists {
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	if _, exists := m.objects[userID][calendarID]; !exists {
		m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)
	}

	// Store the event
	m.objects[userID][calendarID][objectID] = event

	// Update the calendar's CTag
	if userCals, exists := m.calendars[userID]; exists {
		if cal, exists := userCals[calendarID]; exists {
			cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
			m.calendars[userID][calendarID] = cal
		}
	}
}
