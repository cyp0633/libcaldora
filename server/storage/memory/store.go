package memory

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
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
	logger    *slog.Logger
}

// New creates a new in-memory storage
func New(opts ...Option) *Store {
	s := &Store{
		users:     make(map[string]*storage.User),
		calendars: make(map[string]*storage.Calendar),
		objects:   make(map[string]*storage.CalendarObject),
		props:     make(map[string]storage.Properties),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Option represents a configuration option for the Store
type Option func(*Store)

// WithLogger sets the logger for the store
func WithLogger(logger *slog.Logger) Option {
	return func(s *Store) {
		if logger != nil {
			s.logger = logger
		}
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
		s.logger.Info("user not found", "user_id", userID)
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "user not found",
		}
	}

	s.logger.Debug("retrieved user", "user_id", userID)
	return user, nil
}

// Calendar operations

func (s *Store) GetCalendar(_ context.Context, userID, calendarID string) (*storage.Calendar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cal, ok := s.calendars[s.calendarKey(userID, calendarID)]
	if !ok {
		s.logger.Info("calendar not found",
			"user_id", userID,
			"calendar_id", calendarID)
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	s.logger.Debug("retrieved calendar",
		"user_id", userID,
		"calendar_id", calendarID)
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

	s.logger.Debug("listed calendars",
		"user_id", userID,
		"count", len(calendars))
	return calendars, nil
}

func (s *Store) CreateCalendar(_ context.Context, cal *storage.Calendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(cal.UserID, cal.ID)
	if _, exists := s.calendars[key]; exists {
		s.logger.Info("failed to create calendar: already exists",
			"user_id", cal.UserID,
			"calendar_id", cal.ID)
		return &storage.Error{
			Type:    storage.ErrAlreadyExists,
			Message: "calendar already exists",
		}
	}

	now := time.Now()
	cal.Created = now
	cal.Modified = now
	s.calendars[key] = cal

	s.logger.Info("created calendar",
		"user_id", cal.UserID,
		"calendar_id", cal.ID)
	return nil
}

func (s *Store) UpdateCalendar(_ context.Context, cal *storage.Calendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(cal.UserID, cal.ID)
	if _, exists := s.calendars[key]; !exists {
		s.logger.Info("failed to update calendar: not found",
			"user_id", cal.UserID,
			"calendar_id", cal.ID)
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	cal.Modified = time.Now()
	s.calendars[key] = cal

	s.logger.Info("updated calendar",
		"user_id", cal.UserID,
		"calendar_id", cal.ID)
	return nil
}

func (s *Store) DeleteCalendar(_ context.Context, userID, calendarID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.calendarKey(userID, calendarID)
	if _, exists := s.calendars[key]; !exists {
		s.logger.Info("failed to delete calendar: not found",
			"user_id", userID,
			"calendar_id", calendarID)
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	delete(s.calendars, key)

	// Delete all objects in this calendar
	var deletedCount int
	for objKey, obj := range s.objects {
		if obj.CalendarID == calendarID && obj.UserID == userID {
			delete(s.objects, objKey)
			deletedCount++
		}
	}

	s.logger.Info("deleted calendar and objects",
		"user_id", userID,
		"calendar_id", calendarID,
		"deleted_objects", deletedCount)
	return nil
}

// Calendar object operations

func (s *Store) GetObject(_ context.Context, userID, objectID string) (*storage.CalendarObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[s.objectKey(userID, objectID)]
	if !ok {
		s.logger.Info("object not found",
			"user_id", userID,
			"object_id", objectID)
		return nil, &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	s.logger.Debug("retrieved object",
		"user_id", userID,
		"object_id", objectID,
		"type", obj.ObjectType)
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
						s.logger.Warn("failed to get event start time",
							"user_id", userID,
							"object_id", obj.ID,
							"error", err)
						continue
					}
					end, err := obj.Event.DateTimeEnd(nil)
					if err != nil {
						s.logger.Warn("failed to get event end time",
							"user_id", userID,
							"object_id", obj.ID,
							"error", err)
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

	s.logger.Debug("listed objects",
		"user_id", userID,
		"calendar_id", calendarID,
		"count", len(objects))
	return objects, nil
}

func (s *Store) CreateObject(_ context.Context, obj *storage.CalendarObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(obj.UserID, obj.ID)
	if _, exists := s.objects[key]; exists {
		s.logger.Info("failed to create object: already exists",
			"user_id", obj.UserID,
			"object_id", obj.ID,
			"calendar_id", obj.CalendarID)
		return &storage.Error{
			Type:    storage.ErrAlreadyExists,
			Message: "object already exists",
		}
	}

	// Verify calendar exists
	calKey := s.calendarKey(obj.UserID, obj.CalendarID)
	if _, exists := s.calendars[calKey]; !exists {
		s.logger.Info("failed to create object: calendar not found",
			"user_id", obj.UserID,
			"calendar_id", obj.CalendarID)
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "calendar not found",
		}
	}

	now := time.Now()
	obj.Created = now
	obj.Modified = now
	s.objects[key] = obj

	s.logger.Info("created object",
		"user_id", obj.UserID,
		"object_id", obj.ID,
		"calendar_id", obj.CalendarID,
		"type", obj.ObjectType)
	return nil
}

func (s *Store) UpdateObject(_ context.Context, obj *storage.CalendarObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(obj.UserID, obj.ID)
	if _, exists := s.objects[key]; !exists {
		s.logger.Info("failed to update object: not found",
			"user_id", obj.UserID,
			"object_id", obj.ID)
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	obj.Modified = time.Now()
	s.objects[key] = obj

	s.logger.Info("updated object",
		"user_id", obj.UserID,
		"object_id", obj.ID,
		"calendar_id", obj.CalendarID,
		"type", obj.ObjectType)
	return nil
}

func (s *Store) DeleteObject(_ context.Context, userID, objectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.objectKey(userID, objectID)
	obj, exists := s.objects[key]
	if !exists {
		s.logger.Info("failed to delete object: not found",
			"user_id", userID,
			"object_id", objectID)
		return &storage.Error{
			Type:    storage.ErrNotFound,
			Message: "object not found",
		}
	}

	delete(s.objects, key)

	s.logger.Info("deleted object",
		"user_id", userID,
		"object_id", objectID,
		"calendar_id", obj.CalendarID,
		"type", obj.ObjectType)
	return nil
}
