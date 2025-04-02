package propfind

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/samber/mo"
)

// Extend ParseRequest to handle different request types
func ParseRequest(xmlStr string) (map[string]mo.Option[PropertyEncoder], string) {
	props := make(map[string]mo.Option[PropertyEncoder])
	requestType := "prop" // Default

	// Parse XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return props, requestType
	}

	// Find all property elements under propfind/prop
	propfindElem := doc.FindElement("//propfind")
	if propfindElem == nil {
		return props, requestType
	}

	// Check for allprop or propname
	if allprop := propfindElem.FindElement("allprop"); allprop != nil {
		requestType = "allprop"
		// For allprop, add all known properties
		for propName, structPtr := range propNameToStruct {
			props[propName] = mo.Some(structPtr)
		}
		return props, requestType
	}

	if propname := propfindElem.FindElement("propname"); propname != nil {
		requestType = "propname"
		// For propname, add all known properties but mark them as None
		// (they'll be rendered as empty elements)
		for propName := range propNameToStruct {
			props[propName] = mo.None[PropertyEncoder]()
		}
		return props, requestType
	}

	// Handle standard prop requests (already implemented)
	propElem := propfindElem.FindElement("prop")
	if propElem == nil {
		return props, requestType
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
		// Skip unknown properties instead of adding them as None
	}

	return props, requestType
}

func EncodeResponse(props map[string]mo.Option[PropertyEncoder], href string) *etree.Document {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)

	// Create multistatus root element
	multistatus := doc.CreateElement("d:multistatus")

	// Add all required namespaces
	for prefix, uri := range namespaceMap {
		multistatus.CreateAttr("xmlns:"+prefix, uri)
	}

	// Create response element - assumes single resource for now
	response := multistatus.CreateElement("d:response")

	hrefElem := response.CreateElement("d:href")
	hrefElem.SetText(href)

	// Create propstat for 200 OK properties
	okPropstat := response.CreateElement("d:propstat")
	okProp := okPropstat.CreateElement("d:prop")
	okStatus := okPropstat.CreateElement("d:status")
	okStatus.SetText("HTTP/1.1 200 OK")

	// Create propstat for 404 Not Found properties
	notFoundPropstat := response.CreateElement("d:propstat")
	notFoundProp := notFoundPropstat.CreateElement("d:prop")
	notFoundStatus := notFoundPropstat.CreateElement("d:status")
	notFoundStatus.SetText("HTTP/1.1 404 Not Found")

	// Track if we have any properties in each category
	hasOkProps := false
	hasNotFoundProps := false

	// Process each property
	for propName, propOption := range props {
		if propOption.IsPresent() {
			// Property is available, get its element
			propEncoder := propOption.MustGet()
			propElem := propEncoder.Encode()
			okProp.AddChild(propElem)
			hasOkProps = true
		} else {
			// Property was requested but not available
			// Find the appropriate prefix for this property
			prefix, exists := propPrefixMap[propName]
			if !exists {
				prefix = "d" // Default to WebDAV namespace
			}

			// Create an empty element to indicate it wasn't found
			emptyElem := etree.NewElement(propName)
			emptyElem.Space = prefix
			notFoundProp.AddChild(emptyElem)
			hasNotFoundProps = true
		}
	}

	// Remove empty propstat sections if needed
	if !hasOkProps {
		response.RemoveChild(okPropstat)
	}

	if !hasNotFoundProps {
		response.RemoveChild(notFoundPropstat)
	}

	return doc
}
