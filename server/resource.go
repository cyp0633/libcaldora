package server

import (
	"fmt"
	"log"
	"strings"

	"github.com/cyp0633/libcaldora/server/storage"
)

// URLConverter helps you define URL path convention. Leave this blank when creating handler defaults to DefaultURLConverter.
//
// However, there are some basic assumptions you should respect:
//
// A resource should be able to find its parent from its path. For example, /<userid>/cal/<calendarid>/<objectid> belongs to
// user <userid> and calendar <calendarid>. Please consider including all those information in your URI, or you might
// encounter excessive overhead looking for parent resources.
//
// If you set prefix in the handler, you should consider initializing your URLConverter with the same prefix, like DefaultURLConverter does.
type URLConverter interface {
	// ParsePath parses a given path and returns the corresponding Resource.
	ParsePath(path string) (Resource, error)
	// EncodePath encodes a Resource back to its URL path representation.
	EncodePath(resource Resource) (string, error)
}

type Resource struct {
	UserID       string
	CalendarID   string
	ObjectID     string
	URI          string // may save encode/parsing overhead
	ResourceType storage.ResourceType
}

// DefaultURLConverter implements the URLConverter interface with a standard CalDAV URL structure:
// /<userid>/cal/<calendarid>/<objectid>
//
// The URL structure follows these rules:
// - Service Root: /
// - Principal: /<userid>
// - Home Set: /<userid>/cal
// - Collection: /<userid>/cal/<calendarid>
// - Object: /<userid>/cal/<calendarid>/<objectid>
//
// The Prefix field can be used to add a common prefix to all paths (e.g., "/caldav/")
type DefaultURLConverter struct {
	Prefix string
}

// ParsePath parses a CalDAV path into its components.
// It handles paths with or without the configured prefix.
func (c *DefaultURLConverter) ParsePath(path string) (Resource, error) {
	resource := Resource{ResourceType: storage.ResourceUnknown, URI: path}

	// Strip the prefix if present
	path = strings.TrimPrefix(path, c.Prefix)
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
	case 0: // service root
		resource.ResourceType = storage.ResourceServiceRoot

	case 1: // /<userid>
		resource.UserID = segments[0]
		resource.ResourceType = storage.ResourcePrincipal
		// TODO: Check if resource.UserID is a valid user identifier format/exists
		log.Printf("TODO: Validate principal UserID: %s", resource.UserID)

	case 2: // /<userid>/cal
		if segments[1] == "cal" {
			resource.UserID = segments[0]
			resource.ResourceType = storage.ResourceHomeSet
			// TODO: Check if resource.UserID is valid and has a calendar home set
			log.Printf("TODO: Validate homeset UserID: %s", resource.UserID)
		} else {
			return resource, fmt.Errorf("invalid path: expected '/<userid>/cal', got '/%s/%s'", segments[0], segments[1])
		}

	case 3: // /<userid>/cal/<calendarid>
		if segments[1] == "cal" {
			resource.UserID = segments[0]
			resource.CalendarID = segments[2]
			resource.ResourceType = storage.ResourceCollection
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
			resource.ResourceType = storage.ResourceObject
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

// EncodePath encodes a Resource into a CalDAV path.
// It validates that the resource has all required fields for its type
// and adds the configured prefix to the path.
func (c *DefaultURLConverter) EncodePath(resource Resource) (string, error) {
	var path string

	switch resource.ResourceType {
	case storage.ResourcePrincipal:
		if resource.UserID == "" {
			return "", fmt.Errorf("invalid resource: principal must have a UserID")
		}
		path = "/" + resource.UserID

	case storage.ResourceHomeSet:
		if resource.UserID == "" {
			return "", fmt.Errorf("invalid resource: home set must have a UserID")
		}
		path = "/" + resource.UserID + "/cal"

	case storage.ResourceCollection:
		if resource.UserID == "" || resource.CalendarID == "" {
			return "", fmt.Errorf("invalid resource: collection must have both UserID and CalendarID")
		}
		path = "/" + resource.UserID + "/cal/" + resource.CalendarID

	case storage.ResourceObject:
		if resource.UserID == "" || resource.CalendarID == "" || resource.ObjectID == "" {
			return "", fmt.Errorf("invalid resource: object must have UserID, CalendarID, and ObjectID")
		}
		path = "/" + resource.UserID + "/cal/" + resource.CalendarID + "/" + resource.ObjectID

	case storage.ResourceServiceRoot:
		path = "/"

	default:
		return "", fmt.Errorf("invalid resource type: %s", resource.ResourceType.String())
	}

	// Add the prefix to the path
	return c.Prefix + strings.TrimPrefix(path, "/"), nil
}
