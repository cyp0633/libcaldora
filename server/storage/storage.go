package storage

import (
	"errors"
	"time"

	"github.com/emersion/go-ical"
)

// Storage interface connects your backend storage (e.g. database) with this server. Please use the error types provided.
type Storage interface {
	// GetObjectsInCollection retrieves all calendar objects (like VEVENT, VTODO) in a given calendar collection.
	GetObjectsInCollection(calendarID string) ([]CalendarObject, error)
	// GetObjectPathsInCollection retrieves paths of all calendar objects in a given calendar collection.
	GetObjectPathsInCollection(calendarID string) ([]string, error)
	// GetUserCalendars retrieves all calendar collections for a user.
	GetUserCalendars(userID string) ([]Calendar, error)
	// GetUser gets user information.
	GetUser(userID string) (*User, error)
	// AuthUser authenticates a user with username and password, returns the user ID if successful.
	AuthUser(username, password string) (string, error)
	// GetCalendar retrieves a specific calendar by user id and calendar id.
	GetCalendar(userID, calendarID string) (*Calendar, error)
	// GetObject finds a calendar object (VEVENT, VTODO, VJOURNAL, etc) by user id, calendar id and object id
	GetObject(userID, calendarID, objectID string) (*CalendarObject, error)
	// GetObjectByFilter finds calendar objects by user id, calendar id and filter
	GetObjectByFilter(userID, calendarID string, filter *Filter) ([]CalendarObject, error)
	// UpdateObject updates a calendar object. If not existing, create one
	// Should return the new ETag
	UpdateObject(userID, calendarID string, object *CalendarObject) (etag string, err error)
	// DeleteObject removes a calendar object.
	DeleteObject(userID, calendarID, objectID string) error
	// CreateCalendar creates a new calendar collection.
	// Implementation should set the etag and path inside the Calendar struct.
	CreateCalendar(userID string, calendar *Calendar) error
}

// Calendar represents a CalDAV calendar collection.
// It holds metadata and the core iCalendar data.
type Calendar struct {
	// Path is the unique URI path for this calendar resource.
	// Example: "/alice/cal/work"
	Path string
	// CTag represents the calendar collection tag.
	// It changes when the content (objects) of the calendar changes.
	CTag string
	// ETag represents the entity tag of the calendar properties.
	// It changes when the calendar's own properties (like NAME, COLOR) change.
	ETag string
	// Component stores the underlying VCALENDAR data using go-ical.
	// This holds properties like NAME, DESCRIPTION, COLOR etc.
	CalendarData *ical.Calendar
	// SupportedComponents lists the types of components supported by this calendar.
	// e.g. "VEVENT", "VTODO", "VJOURNAL"
	SupportedComponents []string
}

// CalendarObject represents an individual calendar resource like an event (VEVENT),
// task (VTODO), or journal entry (VJOURNAL) within a calendar collection.
type CalendarObject struct {
	// Path is the unique URI path for this calendar object resource.
	//
	// NOTE: This has nothing to do with iCal UID.
	//
	// Example: "/alice/cal/work/event1.ics"
	Path string

	// ETag represents the entity tag of the calendar object.
	// It changes whenever the object's data changes.
	// Generating etag is user's responsibility; libcaldora just uses the provided value.
	ETag string

	// LastModified timestamp can be useful for generating ETags and handling synchronization.
	LastModified time.Time

	// Component stores the underlying VEVENT, VTODO, etc. data using go-ical.
	Component *ical.Component
}

type User struct {
	// Will be returned in displayname
	DisplayName string
	// used for calendar-user-address-set
	UserAddress string
	// 6-character HEX string with # prefix, used for cs:calendar-color and g:color
	PreferredColor string
	// ISO 8601 timezone, e.g. Asia/Shanghai, used for g:timezone
	PreferredTimezone string
	// The user's principal path
	Path string
}

var (
	// ErrNotFound is returned when a requested resource doesn't exist
	ErrNotFound = errors.New("resource not found")
	// ErrInvalidInput is returned when the input parameters are invalid
	ErrInvalidInput = errors.New("invalid input parameters")
	// ErrPermissionDenied is returned when the operation is not allowed
	ErrPermissionDenied = errors.New("permission denied")
	// ErrConflict is returned when there's a conflict with an existing resource
	ErrConflict = errors.New("resource conflict")
	// ErrStorageUnavailable is returned when the storage backend is unavailable
	ErrStorageUnavailable = errors.New("storage unavailable")
)

// ResourceType indicates the type of CalDAV resource identified by the URL path.
// This is distinct from CalDAV prop "resourcetype".
type ResourceType int

const (
	ResourceUnknown ResourceType = iota
	ResourcePrincipal
	ResourceHomeSet
	ResourceCollection
	ResourceObject
	ResourceServiceRoot // Not really a resource, treat as unknown if not specified
)

// String provides a human-readable representation of the ResourceType.
func (rt ResourceType) String() string {
	switch rt {
	case ResourcePrincipal:
		return "Principal"
	case ResourceHomeSet:
		return "HomeSet"
	case ResourceCollection:
		return "Collection"
	case ResourceObject:
		return "Object"
	default:
		return "Unknown"
	}
}
