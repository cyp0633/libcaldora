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
				"displayname":     reflect.TypeOf(new(DisplayName)),
				"resourcetype":    reflect.TypeOf(new(Resourcetype)),
				"getetag":         reflect.TypeOf(new(GetEtag)),
				"getlastmodified": reflect.TypeOf(new(GetLastModified)),
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
				"displayname":                      reflect.TypeOf(new(DisplayName)),
				"calendar-description":             reflect.TypeOf(new(CalendarDescription)),
				"resourcetype":                     reflect.TypeOf(new(Resourcetype)),
				"supported-calendar-component-set": reflect.TypeOf(new(SupportedCalendarComponentSet)),
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
				"getctag": reflect.TypeOf(new(GetCTag)),
				"color":   reflect.TypeOf(new(Color)),
				"invite":  reflect.TypeOf(new(Invite)),
				"hidden":  reflect.TypeOf(new(Hidden)),
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
				"displayname": reflect.TypeOf(new(DisplayName)),
				"getetag":     reflect.TypeOf(new(GetEtag)),
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
				"displayname": mo.Ok[PropertyEncoder](&DisplayName{Value: "Test Calendar"}),
				"resourcetype": mo.Ok[PropertyEncoder](&Resourcetype{
					Types: []string{"collection", "calendar"},
				}),
				"getetag":                mo.Err[PropertyEncoder](ErrNotFound),
				"calendar-color":         mo.Err[PropertyEncoder](ErrNotFound),
				"getcontenttype":         mo.Ok[PropertyEncoder](&GetContentType{Value: "text/calendar"}),
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
				"displayname": mo.Ok[PropertyEncoder](&DisplayName{Value: "Test Calendar"}),
				"getetag":     mo.Ok[PropertyEncoder](&GetEtag{Value: "\"etag12345\""}),
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
				"supported-calendar-component-set": mo.Ok[PropertyEncoder](&SupportedCalendarComponentSet{
					Components: []string{"VEVENT", "VTODO"},
				}),
				"calendar-user-address-set": mo.Ok[PropertyEncoder](&CalendarUserAddressSet{
					Addresses: []string{"mailto:user1@example.com", "mailto:user.one@example.org"},
				}),
				"current-user-privilege-set": mo.Ok[PropertyEncoder](&CurrentUserPrivilegeSet{
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
				"displayname":            mo.Ok[PropertyEncoder](&DisplayName{Value: "Test Calendar"}),
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
		"displayname": mo.Ok[PropertyEncoder](&DisplayName{Value: "Test"}),
	}

	customHref := "/custom/path/to/resource/"
	doc := EncodeResponse(props, customHref)

	href := doc.FindElement("//d:response/d:href")
	assert.NotNil(t, href, "Should have href element")
	assert.Equal(t, customHref, href.Text(), "Href should match input value")
}

// Helper function to create a minimal valid sub-response document string
// Mimics the structure produced by EncodeResponse for testing MergeResponses
func createSubResponseXML(href, etag, description string) string {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
	multistatus := doc.CreateElement("d:multistatus")
	multistatus.Space = "d"
	// Add namespaces as EncodeResponse would
	for prefix, uri := range namespaceMap { // Assumes namespaceMap is accessible
		multistatus.CreateAttr("xmlns:"+prefix, uri)
	}

	response := multistatus.CreateElement("d:response")
	response.Space = "d"

	hrefElem := response.CreateElement("d:href")
	hrefElem.Space = "d"
	hrefElem.SetText(href)

	propstat := response.CreateElement("d:propstat")
	propstat.Space = "d"

	prop := propstat.CreateElement("d:prop")
	prop.Space = "d"

	etagElem := prop.CreateElement("d:getetag")
	etagElem.Space = "d"
	etagElem.SetText(etag)

	if description != "" {
		descElem := prop.CreateElement("cs:calendar-description")
		descElem.Space = "cs" // Assuming 'cs' is in namespaceMap
		descElem.SetText(description)
	}

	status := propstat.CreateElement("d:status")
	status.Space = "d"
	status.SetText("HTTP/1.1 200 OK")

	xmlStr, _ := doc.WriteToString()
	return xmlStr
}

// Helper function to parse XML string into etree.Document for tests
func parseXML(t *testing.T, xmlStr string) *etree.Document {
	doc := etree.NewDocument()
	err := doc.ReadFromString(xmlStr)
	assert.NoError(t, err, "Failed to parse helper XML")
	return doc
}

// --- Test Suite ---

func TestMergeResponses(t *testing.T) {
	// Setup: Ensure namespaceMap is defined (copy from main code or define here)
	// Assuming namespaceMap is accessible or redefined for test scope
	// namespaceMap = map[string]string{
	// 	"d":  "DAV:",
	// 	"cs": "urn:ietf:params:xml:ns:caldav",
	// }

	t.Run("Merge multiple valid documents", func(t *testing.T) {
		// Arrange
		doc1XML := createSubResponseXML("/cal/res1", `"etag1"`, "Desc 1")
		doc2XML := createSubResponseXML("/cal/res2", `"etag2"`, "Desc 2")
		doc1 := parseXML(t, doc1XML)
		doc2 := parseXML(t, doc2XML)
		inputDocs := []*etree.Document{doc1, doc2}

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)

		// Verify root element and namespaces
		root := mergedDoc.Root()
		assert.NotNil(t, root)
		assert.Equal(t, "d", root.Space, "Root element should have 'd' namespace prefix")
		assert.Equal(t, "multistatus", root.Tag, "Root element should be 'multistatus'")
		assert.Equal(t, namespaceMap["d"], root.SelectAttrValue("xmlns:d", ""), "DAV namespace declaration missing")
		assert.Equal(t, namespaceMap["cs"], root.SelectAttrValue("xmlns:cs", ""), "CalDAV namespace declaration missing")

		// Verify correct number of response elements
		responses := root.FindElements("./d:response") // Find direct children
		assert.Len(t, responses, 2, "Should contain two response elements")

		// Verify content of responses (basic check)
		hrefs := []string{}
		for _, resp := range responses {
			hrefElem := resp.FindElement("./d:href")
			assert.NotNil(t, hrefElem)
			hrefs = append(hrefs, hrefElem.Text())
		}
		assert.Contains(t, hrefs, "/cal/res1")
		assert.Contains(t, hrefs, "/cal/res2")

		// Optional: More detailed check using XML string comparison
		mergedDoc.Indent(2) // Make output readable/comparable
		actualXML, _ := mergedDoc.WriteToString()
		// Define expected XML carefully, ensuring namespace order doesn't matter for assertion if possible
		// Or use a more robust XML comparison library if needed.
		t.Logf("Merged XML:\n%s", actualXML) // Log for debugging
		// Add specific string asserts if required, e.g., assert.Contains(t, actualXML, ...)
	})

	t.Run("Merge empty slice", func(t *testing.T) {
		// Arrange
		inputDocs := []*etree.Document{}

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)

		root := mergedDoc.Root()
		assert.NotNil(t, root)
		assert.Equal(t, "multistatus", root.Tag, "Root element tag should be 'multistatus'")
		assert.Equal(t, "d", root.Space, "Root element namespace prefix should be 'd'")
		assert.Equal(t, namespaceMap["d"], root.SelectAttrValue("xmlns:d", ""), "DAV namespace declaration missing")
		assert.Equal(t, namespaceMap["cs"], root.SelectAttrValue("xmlns:cs", ""), "CalDAV namespace declaration missing")

		responses := root.FindElements("./d:response")
		assert.Len(t, responses, 0, "Should contain zero response elements")
	})

	t.Run("Merge nil slice", func(t *testing.T) {
		// Arrange
		var inputDocs []*etree.Document = nil // Explicitly nil slice

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)

		root := mergedDoc.Root()
		assert.NotNil(t, root)
		assert.Equal(t, "multistatus", root.Tag, "Root element tag should be 'multistatus'")
		assert.Equal(t, "d", root.Space, "Root element namespace prefix should be 'd'")
		assert.Equal(t, namespaceMap["d"], root.SelectAttrValue("xmlns:d", ""), "DAV namespace declaration missing")
		assert.Equal(t, namespaceMap["cs"], root.SelectAttrValue("xmlns:cs", ""), "CalDAV namespace declaration missing")

		responses := root.FindElements("./d:response")
		assert.Len(t, responses, 0, "Should contain zero response elements")
	})

	t.Run("Merge slice with nil documents", func(t *testing.T) {
		// Arrange
		doc1XML := createSubResponseXML("/cal/res1", `"etag1"`, "Desc 1")
		doc3XML := createSubResponseXML("/cal/res3", `"etag3"`, "Desc 3")
		doc1 := parseXML(t, doc1XML)
		doc3 := parseXML(t, doc3XML)
		inputDocs := []*etree.Document{doc1, nil, doc3, nil} // Include nils

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)

		root := mergedDoc.Root()
		assert.NotNil(t, root)
		assert.Equal(t, "multistatus", root.Tag, "Root element tag should be 'multistatus'")
		assert.Equal(t, "d", root.Space, "Root element namespace prefix should be 'd'")

		responses := root.FindElements("./d:response")
		assert.Len(t, responses, 2, "Should contain two response elements (non-nil inputs)")

		hrefs := []string{}
		for _, resp := range responses {
			hrefElem := resp.FindElement("./d:href")
			assert.NotNil(t, hrefElem)
			hrefs = append(hrefs, hrefElem.Text())
		}
		assert.Contains(t, hrefs, "/cal/res1")
		assert.Contains(t, hrefs, "/cal/res3")
	})

	t.Run("Merge single document", func(t *testing.T) {
		// Arrange
		doc1XML := createSubResponseXML("/cal/single", `"etag_single"`, "Single Desc")
		doc1 := parseXML(t, doc1XML)
		inputDocs := []*etree.Document{doc1}

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)

		root := mergedDoc.Root()
		assert.NotNil(t, root)
		assert.Equal(t, "multistatus", root.Tag, "Root element tag should be 'multistatus'")
		assert.Equal(t, "d", root.Space, "Root element namespace prefix should be 'd'")
		assert.Equal(t, namespaceMap["d"], root.SelectAttrValue("xmlns:d", ""), "DAV namespace declaration missing")
		assert.Equal(t, namespaceMap["cs"], root.SelectAttrValue("xmlns:cs", ""), "CalDAV namespace declaration missing")

		responses := root.FindElements("./d:response")
		assert.Len(t, responses, 1, "Should contain one response element")

		hrefElem := responses[0].FindElement("./d:href")
		assert.NotNil(t, hrefElem)
		assert.Equal(t, "/cal/single", hrefElem.Text())

		etagElem := responses[0].FindElement("./d:propstat/d:prop/d:getetag")
		assert.NotNil(t, etagElem)
		assert.Equal(t, `"etag_single"`, etagElem.Text())

		descElem := responses[0].FindElement("./d:propstat/d:prop/cs:calendar-description")
		assert.NotNil(t, descElem)
		assert.Equal(t, "Single Desc", descElem.Text())
	})

	t.Run("Merge documents with different properties and statuses (ensure structure copied)", func(t *testing.T) {
		// Arrange: Create more complex/varied sub-responses if EncodeResponse supported errors properly
		// For now, using the basic helper which only does 200 OK
		doc1XML := createSubResponseXML("/cal/ok", `"etag_ok"`, "OK Desc")
		doc2XML := createSubResponseXML("/cal/ok-no-desc", `"etag_no_desc"`, "") // No optional description
		doc1 := parseXML(t, doc1XML)
		doc2 := parseXML(t, doc2XML)
		inputDocs := []*etree.Document{doc1, doc2}

		// Act
		mergedDoc, err := MergeResponses(inputDocs)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, mergedDoc)
		root := mergedDoc.Root()
		assert.NotNil(t, root)
		responses := root.FindElements("./d:response")
		assert.Len(t, responses, 2)

		// Check details for the first response
		resp1 := root.FindElement("./d:response[d:href='/cal/ok']")
		assert.NotNil(t, resp1)
		prop1 := resp1.FindElement("./d:propstat/d:prop")
		assert.NotNil(t, prop1)
		assert.NotNil(t, prop1.FindElement("./d:getetag"), "getetag missing in resp1")
		assert.NotNil(t, prop1.FindElement("./cs:calendar-description"), "calendar-description missing in resp1")

		// Check details for the second response
		resp2 := root.FindElement("./d:response[d:href='/cal/ok-no-desc']")
		assert.NotNil(t, resp2)
		prop2 := resp2.FindElement("./d:propstat/d:prop")
		assert.NotNil(t, prop2)
		assert.NotNil(t, prop2.FindElement("./d:getetag"), "getetag missing in resp2")
		assert.Nil(t, prop2.FindElement("./cs:calendar-description"), "calendar-description should be absent in resp2")

	})
}
