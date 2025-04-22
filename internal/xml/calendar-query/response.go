package calendarquery

import (
	"errors"
	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/samber/mo"
)

// ParseRequest parses a calendar-query REPORT request XML into a property map and filter
func ParseRequest(xmlStr string) (propfind.ResponseMap, *storage.Filter, error) {
	propsMap := make(propfind.ResponseMap)
	var filter *storage.Filter

	// Check for empty XML
	if xmlStr == "" {
		return propsMap, nil, errors.New("empty XML document")
	}

	// Parse XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return propsMap, nil, err
	}

	// Find the calendar-query element (root element)
	calendarQueryElem := doc.Root()
	if calendarQueryElem == nil {
		return propsMap, nil, errors.New("missing calendar-query root element")
	}

	// Find the prop element inside calendar-query
	propElem := findElementIgnoreNS(calendarQueryElem, "prop")
	if propElem != nil {
		// Parse requested properties, similar to propfind.ParseRequest
		for _, elem := range propElem.ChildElements() {
			// Get local name of the property (without namespace)
			localName := elem.Tag

			// If there's a namespace prefix, remove it
			if strings.Contains(localName, ":") {
				localName = strings.Split(localName, ":")[1]
			}

			// Convert to lowercase for case-insensitive matching
			localName = strings.ToLower(localName)

			// Check if we have a struct for this property
			if structPtr, exists := props.PropNameToStruct[localName]; exists {
				// Add the property to the response map
				propsMap[localName] = mo.Ok(structPtr)
			}
			// Skip unknown properties
		}
	}

	// Find the filter element inside calendar-query
	filterElem := findElementIgnoreNS(calendarQueryElem, "filter")
	if filterElem != nil {
		// Parse the filter using the existing ParseFilterElement function
		parsedFilter, err := ParseFilterElement(filterElem)
		if err != nil {
			return propsMap, nil, err
		}
		filter = parsedFilter
	}

	return propsMap, filter, nil
}
