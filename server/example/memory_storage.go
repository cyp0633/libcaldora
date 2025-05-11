package main

import (
	"fmt"
	"log/slog"
	"os"
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

	// Logger
	log *slog.Logger
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	// Create a logger for storage operations
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &MemoryStorage{
		users:     make(map[string]storage.User),
		calendars: make(map[string]map[string]storage.Calendar),
		objects:   make(map[string]map[string]map[string]storage.CalendarObject),
		log:       logger,
	}
}

// RegisterUser adds a new user to the storage
func (m *MemoryStorage) RegisterUser(userID string, displayName string) {
	m.log.Debug("Registering user", "userID", userID, "displayName", displayName)

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

	m.log.Info("User registered successfully", "userID", userID)
}

// GetUser gets user information
func (m *MemoryStorage) GetUser(userID string) (*storage.User, error) {
	m.log.Debug("Getting user information", "userID", userID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		m.log.Warn("User not found", "userID", userID)
		return nil, storage.ErrNotFound
	}

	m.log.Debug("User retrieved", "userID", userID, "displayName", user.DisplayName)
	return &user, nil
}

// GetUserCalendars retrieves all calendar collections for a user
func (m *MemoryStorage) GetUserCalendars(userID string) ([]storage.Calendar, error) {
	m.log.Debug("Getting calendars for user", "userID", userID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		m.log.Warn("User not found when retrieving calendars", "userID", userID)
		return nil, storage.ErrNotFound
	}

	// Check if user has any calendars
	userCalendars, exists := m.calendars[userID]
	if !exists {
		m.log.Debug("No calendars found for user", "userID", userID)
		return []storage.Calendar{}, nil
	}

	// Convert map to slice
	calendars := make([]storage.Calendar, 0, len(userCalendars))
	for calID, cal := range userCalendars {
		// Create a copy to avoid modifying the original
		calCopy := cal
		calendars = append(calendars, calCopy)
		m.log.Debug("Added calendar to result", "userID", userID, "calendarID", calID, "path", cal.Path)
	}

	m.log.Info("Retrieved calendars for user", "userID", userID, "count", len(calendars))
	return calendars, nil
}

// GetCalendar retrieves a specific calendar by user id and calendar id
func (m *MemoryStorage) GetCalendar(userID, calendarID string) (*storage.Calendar, error) {
	m.log.Debug("Getting calendar", "userID", userID, "calendarID", calendarID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userCalendars, exists := m.calendars[userID]
	if !exists {
		m.log.Warn("User not found when retrieving calendar", "userID", userID, "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calendar, exists := userCalendars[calendarID]
	if !exists {
		m.log.Warn("Calendar not found", "userID", userID, "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	m.log.Debug("Calendar retrieved", "userID", userID, "calendarID", calendarID, "path", calendar.Path)
	return &calendar, nil
}

// CreateCalendar creates a new calendar collection
func (m *MemoryStorage) CreateCalendar(userID string, calendar *storage.Calendar) error {
	m.log.Debug("Creating calendar", "userID", userID, "calendarPath", calendar.Path)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		m.log.Warn("User not found when creating calendar", "userID", userID)
		return storage.ErrNotFound
	}

	// Ensure the user's calendar map exists
	if _, exists := m.calendars[userID]; !exists {
		m.calendars[userID] = make(map[string]storage.Calendar)
	}

	// Extract calendar ID from path (this is a simplification)
	pathParts := pathSplit(calendar.Path)
	if len(pathParts) < 3 {
		m.log.Error("Invalid calendar path", "path", calendar.Path)
		return storage.ErrInvalidInput
	}

	calendarID := pathParts[3] // Assuming path format: "/caldav/userID/cal/calendarID/"
	m.log.Debug("Extracted calendarID from path", "calendarID", calendarID)

	// Check if calendar already exists
	if _, exists := m.calendars[userID][calendarID]; exists {
		m.log.Warn("Calendar already exists", "userID", userID, "calendarID", calendarID)
		return storage.ErrConflict
	}

	// Set ETag if not already set
	if calendar.ETag == "" {
		calendar.ETag = fmt.Sprintf("etag-calendar-%s", uuid.New().String())
		m.log.Debug("Generated new ETag for calendar", "ETag", calendar.ETag)
	}

	// Set path if not already set
	if calendar.Path == "" {
		calendar.Path = fmt.Sprintf("/%s/cal/%s/", userID, calendarID)
		m.log.Debug("Set calendar path", "path", calendar.Path)
	}

	// Create calendar
	m.calendars[userID][calendarID] = *calendar

	// Initialize map for calendar objects
	if _, exists := m.objects[userID]; !exists {
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)

	m.log.Info("Calendar created successfully", "userID", userID, "calendarID", calendarID, "path", calendar.Path)
	return nil
}

// GetObjectsInCollection retrieves all calendar objects in a given calendar collection
func (m *MemoryStorage) GetObjectsInCollection(calendarID string) ([]storage.CalendarObject, error) {
	m.log.Debug("Getting objects in collection", "calendarID", calendarID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	foundCalendar := false

	for userID, userCals := range m.calendars {
		for calID := range userCals {
			if calID == calendarID {
				foundCalendar = true
				m.log.Debug("Found calendar in user's collections", "calendarID", calendarID, "userID", userID)

				// Check if objects exist for this calendar
				userObjs, exists := m.objects[userID]
				if !exists {
					m.log.Debug("No objects map for user", "userID", userID)
					return []storage.CalendarObject{}, nil
				}

				calObjs, exists := userObjs[calendarID]
				if !exists {
					m.log.Debug("Empty calendar collection", "calendarID", calendarID)
					return []storage.CalendarObject{}, nil
				}

				// Convert map to slice
				objects := make([]storage.CalendarObject, 0, len(calObjs))
				for objID, obj := range calObjs {
					objects = append(objects, obj)
					m.log.Debug("Added object to result", "objectID", objID, "path", obj.Path)
				}

				m.log.Info("Retrieved objects in collection", "calendarID", calendarID, "count", len(objects))
				return objects, nil
			}
		}
	}

	if !foundCalendar {
		m.log.Warn("Calendar collection not found", "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	return []storage.CalendarObject{}, nil
}

// GetObjectPathsInCollection retrieves paths of all calendar objects in a given calendar collection
func (m *MemoryStorage) GetObjectPathsInCollection(calendarID string) ([]string, error) {
	m.log.Debug("Getting object paths in collection", "calendarID", calendarID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	foundCalendar := false

	for userID, userCals := range m.calendars {
		for calID := range userCals {
			if calID == calendarID {
				foundCalendar = true
				m.log.Debug("Found calendar in user's collections", "calendarID", calendarID, "userID", userID)

				// Check if objects exist for this calendar
				userObjs, exists := m.objects[userID]
				if !exists {
					m.log.Debug("No objects map for user", "userID", userID)
					return []string{}, nil
				}

				calObjs, exists := userObjs[calendarID]
				if !exists {
					m.log.Debug("Empty calendar collection", "calendarID", calendarID)
					return []string{}, nil
				}

				// Extract paths
				paths := make([]string, 0, len(calObjs))
				for objID, obj := range calObjs {
					paths = append(paths, obj.Path)
					m.log.Debug("Added path to result", "objectID", objID, "path", obj.Path)
				}

				m.log.Info("Retrieved object paths in collection", "calendarID", calendarID, "count", len(paths))
				return paths, nil
			}
		}
	}

	if !foundCalendar {
		m.log.Warn("Calendar collection not found", "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	return []string{}, nil
}

// GetObject finds a calendar object by user id, calendar id and object id
func (m *MemoryStorage) GetObject(userID, calendarID, objectID string) (*storage.CalendarObject, error) {
	m.log.Debug("Getting object", "userID", userID, "calendarID", calendarID, "objectID", objectID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		m.log.Warn("User not found when retrieving object", "userID", userID)
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		m.log.Warn("Calendar not found when retrieving object",
			"userID", userID, "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	// Check if object exists
	obj, exists := calObjs[objectID]
	if !exists {
		m.log.Warn("Object not found",
			"userID", userID, "calendarID", calendarID, "objectID", objectID)
		return nil, storage.ErrNotFound
	}

	m.log.Debug("Object retrieved successfully",
		"userID", userID, "calendarID", calendarID, "objectID", objectID, "etag", obj.ETag)
	return &obj, nil
}

// GetObjectByFilter finds calendar objects by filter
func (m *MemoryStorage) GetObjectByFilter(userID, calendarID string, filter *storage.Filter) ([]storage.CalendarObject, error) {
	filterDesc := "nil"
	if filter != nil {
		filterDesc = "present"
	}
	m.log.Debug("Getting objects by filter",
		"userID", userID, "calendarID", calendarID, "filter", filterDesc)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		m.log.Warn("User not found when filtering objects", "userID", userID)
		return nil, storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		m.log.Warn("Calendar not found when filtering objects",
			"userID", userID, "calendarID", calendarID)
		return nil, storage.ErrNotFound
	}

	// Convert map to slice
	objects := make([]storage.CalendarObject, 0, len(calObjs))
	matchCount := 0
	for objID, obj := range calObjs {
		// Skip if object doesn't match the filter
		if filter != nil && !filter.Validate(&obj) {
			m.log.Debug("Object doesn't match filter",
				"userID", userID, "calendarID", calendarID, "objectID", objID)
			continue
		}
		objects = append(objects, obj)
		matchCount++
		m.log.Debug("Object matches filter criteria",
			"userID", userID, "calendarID", calendarID, "objectID", objID)
	}

	m.log.Info("Filtered objects",
		"userID", userID, "calendarID", calendarID,
		"totalObjects", len(calObjs), "matchingObjects", matchCount)
	return objects, nil
}

// UpdateObject updates a calendar object, or creates one if it doesn't exist
func (m *MemoryStorage) UpdateObject(userID, calendarID string, object *storage.CalendarObject) (string, error) {
	m.log.Debug("Updating object",
		"userID", userID, "calendarID", calendarID, "objectPath", object.Path)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		m.log.Warn("User not found when updating object", "userID", userID)
		return "", storage.ErrNotFound
	}

	// Check if calendar exists
	userCals, exists := m.calendars[userID]
	if !exists || userCals[calendarID].Path == "" {
		m.log.Warn("Calendar not found when updating object",
			"userID", userID, "calendarID", calendarID)
		return "", storage.ErrNotFound
	}

	// Ensure the user's objects map hierarchy exists
	if _, exists := m.objects[userID]; !exists {
		m.log.Debug("Creating objects map for user", "userID", userID)
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	if _, exists := m.objects[userID][calendarID]; !exists {
		m.log.Debug("Creating objects map for calendar",
			"userID", userID, "calendarID", calendarID)
		m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)
	}

	// Extract object ID from path
	pathParts := pathSplit(object.Path)
	if len(pathParts) < 4 {
		m.log.Error("Invalid object path format", "path", object.Path, "parts", len(pathParts))
		return "", storage.ErrInvalidInput
	}

	objectID := pathParts[4] // Assuming path format: "/caldav/userID/cal/calendarID/objectID.ics"
	m.log.Debug("Extracted objectID from path", "objectID", objectID)

	oldETag := ""
	// Check if object already exists
	if existingObj, exists := m.objects[userID][calendarID][objectID]; exists {
		oldETag = existingObj.ETag
		m.log.Debug("Updating existing object",
			"userID", userID, "calendarID", calendarID,
			"objectID", objectID, "oldETag", oldETag)
	} else {
		m.log.Debug("Creating new object",
			"userID", userID, "calendarID", calendarID, "objectID", objectID)
	}

	// Generate new ETag if not provided
	if object.ETag == "" {
		object.ETag = fmt.Sprintf("etag-%s-%d", uuid.New().String(), time.Now().Unix())
		m.log.Debug("Generated new ETag for object", "ETag", object.ETag)
	}

	// Update LastModified
	object.LastModified = time.Now()

	// Store the object
	m.objects[userID][calendarID][objectID] = *object

	// Update the calendar's CTag
	oldCTag := userCals[calendarID].CTag
	cal := userCals[calendarID]
	cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
	m.calendars[userID][calendarID] = cal
	m.log.Debug("Updated calendar CTag",
		"userID", userID, "calendarID", calendarID,
		"oldCTag", oldCTag, "newCTag", cal.CTag)

	m.log.Info("Object updated successfully",
		"userID", userID, "calendarID", calendarID,
		"objectID", objectID, "etag", object.ETag)
	return object.ETag, nil
}

// DeleteObject removes a calendar object
func (m *MemoryStorage) DeleteObject(userID, calendarID, objectID string) error {
	m.log.Debug("Deleting object",
		"userID", userID, "calendarID", calendarID, "objectID", objectID)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	userObjs, exists := m.objects[userID]
	if !exists {
		m.log.Warn("User not found when deleting object", "userID", userID)
		return storage.ErrNotFound
	}

	// Check if calendar exists
	calObjs, exists := userObjs[calendarID]
	if !exists {
		m.log.Warn("Calendar not found when deleting object",
			"userID", userID, "calendarID", calendarID)
		return storage.ErrNotFound
	}

	// Check if object exists
	obj, exists := calObjs[objectID]
	if !exists {
		m.log.Warn("Object not found when deleting",
			"userID", userID, "calendarID", calendarID, "objectID", objectID)
		return storage.ErrNotFound
	}
	m.log.Debug("Found object to delete",
		"userID", userID, "calendarID", calendarID, "objectID", objectID, "etag", obj.ETag)

	// Delete the object
	delete(m.objects[userID][calendarID], objectID)

	// Update the calendar's CTag
	userCals := m.calendars[userID]
	oldCTag := userCals[calendarID].CTag
	cal := userCals[calendarID]
	cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
	m.calendars[userID][calendarID] = cal
	m.log.Debug("Updated calendar CTag",
		"userID", userID, "calendarID", calendarID,
		"oldCTag", oldCTag, "newCTag", cal.CTag)

	m.log.Info("Object deleted successfully",
		"userID", userID, "calendarID", calendarID, "objectID", objectID)
	return nil
}

// AddEvent adds a calendar object to a specific calendar
func (m *MemoryStorage) AddEvent(userID, calendarID string, event storage.CalendarObject) {
	m.log.Debug("Adding event to calendar", "userID", userID, "calendarID", calendarID, "eventPath", event.Path)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract object ID from path
	pathParts := pathSplit(event.Path)
	if len(pathParts) < 4 {
		m.log.Error("Invalid event path format", "path", event.Path, "parts", len(pathParts))
		return
	}

	objectID := pathParts[4] // Assuming path format: "/caldav/userID/cal/calendarID/objectID.ics"
	m.log.Debug("Extracted objectID from path", "objectID", objectID)

	// Check if user exists
	if _, exists := m.users[userID]; !exists {
		m.log.Warn("User not found when adding event", "userID", userID)
		return
	}

	// Check if calendar exists
	if _, exists := m.calendars[userID][calendarID]; !exists {
		m.log.Warn("Calendar not found when adding event", "userID", userID, "calendarID", calendarID)
		return
	}

	// Ensure the user's objects map hierarchy exists
	if _, exists := m.objects[userID]; !exists {
		m.log.Debug("Creating objects map for user", "userID", userID)
		m.objects[userID] = make(map[string]map[string]storage.CalendarObject)
	}
	if _, exists := m.objects[userID][calendarID]; !exists {
		m.log.Debug("Creating objects map for calendar", "userID", userID, "calendarID", calendarID)
		m.objects[userID][calendarID] = make(map[string]storage.CalendarObject)
	}

	// Check if event already exists (for logging purposes)
	if _, exists := m.objects[userID][calendarID][objectID]; exists {
		m.log.Info("Updating existing event", "userID", userID, "calendarID", calendarID, "objectID", objectID)
	} else {
		m.log.Info("Creating new event", "userID", userID, "calendarID", calendarID, "objectID", objectID)
	}

	// Store the event
	m.objects[userID][calendarID][objectID] = event

	// Update the calendar's CTag
	if userCals, exists := m.calendars[userID]; exists {
		if cal, exists := userCals[calendarID]; exists {
			oldCTag := cal.CTag
			cal.CTag = fmt.Sprintf("ctag-%s-%d", calendarID, time.Now().Unix())
			m.calendars[userID][calendarID] = cal
			m.log.Debug("Updated calendar CTag", "userID", userID, "calendarID", calendarID,
				"oldCTag", oldCTag, "newCTag", cal.CTag)
		}
	}

	m.log.Info("Event added successfully", "userID", userID, "calendarID", calendarID,
		"objectID", objectID, "etag", event.ETag)
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

// AuthUser authenticates a user with username and password, returns the user ID if successful.
func (m *MemoryStorage) AuthUser(username, password string) (string, error) {
	m.log.Debug("Authenticating user", "username", username)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// In this example implementation, we just check if the user exists
	// and if the password is "password" (for demonstration purposes)
	user, exists := m.users[username]
	if !exists {
		m.log.Warn("User not found during authentication", "username", username)
		return "", storage.ErrNotFound
	}

	// For this example, we accept any password
	// In a real implementation, you would verify the password hash
	if password == "" {
		m.log.Warn("Empty password provided", "username", username)
		return "", storage.ErrNotFound
	}

	m.log.Info("User authenticated successfully", "username", username, "displayName", user.DisplayName)
	return username, nil
}
