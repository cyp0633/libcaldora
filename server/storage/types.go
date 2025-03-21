package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
)

// Error types
type ErrorType string

const (
	ErrNotFound      ErrorType = "not_found"
	ErrAlreadyExists ErrorType = "already_exists"
	ErrInvalidInput  ErrorType = "invalid_input"
)

// Error represents a storage-related error
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// User represents a CalDAV user
type User struct {
	ID string
	// Additional user properties if needed
}

// Calendar represents a calendar collection
type Calendar struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Color       string
	TimeZone    string
	Components  []string // Supported component types (VEVENT, VTODO, etc.)
	Created     time.Time
	Modified    time.Time
	SyncToken   string
	// Embedded go-ical calendar for iCalendar properties
	*ical.Calendar
}

// CalendarObject represents a calendar object (event, todo, etc.)
type CalendarObject struct {
	ID         string
	CalendarID string
	UserID     string
	ETag       string
	ObjectType string // VEVENT, VTODO, etc.
	Created    time.Time
	Modified   time.Time
	// Use go-ical Event type for calendar object properties
	*ical.Event
}

// ListOptions provides options for listing calendar objects
type ListOptions struct {
	// Time range filter
	Start *time.Time
	End   *time.Time

	// Component filter
	ComponentTypes []string // VEVENT, VTODO, etc.

	// Properties to return
	Properties []string

	// Recurrence expansion
	ExpandRecurrences bool
}

// Storage is the interface that must be implemented by storage backends
type Storage interface {
	// User operations
	GetUser(ctx context.Context, userID string) (*User, error)

	// Calendar operations
	GetCalendar(ctx context.Context, userID, calendarID string) (*Calendar, error)
	ListCalendars(ctx context.Context, userID string) ([]*Calendar, error)
	CreateCalendar(ctx context.Context, cal *Calendar) error
	UpdateCalendar(ctx context.Context, cal *Calendar) error
	DeleteCalendar(ctx context.Context, userID, calendarID string) error

	// Calendar object operations
	GetObject(ctx context.Context, userID, objectID string) (*CalendarObject, error)
	ListObjects(ctx context.Context, userID, calendarID string, opts *ListOptions) ([]*CalendarObject, error)
	CreateObject(ctx context.Context, obj *CalendarObject) error
	UpdateObject(ctx context.Context, obj *CalendarObject) error
	DeleteObject(ctx context.Context, userID, objectID string) error
}

// Property represents a WebDAV property
type Property struct {
	Name      string
	Value     string
	Protected bool
}

// Properties is a map of property names to their values
type Properties map[string]*Property

// ResourceInfo contains information about a CalDAV resource
type ResourceInfo struct {
	Path       *ResourcePath
	ETag       string
	Type       string // calendar, calendar-object
	Properties Properties
	Created    time.Time
	Modified   time.Time
}
