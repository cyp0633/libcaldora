package calendarmultiget

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/samber/mo"
)

// ParseRequest parses a calendar-multiget REPORT request XML.
// It returns a ResponseMap containing the requested properties and a slice of requested URIs.
func ParseRequest(xmlStr string) (propfind.ResponseMap, []string) {
	propsMap := make(propfind.ResponseMap)
	// Initialize with empty slice instead of nil
	hrefs := []string{}

	// Parse XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return propsMap, hrefs
	}

	// Find calendar-multiget element
	multigetElem := doc.FindElement("//calendar-multiget")
	if multigetElem == nil {
		return propsMap, hrefs
	}

	// Handle prop element
	propElem := multigetElem.FindElement("prop")
	if propElem == nil {
		return propsMap, hrefs
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
		if structPtr, exists := props.PropNameToStruct[localName]; exists {
			// Add the property to the response map
			propsMap[localName] = mo.Ok[props.PropertyEncoder](structPtr)
		}
		// Skip unknown properties
	}

	// Collect href elements directly under <calendar-multiget>
	for _, elem := range multigetElem.ChildElements() {
		tag := elem.Tag
		if idx := strings.Index(tag, ":"); idx != -1 {
			tag = tag[idx+1:]
		}
		if strings.ToLower(tag) == "href" {
			hrefs = append(hrefs, elem.Text())
		}
	}

	return propsMap, hrefs
}
