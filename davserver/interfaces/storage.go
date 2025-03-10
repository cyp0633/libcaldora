package interfaces

import (
	"context"
	"time"

	"github.com/emersion/go-ical"
)

// ResourceType represents the type of a DAV resource
type ResourceType int

const (
	ResourceTypeCollection ResourceType = iota
	ResourceTypeCalendar
	ResourceTypeCalendarObject
)

// ResourceProperties represents WebDAV resource properties
type ResourceProperties struct {
	Path                string
	Type                ResourceType
	DisplayName         string
	Color               string
	SupportedComponents []string
	CurrentUserPrivSet  []string
	LastModified        time.Time
	ContentType         string
	ETag                string
	CTag                string
	PrincipalURL        string        `xml:"DAV: principal-URL,omitempty"`                              // URL for the authenticated user's principal
	CalendarHomeURL     string        `xml:"-"`                                                         // Internal field, not directly marshaled
	CalendarHomeSet     *CalendarHome `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set,omitempty"` // CalDAV calendar-home-set property
}

// CalendarHome represents a CalDAV calendar-home-set property
type CalendarHome struct {
	Href string `xml:"DAV: href"`
}

// Calendar represents a CalDAV calendar collection
type Calendar struct {
	Properties *ResourceProperties
	Data       *ical.Calendar
}

// CalendarObject wraps an iCalendar object with its metadata
type CalendarObject struct {
	Properties *ResourceProperties
	Data       *ical.Calendar
}

// QueryFilter represents a calendar query filter
type QueryFilter struct {
	CompFilter string // VEVENT, VTODO, etc.
	TimeRange  *TimeRange
	Status     []string
	Categories []string
	Limit      int
	Properties []string // Properties to return
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ListableProvider is an optional interface that providers can implement to support
// listing child resources in collections (for PROPFIND with Depth=1)
type ListableProvider interface {
	// ListResources returns a list of resources within the given path
	ListResources(ctx context.Context, path string) ([]*ResourceProperties, error)
}

// CalendarProvider defines the interface that storage providers must implement
type CalendarProvider interface {
	// GetResourceProperties returns properties for a resource at the given path
	GetResourceProperties(ctx context.Context, path string) (*ResourceProperties, error)

	// GetCurrentUserPrincipal returns the principal URL for the authenticated user
	GetCurrentUserPrincipal(ctx context.Context) (string, error)

	// GetCalendarHomeSet returns the calendar home collection URL for the authenticated user
	GetCalendarHomeSet(ctx context.Context, principalPath string) (string, error)

	// GetCalendar returns calendar information for a calendar collection
	GetCalendar(ctx context.Context, path string) (*Calendar, error)

	// GetCalendarObject returns a calendar object at the given path
	GetCalendarObject(ctx context.Context, path string) (*CalendarObject, error)

	// ListCalendarObjects returns calendar objects in a calendar collection
	ListCalendarObjects(ctx context.Context, path string) ([]CalendarObject, error)

	// PutCalendarObject creates or updates a calendar object
	PutCalendarObject(ctx context.Context, path string, object *CalendarObject) error

	// DeleteCalendarObject deletes a calendar object
	DeleteCalendarObject(ctx context.Context, path string) error

	// Optional interface methods that providers can implement for better performance
	// If not implemented, the server will use default implementations

	// Query returns calendar objects matching the given filter
	// Default implementation uses ListCalendarObjects and filters in memory
	Query(ctx context.Context, calendarPath string, filter *QueryFilter) ([]CalendarObject, error)

	// MultiGet returns calendar objects at the given paths
	// Default implementation calls GetCalendarObject for each path
	MultiGet(ctx context.Context, paths []string) ([]CalendarObject, error)
}
