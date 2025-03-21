package storage

import (
	"fmt"
	"strings"
)

// ResourceType represents the type of a CalDAV resource
type ResourceType int

const (
	ResourceTypePrincipal ResourceType = iota
	ResourceTypeCalendarHome
	ResourceTypeCalendar
	ResourceTypeObject
)

// String returns the string representation of the ResourceType
func (rt ResourceType) String() string {
	switch rt {
	case ResourceTypePrincipal:
		return "principal"
	case ResourceTypeCalendarHome:
		return "calendar-home"
	case ResourceTypeCalendar:
		return "calendar"
	case ResourceTypeObject:
		return "object"
	default:
		return "unknown"
	}
}

// ResourcePath represents a parsed CalDAV resource path
type ResourcePath struct {
	Type       ResourceType
	UserID     string
	CalendarID string
	ObjectID   string
}

// String returns the string representation of the ResourcePath
func (rp *ResourcePath) String() string {
	switch rp.Type {
	case ResourceTypePrincipal:
		return fmt.Sprintf("/u/%s", rp.UserID)
	case ResourceTypeCalendarHome:
		return fmt.Sprintf("/u/%s/cal", rp.UserID)
	case ResourceTypeCalendar:
		return fmt.Sprintf("/u/%s/cal/%s", rp.UserID, rp.CalendarID)
	case ResourceTypeObject:
		return fmt.Sprintf("/u/%s/evt/%s", rp.UserID, rp.ObjectID)
	default:
		return ""
	}
}

// ParseResourcePath parses a CalDAV resource path into its components
func ParseResourcePath(path string) (*ResourcePath, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Split path into components
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "u" {
		return nil, fmt.Errorf("invalid path format")
	}

	// Get user ID
	userID := parts[1]
	if userID == "" {
		return nil, fmt.Errorf("invalid user ID")
	}

	// User principal path: /u/<userid>
	if len(parts) == 2 {
		return &ResourcePath{
			Type:   ResourceTypePrincipal,
			UserID: userID,
		}, nil
	}

	// Calendar home or calendar object paths need at least 3 parts
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid path format")
	}

	switch parts[2] {
	case "cal":
		// Calendar home path: /u/<userid>/cal
		if len(parts) == 3 {
			return &ResourcePath{
				Type:   ResourceTypeCalendarHome,
				UserID: userID,
			}, nil
		}
		// Calendar path: /u/<userid>/cal/<calendarid>
		if len(parts) == 4 {
			return &ResourcePath{
				Type:       ResourceTypeCalendar,
				UserID:     userID,
				CalendarID: parts[3],
			}, nil
		}
	case "evt":
		// Calendar object path: /u/<userid>/evt/<objectid>
		if len(parts) == 4 {
			return &ResourcePath{
				Type:     ResourceTypeObject,
				UserID:   userID,
				ObjectID: parts[3],
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid path format")
}
