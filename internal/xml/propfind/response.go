package propfind

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/samber/mo"
)

// Extend ParseRequest to handle different request types
func ParseRequest(xmlStr string) (ResponseMap, RequestType) {
	propsMap := make(ResponseMap)
	requestType := RequestTypeProp // Default

	// Parse XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		return propsMap, requestType
	}

	// Find all property elements under propfind/prop
	propfindElem := doc.FindElement("//propfind")
	if propfindElem == nil {
		return propsMap, requestType
	}

	// Check for allprop or propname
	if allprop := propfindElem.FindElement("allprop"); allprop != nil {
		requestType = RequestTypeAllProp
		// For allprop, add all known properties
		for propName, structPtr := range props.PropNameToStruct {
			propsMap[propName] = mo.Ok[props.Property](structPtr)
		}
		return propsMap, requestType
	}

	if propname := propfindElem.FindElement("propname"); propname != nil {
		requestType = RequestTypePropName
		// For propname, add all known properties but mark them with ErrNotFound
		for propName := range props.PropNameToStruct {
			propsMap[propName] = mo.Err[props.Property](ErrNotFound)
		}
		return propsMap, requestType
	}

	// Handle standard prop requests
	propElem := propfindElem.FindElement("prop")
	if propElem == nil {
		return propsMap, requestType
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
			propsMap[localName] = mo.Ok(structPtr)
		}
		// Skip unknown properties
	}

	return propsMap, requestType
}

func EncodeResponse(propsMap ResponseMap, href string) *etree.Document {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)

	// Create multistatus root element
	multistatus := doc.CreateElement("d:multistatus")

	// Add all required namespaces
	for prefix, uri := range props.NamespaceMap {
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
	for propName, propResult := range propsMap {
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
			// Use PropPrefixMap to determine the correct namespace prefix
			prefix, exists := props.PropPrefixMap[propName]
			if !exists {
				prefix = "d" // Default to WebDAV namespace if not found
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
	// 1. Create the final merged document structure
	mergedDoc := etree.NewDocument()
	mergedDoc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)

	// Create the root <d:multistatus> element
	mergedMultistatus := mergedDoc.CreateElement("d:multistatus")
	// Setting Space is important for etree to know the prefix 'd' belongs to the default NS or a defined one.
	// We will define 'd' explicitly below using xmlns:d.
	mergedMultistatus.Space = "d"

	// 2. Add necessary namespace declarations (xmlns attributes) to the root element.
	// Using the same namespaceMap as EncodeResponse ensures consistency.
	for prefix, uri := range props.NamespaceMap {
		mergedMultistatus.CreateAttr("xmlns:"+prefix, uri)
	}

	// 3. Iterate through each input document (sub-response)
	for _, doc := range docs {
		if doc == nil {
			continue // Skip nil documents
		}

		// Find the root <d:multistatus> element in the sub-response document.
		// Using doc.Root() assumes the structure generated by EncodeResponse is correct.
		subMultistatus := doc.Root()
		if subMultistatus == nil || subMultistatus.Tag != "multistatus" || subMultistatus.Space != "d" {
			// Log or handle error: fmt.Errorf("invalid sub-response structure: expected d:multistatus root in doc %p", doc)
			continue // Skip documents with unexpected root elements
		}

		// Find all direct child <d:response> elements within the sub-response's <d:multistatus>.
		// Using FindElements("./d:response") ensures we only get direct children with the correct tag and namespace prefix 'd'.
		subResponses := subMultistatus.FindElements("./d:response")

		// 4. Add each found <d:response> element to the merged <d:multistatus> element.
		// AddChild effectively moves the element from the source doc to the target doc.
		// If the original sub-response docs needed to be preserved elsewhere, use subResponse.Copy()
		for _, subResponse := range subResponses {
			mergedMultistatus.AddChild(subResponse)
		}
	}

	//  5. Return the completed merged document. No errors are expected in this aggregation logic
	//     assuming input docs are valid, so return nil error.
	return mergedDoc, nil
}
