package storage

import (
	"time"

	"github.com/emersion/go-ical"
)

type Storage interface {
	GetEventsInCollection(calendarID string) (CalendarObject, error)
	GetUserCalendars(userID string) (Calendar, error) // Returns a list of calendars for a user
}

// Calendar represents a CalDAV calendar collection.
// It holds metadata and the core iCalendar data.
type Calendar struct {
	// Path is the unique URI path for this calendar resource.
	// Example: "/calendars/users/alice/work/"
	Path string

	// CTag represents the calendar collection tag.
	// It changes when the content (objects) of the calendar changes.
	CTag string

	// ETag represents the entity tag of the calendar properties.
	// It changes when the calendar's own properties (like NAME, COLOR) change.
	ETag string

	// Component stores the underlying VCALENDAR data using go-ical.
	// This holds properties like NAME, DESCRIPTION, COLOR etc.
	Component *ical.Component

	// Add any other necessary metadata specific to your implementation,
	// e.g., OwnerPrincipalPath string
}

// CalendarObject represents an individual calendar resource like an event (VEVENT),
// task (VTODO), or journal entry (VJOURNAL) within a calendar collection.
type CalendarObject struct {
	// Path is the unique URI path for this calendar object resource.
	// Example: "/calendars/users/alice/work/event123.ics"
	Path string

	// ETag represents the entity tag of the calendar object.
	// It changes whenever the object's data changes.
	ETag string

	// LastModified timestamp can be useful for generating ETags and handling synchronization.
	LastModified time.Time

	// Component stores the underlying VEVENT, VTODO, etc. data using go-ical.
	Component *ical.Component

	// CalendarPath links back to the parent Calendar collection's Path.
	CalendarPath string
}
