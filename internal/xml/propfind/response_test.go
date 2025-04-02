package propfind

import (
	"reflect"
	"strings"
	"testing"

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
