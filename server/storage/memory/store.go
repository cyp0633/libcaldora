// memory based implementation for testing purposes
package memory

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
)

// Store implements storage.Storage interface using in-memory maps
type Store struct {
	mu        sync.RWMutex
	users     map[string]*storage.User
	calendars map[string]*storage.Calendar       // key: userID/calendarID
	objects   map[string]*storage.CalendarObject // key: userID/objectID
	props     map[string]storage.Properties      // key: resource path
}

// New creates a new in-memory storage
func New() *Store {
	return &Store{
		users:     make(map[string]*storage.User),
		calendars: make(map[string]*storage.Calendar),
		objects:   make(map[string]*storage.CalendarObject),
		props:     make(map[string]storage.Properties),
	}
}

func (s *Store) calendarKey(userID, calendarID string) string {
	return fmt.Sprintf("%s/%s", userID, calendarID)
}

func (s *Store) objectKey(userID, objectID string) string {
	return fmt.Sprintf("%s/%s", userID, objectID)
}

func generateETag(data []byte) string {
	hash := sha1.Sum(data)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// User operations

func (s *Store) GetUser(_ context.Context, userID string) (*storage.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok {
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "user not found",
		}
	}

	return user, nil
}

// Calendar operations

func (s *Store) GetCalendar(_ context.Context, userID, calendarID string) (*storage.Calendar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cal, ok := s.calendars[s.calendarKey(userID, calendarID)]
	if !ok {
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	return cal, nil
}

func (s *Store) ListCalendars(_ context.Context, userID string) ([]*storage.Calendar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var calendars []*storage.Calendar
	for _, cal := range s.calendars {
		if cal.UserID == userID {
			calendars = append(calendars, cal)
		}
	}

	return calendars, nil
}

func (s *Store) CreateCalendar(_ context.Context, cal *storage.Calendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(cal.UserID, cal.ID)
	if _, exists := s.calendars[key]; exists {
		return &storage.Error{
			Type:    storage.ErrAlreadyExists,
			Message: "calendar already exists",
		}
	}

	now := time.Now()
	cal.Created = now
	cal.Modified = now
	s.calendars[key] = cal

	return nil
}

func (s *Store) UpdateCalendar(_ context.Context, cal *storage.Calendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(cal.UserID, cal.ID)
	if _, exists := s.calendars[key]; !exists {
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	cal.Modified = time.Now()
	s.calendars[key] = cal

	return nil
}

func (s *Store) DeleteCalendar(_ context.Context, userID, calendarID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(userID, calendarID)
	if _, exists := s.calendars[key]; !exists {
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	delete(s.calendars, key)

	// Delete all objects in this calendar
	for objKey, obj := range s.objects {
		if obj.CalendarID == calendarID && obj.UserID == userID {
			delete(s.objects, objKey)
		}
	}

	return nil
}

// Calendar object operations

func (s *Store) GetObject(_ context.Context, userID, objectID string) (*storage.CalendarObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[s.objectKey(userID, objectID)]
	if !ok {
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	return obj, nil
}

func (s *Store) ListObjects(_ context.Context, userID, calendarID string, opts *storage.ListOptions) ([]*storage.CalendarObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var objects []*storage.CalendarObject
	for _, obj := range s.objects {
		if obj.UserID == userID && obj.CalendarID == calendarID {
			// Apply filters if provided
			if opts != nil {
				// Filter by component type
				if len(opts.ComponentTypes) > 0 {
					found := false
					for _, t := range opts.ComponentTypes {
						if obj.ObjectType == t {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				// Filter by time range
				if opts.Start != nil || opts.End != nil {
					// Implementation note: For a real storage backend,
					// you would need to handle recurrence rules here
					start, err := obj.Event.DateTimeStart(nil)
					if err != nil {
						continue
					}
					end, err := obj.Event.DateTimeEnd(nil)
					if err != nil {
						continue
					}

					if opts.Start != nil && end.Before(*opts.Start) {
						continue
					}
					if opts.End != nil && start.After(*opts.End) {
						continue
					}
				}
			}

			objects = append(objects, obj)
		}
	}

	return objects, nil
}

func (s *Store) CreateObject(_ context.Context, obj *storage.CalendarObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(obj.UserID, obj.ID)
	if _, exists := s.objects[key]; exists {
		return &storage.Error{
			Type:    storage.ErrAlreadyExists,
			Message: "object already exists",
		}
	}

	// Verify calendar exists
	calKey := s.calendarKey(obj.UserID, obj.CalendarID)
	if _, exists := s.calendars[calKey]; !exists {
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	now := time.Now()
	obj.Created = now
	obj.Modified = now
	s.objects[key] = obj

	return nil
}

func (s *Store) UpdateObject(_ context.Context, obj *storage.CalendarObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(obj.UserID, obj.ID)
	if _, exists := s.objects[key]; !exists {
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	obj.Modified = time.Now()
	s.objects[key] = obj

	return nil
}

func (s *Store) DeleteObject(_ context.Context, userID, objectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(userID, objectID)
	if _, exists := s.objects[key]; !exists {
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	delete(s.objects, key)
	return nil
}
