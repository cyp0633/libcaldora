package caldav

import (
	"fmt"
	"log"
	"strings"
)

// ResourceType indicates the type of CalDAV resource identified by the URL path.
type ResourceType int

// URLConverter helps you define URL path convention. Leave this blank when creating handler defaults to defaultURLConverter.
//
// However, there are some basic assumptions you should respect:
//
// A resource should be able to find its parent from its path. For example, /<userid>/cal/<calendarid>/<objectid> belongs to
// user <userid> and calendar <calendarid>. Please consider including all those information in your URI, or you might
// encounter excessive overhead looking for parent resources.
type URLConverter interface {
	// ParsePath parses a given path and returns the corresponding Resource.
	ParsePath(path string) (Resource, error)
	// EncodePath encodes a Resource back to its URL path representation.
	EncodePath(resource Resource) (string, error)
}

const (
	ResourceUnknown ResourceType = iota
	ResourcePrincipal
	ResourceHomeSet
	ResourceCollection
	ResourceObject
)

type Resource struct {
	UserID       string
	CalendarID   string
	ObjectID     string
	ResourceType ResourceType
}

type defaultURLConverter struct {
}

func (c defaultURLConverter) ParsePath(path string) (Resource, error) {
	resource := Resource{ResourceType: ResourceUnknown}
	parts := strings.Split(path, "/")

	// Filter out empty segments caused by leading/trailing/double slashes
	var segments []string
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}

	numSegments := len(segments)

	switch numSegments {
	case 0: // Path was just the prefix itself (e.g., /caldav/) - Usually not a valid CalDAV resource
		return resource, fmt.Errorf("invalid path: root path not directly addressable")

	case 1: // /<userid>
		resource.UserID = segments[0]
		resource.ResourceType = ResourcePrincipal
		// TODO: Check if resource.UserID is a valid user identifier format/exists
		log.Printf("TODO: Validate principal UserID: %s", resource.UserID)

	case 2: // /<userid>/cal
		if segments[1] == "cal" {
			resource.UserID = segments[0]
			resource.ResourceType = ResourceHomeSet
			// TODO: Check if resource.UserID is valid and has a calendar home set
			log.Printf("TODO: Validate homeset UserID: %s", resource.UserID)
		} else {
			return resource, fmt.Errorf("invalid path: expected '/<userid>/cal', got '/%s/%s'", segments[0], segments[1])
		}

	case 3: // /<userid>/cal/<calendarid>
		if segments[1] == "cal" {
			resource.UserID = segments[0]
			resource.CalendarID = segments[2]
			resource.ResourceType = ResourceCollection
			// TODO: Check if UserID and CalendarID are valid/exist
			log.Printf("TODO: Validate collection UserID: %s, CalendarID: %s", resource.UserID, resource.CalendarID)
		} else {
			return resource, fmt.Errorf("invalid path: expected '/<userid>/cal/<calendarid>', got '/%s/%s/%s'", segments[0], segments[1], segments[2])
		}

	case 4: // /<userid>/cal/<calendarid>/<objectid>
		if segments[1] == "cal" {
			resource.UserID = segments[0]
			resource.CalendarID = segments[2]
			resource.ObjectID = segments[3] // Object ID might contain ".ics"
			resource.ResourceType = ResourceObject
			// TODO: Check if UserID, CalendarID, and ObjectID are valid/exist
			log.Printf("TODO: Validate object UserID: %s, CalendarID: %s, ObjectID: %s", resource.UserID, resource.CalendarID, resource.ObjectID)
		} else {
			return resource, fmt.Errorf("invalid path: expected '/<userid>/cal/<calendarid>/<objectid>', got '/%s/%s/%s/%s'", segments[0], segments[1], segments[2], segments[3])
		}

	default: // More than 4 segments - not defined by our convention
		return resource, fmt.Errorf("invalid path: too many segments (%d)", numSegments)
	}

	return resource, nil
}

func (c defaultURLConverter) EncodePath(resource Resource) (string, error) {
	switch resource.ResourceType {
	case ResourcePrincipal:
		if resource.UserID == "" {
			return "", fmt.Errorf("invalid resource: principal must have a UserID")
		}
		return "/" + resource.UserID, nil

	case ResourceHomeSet:
		if resource.UserID == "" {
			return "", fmt.Errorf("invalid resource: home set must have a UserID")
		}
		return "/" + resource.UserID + "/cal", nil

	case ResourceCollection:
		if resource.UserID == "" || resource.CalendarID == "" {
			return "", fmt.Errorf("invalid resource: collection must have both UserID and CalendarID")
		}
		return "/" + resource.UserID + "/cal/" + resource.CalendarID, nil

	case ResourceObject:
		if resource.UserID == "" || resource.CalendarID == "" || resource.ObjectID == "" {
			return "", fmt.Errorf("invalid resource: object must have UserID, CalendarID, and ObjectID")
		}
		return "/" + resource.UserID + "/cal/" + resource.CalendarID + "/" + resource.ObjectID, nil

	default:
		return "", fmt.Errorf("invalid resource type: %s", resource.ResourceType.String())
	}
}

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
