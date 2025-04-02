package propfind

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/samber/mo"
)

func ParseRequest(xmlStr string) map[string]mo.Option[any] {
	props := make(map[string]mo.Option[any])

	// Parse XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return props
	}

	// Find all property elements under propfind/prop
	propfindElem := doc.FindElement("//propfind")
	if propfindElem == nil {
		return props
	}

	propElem := propfindElem.FindElement("prop")
	if propElem == nil {
		return props
	}

	// Iterate through all child elements of prop
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
		if structPtr, exists := propNameToStruct[localName]; exists {
			// Add the property to the response map
			props[localName] = mo.Some(structPtr)
		}
	}

	return props
}
