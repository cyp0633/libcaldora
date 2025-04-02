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
			got := ParseRequest(tt.xmlInput)

			// Check if the number of properties matches
			assert.Equal(t, len(tt.want), len(got),
				"Result should have %d properties, got %d", len(tt.want), len(got))

			// Check each expected property type
			for propName, expectedType := range tt.want {
				option, exists := got[propName]
				assert.True(t, exists, "Property %s should exist in result", propName)

				if exists {
					// Check if the option has a value
					assert.True(t, option.IsPresent(), "Property %s should have a value", propName)

					// Check if the value is of the expected type
					value := option.MustGet()
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
	got := ParseRequest(xmlInput)

	// Check if all properties are correctly parsed
	assert.Equal(t, len(expectedProps), len(got),
		"Should have parsed all %d properties", len(expectedProps))

	// Check each property type
	for propName, expectedType := range expectedProps {
		option, exists := got[propName]
		assert.True(t, exists, "Property %s should exist in result", propName)

		if exists {
			// Check if the option has a value
			assert.True(t, option.IsPresent(), "Property %s should have a value", propName)

			// Check if the value is of the expected type
			value := option.MustGet()
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
		props    map[string]mo.Option[PropertyEncoder]
		href     string
		expected func(t *testing.T, doc *etree.Document)
	}{
		{
			name: "Mix of found and not found properties",
			props: map[string]mo.Option[PropertyEncoder]{
				"displayname": mo.Some(PropertyEncoder(&displayName{Value: "Test Calendar"})),
				"resourcetype": mo.Some(PropertyEncoder(&resourcetype{
					Types: []string{"collection", "calendar"},
				})),
				"getetag":                mo.None[PropertyEncoder](),
				"calendar-color":         mo.None[PropertyEncoder](),
				"getcontenttype":         mo.Some(PropertyEncoder(&getContentType{Value: "text/calendar"})),
				"current-user-principal": mo.None[PropertyEncoder](),
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
			props: map[string]mo.Option[PropertyEncoder]{
				"displayname": mo.Some(PropertyEncoder(&displayName{Value: "Test Calendar"})),
				"getetag":     mo.Some(PropertyEncoder(&getEtag{Value: "\"etag12345\""})),
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
			props: map[string]mo.Option[PropertyEncoder]{
				"displayname": mo.None[PropertyEncoder](),
				"getetag":     mo.None[PropertyEncoder](),
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
			props: map[string]mo.Option[PropertyEncoder]{},
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
			props: map[string]mo.Option[PropertyEncoder]{
				"supported-calendar-component-set": mo.Some(PropertyEncoder(&supportedCalendarComponentSet{
					Components: []string{"VEVENT", "VTODO"},
				})),
				"calendar-user-address-set": mo.Some(PropertyEncoder(&calendarUserAddressSet{
					Addresses: []string{"mailto:user1@example.com", "mailto:user.one@example.org"},
				})),
				"current-user-privilege-set": mo.Some(PropertyEncoder(&currentUserPrivilegeSet{
					Privileges: []string{"read", "write", "read-acl"},
				})),
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
	props := map[string]mo.Option[PropertyEncoder]{
		"displayname": mo.Some(PropertyEncoder(&displayName{Value: "Test"})),
	}

	customHref := "/custom/path/to/resource/"
	doc := EncodeResponse(props, customHref)

	href := doc.FindElement("//d:response/d:href")
	assert.NotNil(t, href, "Should have href element")
	assert.Equal(t, customHref, href.Text(), "Href should match input value")
}
