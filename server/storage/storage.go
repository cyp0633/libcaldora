package storage

import (
	"time"

	"github.com/emersion/go-ical"
)

type Storage interface {
	// GetObjectsInCollection retrieves all calendar objects (like VEVENT, VTODO) in a given calendar collection.
	GetObjectsInCollection(calendarID string) ([]CalendarObject, error)
	// GetObjectPathsInCollection retrieves paths of all calendar objects in a given calendar collection.
	GetObjectPathsInCollection(calendarID string) ([]string, error)
	// GetUserCalendars retrieves all calendar collections for a user.
	GetUserCalendars(userID string) ([]Calendar, error)
	// GetUser gets user information.
	GetUser(userID string) (*User, error)
	// GetCalendar retrieves a specific calendar by user id and calendar id.
	GetCalendar(userID, calendarID string) (*Calendar, error)
	// GetObject finds a calendar object (VEVENT, VTODO, VJOURNAL, etc) by user id, calendar id and object id
	GetObject(userID, calendarID, objectID string) (*CalendarObject, error)
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

	// Add any other necessary metadata specific to your implementation,
	// e.g., OwnerPrincipalPath string
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
}

// ResourceType indicates the type of CalDAV resource identified by the URL path.
// This is distinct from CalDAV prop "resourcetype".
type ResourceType int

const (
	ResourceUnknown ResourceType = iota
	ResourcePrincipal
	ResourceHomeSet
	ResourceCollection
	ResourceObject
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
