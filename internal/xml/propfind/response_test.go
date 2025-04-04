package propfind

import (
	"reflect"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name     string
		xmlInput string
		want     map[string]reflect.Type // Expected property names and their types
	}{
		{
			name: "Basic WebDAV properties",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:displayname/>
    <d:resourcetype/>
    <d:getetag/>
    <d:getlastmodified/>
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{
				"displayname":     reflect.TypeOf(new(displayName)),
				"resourcetype":    reflect.TypeOf(new(resourcetype)),
				"getetag":         reflect.TypeOf(new(getEtag)),
				"getlastmodified": reflect.TypeOf(new(getLastModified)),
			},
		},
		{
			name: "Mixed WebDAV and CalDAV properties",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:displayname/>
    <c:calendar-description/>
    <d:resourcetype/>
    <c:supported-calendar-component-set/>
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{
				"displayname":                      reflect.TypeOf(new(displayName)),
				"calendar-description":             reflect.TypeOf(new(calendarDescription)),
				"resourcetype":                     reflect.TypeOf(new(resourcetype)),
				"supported-calendar-component-set": reflect.TypeOf(new(supportedCalendarComponentSet)),
			},
		},
		{
			name: "Apple and Google extensions",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:g="http://schemas.google.com/gCal/2005">
  <d:prop>
    <cs:getctag/>
    <g:color/>
    <cs:invite/>
    <g:hidden/>
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{
				"getctag": reflect.TypeOf(new(getCTag)),
				"color":   reflect.TypeOf(new(color)),
				"invite":  reflect.TypeOf(new(invite)),
				"hidden":  reflect.TypeOf(new(hidden)),
			},
		},
		{
			name: "Empty propfind",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Invalid XML",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:displayname/
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "No prop element",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
</d:propfind>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Property not in mapping",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:displayname/>
    <d:nonexistent-property/>
    <d:getetag/>
  </d:prop>
</d:propfind>`,
			want: map[string]reflect.Type{
				"displayname": reflect.TypeOf(new(displayName)),
				"getetag":     reflect.TypeOf(new(getEtag)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, typ := ParseRequest(tt.xmlInput)

			// Check if the number of properties matches
			assert.Equal(t, len(tt.want), len(got),
				"Result should have %d properties, got %d", len(tt.want), len(got))
			assert.Equal(t, RequestTypeProp, typ, "Request type should be RequestTypeProp")

			// Check each expected property type
			for propName, expectedType := range tt.want {
				result, exists := got[propName]
				assert.True(t, exists, "Property %s should exist in result", propName)

				if exists {
					// Check if the result has a value
					assert.True(t, result.IsOk(), "Property %s should have a value", propName)

					// Check if the value is of the expected type
					value := result.MustGet()
					actualType := reflect.TypeOf(value)
					assert.Equal(t, expectedType, actualType,
						"Property %s should have type %v, got %v", propName, expectedType, actualType)
				}
			}

			// Check if there are no unexpected properties
			for propName := range got {
				_, exists := tt.want[propName]
				assert.True(t, exists, "Unexpected property in result: %s", propName)
			}
		})
	}
}

func TestParseRequest_AllProperties(t *testing.T) {
	// Create an XML request with all known properties
	xmlStart := `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" 
            xmlns:cs="http://calendarserver.org/ns/" xmlns:g="http://schemas.google.com/gCal/2005">
  <d:prop>
`
	xmlEnd := `  </d:prop>
</d:propfind>`

	// Add all properties from our mapping
	xmlMiddle := ""
	expectedProps := make(map[string]reflect.Type)

	for propName, structPtr := range propNameToStruct {
		// Determine the correct namespace prefix based on the property
		prefix := "d"
		if strings.HasPrefix(propName, "calendar-") ||
			strings.HasPrefix(propName, "supported-calendar") ||
			strings.HasPrefix(propName, "schedule-") ||
			strings.HasPrefix(propName, "max-") ||
			strings.HasPrefix(propName, "min-") {
			prefix = "c"
		} else if propName == "getctag" ||
			strings.HasPrefix(propName, "calendar-proxy") ||
			propName == "auto-schedule" ||
			propName == "invite" ||
			propName == "shared-url" ||
			propName == "notification-url" ||
			propName == "calendar-changes" {
			prefix = "cs"
		} else if propName == "color" || propName == "timezone" ||
			propName == "hidden" || propName == "selected" {
			prefix = "g"
		}

		xmlMiddle += "    <" + prefix + ":" + propName + "/>\n"
		expectedProps[propName] = reflect.TypeOf(structPtr)
	}

	xmlInput := xmlStart + xmlMiddle + xmlEnd

	// Parse the request
	got, typ := ParseRequest(xmlInput)

	// Check if all properties are correctly parsed
	assert.Equal(t, len(expectedProps), len(got),
		"Should have parsed all %d properties", len(expectedProps))
	assert.Equal(t, RequestTypeProp, typ, "Request type should be RequestTypeProp")

	// Check each property type
	for propName, expectedType := range expectedProps {
		result, exists := got[propName]
		assert.True(t, exists, "Property %s should exist in result", propName)

		if exists {
			// Check if the result has a value
			assert.True(t, result.IsOk(), "Property %s should have a value", propName)

			// Check if the value is of the expected type
			value := result.MustGet()
			actualType := reflect.TypeOf(value)
			assert.Equal(t, expectedType, actualType,
				"Property %s should have type %v, got %v", propName, expectedType, actualType)
		}
	}

	// Check if there are no unexpected properties
	for propName := range got {
		_, exists := expectedProps[propName]
		assert.True(t, exists, "Unexpected property in result: %s", propName)
	}
}

func TestEncodeResponse(t *testing.T) {
	tests := []struct {
		name     string
		props    map[string]mo.Result[PropertyEncoder]
		href     string
		expected func(t *testing.T, doc *etree.Document)
	}{
		{
			name: "Mix of found and not found properties",
			props: map[string]mo.Result[PropertyEncoder]{
				"displayname": mo.Ok[PropertyEncoder](&displayName{Value: "Test Calendar"}),
				"resourcetype": mo.Ok[PropertyEncoder](&resourcetype{
					Types: []string{"collection", "calendar"},
				}),
				"getetag":                mo.Err[PropertyEncoder](ErrNotFound),
				"calendar-color":         mo.Err[PropertyEncoder](ErrNotFound),
				"getcontenttype":         mo.Ok[PropertyEncoder](&getContentType{Value: "text/calendar"}),
				"current-user-principal": mo.Err[PropertyEncoder](ErrNotFound),
			},
			href: "/calendars/user1/calendar1/",
			expected: func(t *testing.T, doc *etree.Document) {
				// Verify basic structure
				multistatus := doc.FindElement("//d:multistatus")
				assert.NotNil(t, multistatus, "Should have multistatus element")

				// Check namespaces
				for prefix, uri := range namespaceMap {
					nsAttr := multistatus.SelectAttr("xmlns:" + prefix)
					assert.NotNil(t, nsAttr, "Should declare namespace %s", prefix)
					assert.Equal(t, uri, nsAttr.Value, "Namespace URI should match")
				}

				// Verify href value
				href := doc.FindElement("//d:response/d:href")
				assert.NotNil(t, href, "Should have href element")
				assert.Equal(t, "/calendars/user1/calendar1/", href.Text(), "Href should match input")

				// Check 200 propstat section
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.NotNil(t, okPropstat, "Should have 200 OK propstat")

				// Check 404 propstat section
				notFoundPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 404 Not Found']")
				assert.NotNil(t, notFoundPropstat, "Should have 404 Not Found propstat")

				// Verify found properties
				okProp := okPropstat.FindElement("d:prop")
				assert.NotNil(t, okProp, "Should have prop element in 200 section")

				displayname := okProp.FindElement("d:displayname")
				assert.NotNil(t, displayname, "Should have displayname property")
				assert.Equal(t, "Test Calendar", displayname.Text(), "Displayname should match")

				resourcetype := okProp.FindElement("d:resourcetype")
				assert.NotNil(t, resourcetype, "Should have resourcetype property")
				assert.NotNil(t, resourcetype.FindElement("d:collection"), "Resourcetype should have collection")
				assert.NotNil(t, resourcetype.FindElement("cal:calendar"), "Resourcetype should have calendar")

				contenttype := okProp.FindElement("d:getcontenttype")
				assert.NotNil(t, contenttype, "Should have getcontenttype property")
				assert.Equal(t, "text/calendar", contenttype.Text(), "Content type should match")

				// Verify not found properties
				notFoundProp := notFoundPropstat.FindElement("d:prop")
				assert.NotNil(t, notFoundProp, "Should have prop element in 404 section")

				getetag := notFoundProp.FindElement("d:getetag")
				assert.NotNil(t, getetag, "Should have getetag in 404 section")
				assert.Equal(t, "", getetag.Text(), "Not found property should be empty")

				calendarColor := notFoundProp.FindElement("cs:calendar-color")
				assert.NotNil(t, calendarColor, "Should have calendar-color in 404 section")

				userPrincipal := notFoundProp.FindElement("d:current-user-principal")
				assert.NotNil(t, userPrincipal, "Should have current-user-principal in 404 section")
			},
		},
		{
			name: "All properties found",
			props: map[string]mo.Result[PropertyEncoder]{
				"displayname": mo.Ok[PropertyEncoder](&displayName{Value: "Test Calendar"}),
				"getetag":     mo.Ok[PropertyEncoder](&getEtag{Value: "\"etag12345\""}),
			},
			href: "/calendars/user1/calendar1/",
			expected: func(t *testing.T, doc *etree.Document) {
				// Should only have 200 propstat section
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.NotNil(t, okPropstat, "Should have 200 OK propstat")

				notFoundPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 404 Not Found']")
				assert.Nil(t, notFoundPropstat, "Should not have 404 Not Found propstat")

				// Check found properties
				okProp := okPropstat.FindElement("d:prop")
				displayname := okProp.FindElement("d:displayname")
				assert.Equal(t, "Test Calendar", displayname.Text())

				etag := okProp.FindElement("d:getetag")
				assert.Equal(t, "\"etag12345\"", etag.Text())
			},
		},
		{
			name: "All properties not found",
			props: map[string]mo.Result[PropertyEncoder]{
				"displayname": mo.Err[PropertyEncoder](ErrNotFound),
				"getetag":     mo.Err[PropertyEncoder](ErrNotFound),
			},
			href: "/calendars/user1/calendar1/",
			expected: func(t *testing.T, doc *etree.Document) {
				// Should only have 404 propstat section
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.Nil(t, okPropstat, "Should not have 200 OK propstat")

				notFoundPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 404 Not Found']")
				assert.NotNil(t, notFoundPropstat, "Should have 404 Not Found propstat")

				// Check not found properties
				notFoundProp := notFoundPropstat.FindElement("d:prop")
				assert.NotNil(t, notFoundProp.FindElement("d:displayname"))
				assert.NotNil(t, notFoundProp.FindElement("d:getetag"))
			},
		},
		{
			name:  "Empty properties",
			props: map[string]mo.Result[PropertyEncoder]{},
			href:  "/calendars/user1/calendar1/",
			expected: func(t *testing.T, doc *etree.Document) {
				// Both propstat sections should be missing
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.Nil(t, okPropstat, "Should not have 200 OK propstat")

				notFoundPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 404 Not Found']")
				assert.Nil(t, notFoundPropstat, "Should not have 404 Not Found propstat")

				// But should still have response with href
				response := doc.FindElement("//d:response")
				assert.NotNil(t, response, "Should have response element")

				href := response.FindElement("d:href")
				assert.NotNil(t, href, "Should have href element")
				assert.Equal(t, "/calendars/user1/calendar1/", href.Text())
			},
		},
		{
			name: "Complex CalDAV properties",
			props: map[string]mo.Result[PropertyEncoder]{
				"supported-calendar-component-set": mo.Ok[PropertyEncoder](&supportedCalendarComponentSet{
					Components: []string{"VEVENT", "VTODO"},
				}),
				"calendar-user-address-set": mo.Ok[PropertyEncoder](&calendarUserAddressSet{
					Addresses: []string{"mailto:user1@example.com", "mailto:user.one@example.org"},
				}),
				"current-user-privilege-set": mo.Ok[PropertyEncoder](&currentUserPrivilegeSet{
					Privileges: []string{"read", "write", "read-acl"},
				}),
			},
			href: "/principals/users/user1/",
			expected: func(t *testing.T, doc *etree.Document) {
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.NotNil(t, okPropstat, "Should have 200 OK propstat")

				// Test supported-calendar-component-set
				compSet := doc.FindElement("//d:prop/cal:supported-calendar-component-set")
				assert.NotNil(t, compSet, "Should have supported-calendar-component-set")

				comps := compSet.FindElements("./cal:comp")
				assert.Equal(t, 2, len(comps), "Should have 2 component elements")
				assert.Equal(t, "VEVENT", comps[0].SelectAttr("name").Value)
				assert.Equal(t, "VTODO", comps[1].SelectAttr("name").Value)

				// Test calendar-user-address-set
				addrSet := doc.FindElement("//d:prop/cal:calendar-user-address-set")
				assert.NotNil(t, addrSet, "Should have calendar-user-address-set")

				hrefs := addrSet.FindElements("./d:href")
				assert.Equal(t, 2, len(hrefs), "Should have 2 href elements")
				assert.Equal(t, "mailto:user1@example.com", hrefs[0].Text())
				assert.Equal(t, "mailto:user.one@example.org", hrefs[1].Text())

				// Test current-user-privilege-set
				privSet := doc.FindElement("//d:prop/d:current-user-privilege-set")
				assert.NotNil(t, privSet, "Should have current-user-privilege-set")

				privs := privSet.FindElements("./d:privilege")
				assert.Equal(t, 3, len(privs), "Should have 3 privilege elements")
			},
		},
		{
			name: "Mixed error types",
			props: map[string]mo.Result[PropertyEncoder]{
				"displayname":            mo.Ok[PropertyEncoder](&displayName{Value: "Test Calendar"}),
				"getetag":                mo.Err[PropertyEncoder](ErrNotFound),
				"current-user-principal": mo.Err[PropertyEncoder](ErrForbidden),
				"resourcetype":           mo.Err[PropertyEncoder](ErrInternal),
				"getcontenttype":         mo.Err[PropertyEncoder](ErrBadRequest),
			},
			href: "/calendars/user1/calendar1/",
			expected: func(t *testing.T, doc *etree.Document) {
				// Verify we have different status sections for each error type
				okPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 200 OK']")
				assert.NotNil(t, okPropstat, "Should have 200 OK propstat")

				notFoundPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 404 Not Found']")
				assert.NotNil(t, notFoundPropstat, "Should have 404 Not Found propstat")

				forbiddenPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 403 Forbidden']")
				assert.NotNil(t, forbiddenPropstat, "Should have 403 Forbidden propstat")

				internalPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 500 Internal Server Error']")
				assert.NotNil(t, internalPropstat, "Should have 500 Internal Server Error propstat")

				badRequestPropstat := doc.FindElement("//d:response/d:propstat[d:status='HTTP/1.1 400 Bad Request']")
				assert.NotNil(t, badRequestPropstat, "Should have 400 Bad Request propstat")

				// Check that each prop is in the right section
				notFoundProp := notFoundPropstat.FindElement("d:prop/d:getetag")
				assert.NotNil(t, notFoundProp, "getetag should be in 404 section")

				forbiddenProp := forbiddenPropstat.FindElement("d:prop/d:current-user-principal")
				assert.NotNil(t, forbiddenProp, "current-user-principal should be in 403 section")

				internalProp := internalPropstat.FindElement("d:prop/d:resourcetype")
				assert.NotNil(t, internalProp, "resourcetype should be in 500 section")

				badRequestProp := badRequestPropstat.FindElement("d:prop/d:getcontenttype")
				assert.NotNil(t, badRequestProp, "getcontenttype should be in 400 section")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := EncodeResponse(tt.props, tt.href)

			// Apply the test's validation function
			tt.expected(t, doc)

			// Optional: output XML for debugging
			// xmlStr, _ := doc.WriteToString()
			// t.Logf("XML Output:\n%s", xmlStr)
		})
	}
}

func TestEncodeResponseHref(t *testing.T) {
	// Test that the href parameter is properly used
	props := map[string]mo.Result[PropertyEncoder]{
		"displayname": mo.Ok[PropertyEncoder](&displayName{Value: "Test"}),
	}

	customHref := "/custom/path/to/resource/"
	doc := EncodeResponse(props, customHref)

	href := doc.FindElement("//d:response/d:href")
	assert.NotNil(t, href, "Should have href element")
	assert.Equal(t, customHref, href.Text(), "Href should match input value")
}

func TestMergeResponses(t *testing.T) {
	// Helper function to create a test document with responses
	createTestDoc := func(resourcePaths []string) *etree.Document {
		doc := etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
		multistatus := doc.CreateElement("d:multistatus")
		multistatus.CreateAttr("xmlns:d", "DAV:")
		multistatus.CreateAttr("xmlns:cal", "urn:ietf:params:xml:ns:caldav")
		multistatus.CreateAttr("xmlns:cs", "http://calendarserver.org/ns/")

		for _, path := range resourcePaths {
			response := multistatus.CreateElement("d:response")
			href := response.CreateElement("d:href")
			href.SetText(path)

			// Add a propstat for testing
			propstat := response.CreateElement("d:propstat")
			prop := propstat.CreateElement("d:prop")
			displayname := prop.CreateElement("d:displayname")
			displayname.SetText("Resource " + path)

			// Add a CalDAV property to test namespace preservation
			if strings.Contains(path, "cal") {
				calProp := prop.CreateElement("cal:calendar-color")
				calProp.SetText("#FF0000")
			}

			status := propstat.CreateElement("d:status")
			status.SetText("HTTP/1.1 200 OK")
		}

		return doc
	}

	// Helper function to create an invalid document (no multistatus)
	createInvalidDoc := func() *etree.Document {
		doc := etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
		root := doc.CreateElement("d:root")
		root.CreateAttr("xmlns:d", "DAV:")
		return doc
	}

	t.Run("Empty input", func(t *testing.T) {
		mergedDoc, err := MergeResponses([]*etree.Document{})
		assert.Nil(t, mergedDoc, "Should return nil document for empty input")
		assert.Error(t, err, "Should return error for empty input")
		assert.Equal(t, "no documents to merge", err.Error())
	})

	t.Run("Single document", func(t *testing.T) {
		doc := createTestDoc([]string{"/calendars/user1/cal1/"})
		mergedDoc, err := MergeResponses([]*etree.Document{doc})
		assert.NoError(t, err, "Should not return error for single document")
		assert.Same(t, doc, mergedDoc, "Should return the original document")
	})

	t.Run("Multiple documents with one response each", func(t *testing.T) {
		doc1 := createTestDoc([]string{"/calendars/user1/cal1/"})
		doc2 := createTestDoc([]string{"/calendars/user1/cal2/"})
		doc3 := createTestDoc([]string{"/calendars/user1/cal3/"})

		mergedDoc, err := MergeResponses([]*etree.Document{doc1, doc2, doc3})
		assert.NoError(t, err, "Should not return error for valid documents")

		// Check the merged document structure
		responses := mergedDoc.FindElements("//d:multistatus/d:response")
		assert.Equal(t, 3, len(responses), "Merged doc should have 3 responses")

		hrefs := mergedDoc.FindElements("//d:multistatus/d:response/d:href")
		assert.Equal(t, 3, len(hrefs), "Merged doc should have 3 hrefs")

		// Check all paths are present
		paths := []string{"/calendars/user1/cal1/", "/calendars/user1/cal2/", "/calendars/user1/cal3/"}
		foundPaths := make(map[string]bool)
		for _, href := range hrefs {
			foundPaths[href.Text()] = true
		}

		for _, path := range paths {
			assert.True(t, foundPaths[path], "Path %s should be in merged document", path)
		}

		// Check namespace declarations
		multistatus := mergedDoc.FindElement("//d:multistatus")
		assert.NotNil(t, multistatus.SelectAttr("xmlns:d"))
		assert.NotNil(t, multistatus.SelectAttr("xmlns:cal"))
		assert.Equal(t, "DAV:", multistatus.SelectAttr("xmlns:d").Value)
		assert.Equal(t, "urn:ietf:params:xml:ns:caldav", multistatus.SelectAttr("xmlns:cal").Value)
		assert.Equal(t, "http://calendarserver.org/ns/", multistatus.SelectAttr("xmlns:cs").Value)
	})

	t.Run("Multiple documents with multiple responses", func(t *testing.T) {
		// Create docs with multiple responses each
		doc1 := createTestDoc([]string{"/calendars/user1/cal1/", "/calendars/user1/cal1/event1.ics"})
		doc2 := createTestDoc([]string{"/calendars/user1/cal2/", "/calendars/user1/cal2/event2.ics"})

		mergedDoc, err := MergeResponses([]*etree.Document{doc1, doc2})
		assert.NoError(t, err, "Should not return error for valid documents")

		// Check the merged document structure
		responses := mergedDoc.FindElements("//d:multistatus/d:response")
		assert.Equal(t, 4, len(responses), "Merged doc should have 4 responses")

		hrefs := mergedDoc.FindElements("//d:multistatus/d:response/d:href")
		assert.Equal(t, 4, len(hrefs), "Merged doc should have 4 hrefs")

		// Check all paths are present
		paths := []string{
			"/calendars/user1/cal1/",
			"/calendars/user1/cal1/event1.ics",
			"/calendars/user1/cal2/",
			"/calendars/user1/cal2/event2.ics",
		}

		foundPaths := make(map[string]bool)
		for _, href := range hrefs {
			foundPaths[href.Text()] = true
		}

		for _, path := range paths {
			assert.True(t, foundPaths[path], "Path %s should be in merged document", path)
		}

		// Check that props are copied correctly
		displaynames := mergedDoc.FindElements("//d:multistatus/d:response/d:propstat/d:prop/d:displayname")
		assert.Equal(t, 4, len(displaynames), "Should have 4 displayname properties")

		// Check that CalDAV properties are preserved
		calendarColors := mergedDoc.FindElements("//d:multistatus/d:response/d:propstat/d:prop/cal:calendar-color")
		assert.Equal(t, 4, len(calendarColors), "Should have 4 calendar-color properties")
	})

	t.Run("Invalid document", func(t *testing.T) {
		invalidDoc := createInvalidDoc()
		doc1 := createTestDoc([]string{"/calendars/user1/cal1/"})

		// Test with invalid doc first
		mergedDoc, err := MergeResponses([]*etree.Document{invalidDoc, doc1})
		assert.Nil(t, mergedDoc, "Should return nil when first doc is invalid")
		assert.Error(t, err, "Should return error when first doc is invalid")
		assert.Equal(t, "first document missing multistatus element", err.Error())

		// Test with invalid doc second (should still work, only using first doc for namespace info)
		mergedDoc, err = MergeResponses([]*etree.Document{doc1, invalidDoc})
		assert.NoError(t, err, "Should not error when later docs are invalid")

		// Should still process responses from valid documents
		responses := mergedDoc.FindElements("//d:multistatus/d:response")
		assert.Equal(t, 1, len(responses), "Merged doc should have 1 response")
	})

	t.Run("Complex property handling", func(t *testing.T) {
		// Create a more complex document with error statuses and nested properties
		doc := etree.NewDocument()
		doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
		multistatus := doc.CreateElement("d:multistatus")
		multistatus.CreateAttr("xmlns:d", "DAV:")
		multistatus.CreateAttr("xmlns:cal", "urn:ietf:params:xml:ns:caldav")

		response := multistatus.CreateElement("d:response")
		href := response.CreateElement("d:href")
		href.SetText("/calendars/user1/complex/")

		// Add a propstat with nested elements
		propstat := response.CreateElement("d:propstat")
		prop := propstat.CreateElement("d:prop")
		resourcetype := prop.CreateElement("d:resourcetype")
		resourcetype.CreateElement("d:collection")
		resourcetype.CreateElement("cal:calendar")

		status := propstat.CreateElement("d:status")
		status.SetText("HTTP/1.1 200 OK")

		// Add another response to merge
		response2 := multistatus.CreateElement("d:response")
		href2 := response2.CreateElement("d:href")
		href2.SetText("/calendars/user1/complex/event.ics")

		propstat2 := response2.CreateElement("d:propstat")
		prop2 := propstat2.CreateElement("d:prop")
		etag := prop2.CreateElement("d:getetag")
		etag.SetText("\"etag123456\"")

		status2 := propstat2.CreateElement("d:status")
		status2.SetText("HTTP/1.1 200 OK")

		// Create a simple second document
		doc2 := createTestDoc([]string{"/principals/user1/"})

		// Merge the documents
		mergedDoc, err := MergeResponses([]*etree.Document{doc, doc2})
		assert.NoError(t, err, "Should not return error for complex documents")

		// Check that complex elements were properly merged
		responses := mergedDoc.FindElements("//d:multistatus/d:response")
		assert.Equal(t, 3, len(responses), "Merged doc should have 3 responses")

		// Verify nested elements were preserved
		resourcetypes := mergedDoc.FindElements("//d:resourcetype/d:collection")
		assert.Equal(t, 1, len(resourcetypes), "Should have preserved nested collection element")

		calendars := mergedDoc.FindElements("//d:resourcetype/cal:calendar")
		assert.Equal(t, 1, len(calendars), "Should have preserved nested calendar element")

		// Verify etag was preserved
		etags := mergedDoc.FindElements("//d:getetag")
		assert.Equal(t, 1, len(etags), "Should have preserved etag element")
		assert.Equal(t, "\"etag123456\"", etags[0].Text(), "Etag value should be preserved")
	})
}
