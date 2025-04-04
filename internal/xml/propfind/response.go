package propfind

import (
	"errors"
	"strings"

	"github.com/beevik/etree"
	"github.com/samber/mo"
)

// Extend ParseRequest to handle different request types
func ParseRequest(xmlStr string) (ResponseMap, RequestType) {
	props := make(ResponseMap)
	requestType := RequestTypeProp // Default

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
		requestType = RequestTypeAllProp
		// For allprop, add all known properties
		for propName, structPtr := range propNameToStruct {
			props[propName] = mo.Ok(structPtr)
		}
		return props, requestType
	}

	if propname := propfindElem.FindElement("propname"); propname != nil {
		requestType = RequestTypePropName
		// For propname, add all known properties but mark them with ErrNotFound
		for propName := range propNameToStruct {
			props[propName] = mo.Err[PropertyEncoder](ErrNotFound)
		}
		return props, requestType
	}

	// Handle standard prop requests
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
			props[localName] = mo.Ok(structPtr)
		}
		// Skip unknown properties
	}

	return props, requestType
}

func EncodeResponse(props ResponseMap, href string) *etree.Document {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)

	// Create multistatus root element
	multistatus := doc.CreateElement("d:multistatus")

	// Add all required namespaces
	for prefix, uri := range namespaceMap {
		multistatus.CreateAttr("xmlns:"+prefix, uri)
	}

	// Create response element
	response := multistatus.CreateElement("d:response")

	hrefElem := response.CreateElement("d:href")
	hrefElem.SetText(href)

	// Maps to organize properties by their status code
	statusToPropstat := make(map[string]*etree.Element)
	statusToProp := make(map[string]*etree.Element)

	// Process each property
	for propName, propResult := range props {
		var statusCode string
		var propElem *etree.Element

		if propResult.IsOk() {
			// Property is available
			statusCode = "HTTP/1.1 200 OK"
			propEncoder := propResult.MustGet()
			propElem = propEncoder.Encode()
		} else {
			// Property has an error, determine the appropriate status code
			err := propResult.Error()
			switch err {
			case ErrNotFound:
				statusCode = "HTTP/1.1 404 Not Found"
			case ErrForbidden:
				statusCode = "HTTP/1.1 403 Forbidden"
			case ErrInternal:
				statusCode = "HTTP/1.1 500 Internal Server Error"
			case ErrBadRequest:
				statusCode = "HTTP/1.1 400 Bad Request"
			default:
				statusCode = "HTTP/1.1 500 Internal Server Error" // Default to 500
			}

			// Create an empty element for the property
			prefix, exists := propPrefixMap[propName]
			if !exists {
				prefix = "d" // Default to WebDAV namespace
			}

			propElem = etree.NewElement(propName)
			propElem.Space = prefix
		}

		// Create propstat for this status code if it doesn't exist yet
		if _, exists := statusToPropstat[statusCode]; !exists {
			propstat := response.CreateElement("d:propstat")
			prop := propstat.CreateElement("d:prop")
			status := propstat.CreateElement("d:status")
			status.SetText(statusCode)

			statusToPropstat[statusCode] = propstat
			statusToProp[statusCode] = prop
		}

		// Add property element to the appropriate prop element
		statusToProp[statusCode].AddChild(propElem)
	}

	return doc
}

// MergeResponses is used for merging responses for individual calendar resources into
// one response to a PROPFIND request (often with depth>0).
func MergeResponses(docs []*etree.Document) (*etree.Document, error) {
	if len(docs) == 0 {
		return nil, errors.New("no documents to merge")
	}

	// If only one document, return it directly
	if len(docs) == 1 {
		return docs[0], nil
	}

	// Create a new merged document
	mergedDoc := etree.NewDocument()
	mergedDoc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)

	// Create multistatus root element with namespaces
	mergedMultistatus := mergedDoc.CreateElement("d:multistatus")

	// Explicitly add required namespace declarations first
	mergedMultistatus.CreateAttr("xmlns:d", "DAV:")
	mergedMultistatus.CreateAttr("xmlns:cal", "urn:ietf:params:xml:ns:caldav")
	mergedMultistatus.CreateAttr("xmlns:cs", "http://calendarserver.org/ns/")

	// Copy namespaces from the first document's multistatus (for any additional namespaces)
	firstMultistatus := docs[0].FindElement("//d:multistatus")
	if firstMultistatus == nil {
		return nil, errors.New("first document missing multistatus element")
	}

	// Add any additional namespace declarations that aren't standard
	for _, attr := range firstMultistatus.Attr {
		if strings.HasPrefix(attr.Key, "xmlns:") &&
			attr.Key != "xmlns:d" &&
			attr.Key != "xmlns:cal" &&
			attr.Key != "xmlns:cs" {
			mergedMultistatus.CreateAttr(attr.Key, attr.Value)
		}
	}

	// For each document, find and copy all response elements to the merged document
	for _, doc := range docs {
		// Find all response elements
		responses := doc.FindElements("//d:multistatus/d:response")
		for _, resp := range responses {
			// Deep copy the response element
			respCopy := resp.Copy()
			mergedMultistatus.AddChild(respCopy)
		}
	}

	return mergedDoc, nil
}
