package mkcalendar

import (
	"reflect"
	"testing"

	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/stretchr/testify/assert"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name     string
		xmlInput string
		want     map[string]reflect.Type // Expected property names and their types
	}{
		{
			name: "Basic MKCALENDAR request",
			xmlInput: `<?xml version="1.0" encoding="utf-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Work Calendar</D:displayname>
      <C:calendar-description>Calendar for work-related events</C:calendar-description>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
        <C:comp name="VTODO"/>
      </C:supported-calendar-component-set>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{
				"displayname":                      reflect.TypeOf(new(props.DisplayName)),
				"calendar-description":             reflect.TypeOf(new(props.CalendarDescription)),
				"supported-calendar-component-set": reflect.TypeOf(new(props.SupportedCalendarComponentSet)),
			},
		},
		{
			name: "Extended properties with Apple and timezone",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" 
              xmlns:D="DAV:"
              xmlns:CS="http://calendarserver.org/ns/">
  <D:set>
    <D:prop>
      <D:displayname>Personal</D:displayname>
      <C:calendar-description>Personal Calendar</C:calendar-description>
      <C:calendar-timezone>BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Example Corp.//CalDAV Client//EN\r\nBEGIN:VTIMEZONE\r\nTZID:America/New_York\r\nEND:VTIMEZONE\r\nEND:VCALENDAR</C:calendar-timezone>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
      </C:supported-calendar-component-set>
      <CS:calendar-color>#FF0000</CS:calendar-color>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{
				"displayname":                      reflect.TypeOf(new(props.DisplayName)),
				"calendar-description":             reflect.TypeOf(new(props.CalendarDescription)),
				"calendar-timezone":                reflect.TypeOf(new(props.CalendarTimezone)),
				"supported-calendar-component-set": reflect.TypeOf(new(props.SupportedCalendarComponentSet)),
				"calendar-color":                   reflect.TypeOf(new(props.CalendarColor)),
			},
		},
		{
			name: "Google extensions",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" 
              xmlns:D="DAV:"
              xmlns:G="http://schemas.google.com/gCal/2005">
  <D:set>
    <D:prop>
      <D:displayname>Google Calendar</D:displayname>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
      </C:supported-calendar-component-set>
      <G:timezone>America/Los_Angeles</G:timezone>
      <G:color>#4A86E8</G:color>
      <G:selected>true</G:selected>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{
				"displayname":                      reflect.TypeOf(new(props.DisplayName)),
				"supported-calendar-component-set": reflect.TypeOf(new(props.SupportedCalendarComponentSet)),
				"timezone":                         reflect.TypeOf(new(props.Timezone)),
				"color":                            reflect.TypeOf(new(props.Color)),
				"selected":                         reflect.TypeOf(new(props.Selected)),
			},
		},
		{
			name: "Empty properties",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Malformed XML",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Bad XML</
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Missing set element",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:prop>
    <D:displayname>Missing set</D:displayname>
  </D:prop>
</C:mkcalendar>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Missing prop element",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:displayname>Missing prop</D:displayname>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{},
		},
		{
			name: "Unknown properties",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Known property</D:displayname>
      <D:unknown-property>Should be skipped</D:unknown-property>
      <C:unknown-caldav-property>Should also be skipped</C:unknown-caldav-property>
    </D:prop>
  </D:set>
</C:mkcalendar>`,
			want: map[string]reflect.Type{
				"displayname": reflect.TypeOf(new(props.DisplayName)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRequest(tt.xmlInput)

			// For invalid XML, we expect an error
			if tt.name == "Malformed XML" {
				assert.Error(t, err, "Should return error for malformed XML")
				return
			}

			assert.NoError(t, err)

			// Check if the number of properties matches
			assert.Equal(t, len(tt.want), len(got),
				"Result should have %d properties, got %d", len(tt.want), len(got))

			// Check each expected property type
			for propName, expectedType := range tt.want {
				prop, exists := got[propName]
				assert.True(t, exists, "Property %s should exist in result", propName)

				if exists {
					// Check if the value is of the expected type
					actualType := reflect.TypeOf(prop)
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

func TestParseRequestPropertyValues(t *testing.T) {
	// This test validates that property values are correctly parsed
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<C:mkcalendar xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>Test Calendar</D:displayname>
      <C:calendar-description>Calendar description</C:calendar-description>
      <C:supported-calendar-component-set>
        <C:comp name="VEVENT"/>
        <C:comp name="VTODO"/>
      </C:supported-calendar-component-set>
    </D:prop>
  </D:set>
</C:mkcalendar>`

	property, err := ParseRequest(xmlInput)
	assert.NoError(t, err)

	// Verify displayname
	displayname, exists := property["displayname"]
	assert.True(t, exists)
	dispNameProp, ok := displayname.(*props.DisplayName)
	assert.True(t, ok)
	assert.Equal(t, "Test Calendar", dispNameProp.Value)

	// Verify calendar-description
	calDesc, exists := property["calendar-description"]
	assert.True(t, exists)
	calDescProp, ok := calDesc.(*props.CalendarDescription)
	assert.True(t, ok)
	assert.Equal(t, "Calendar description", calDescProp.Value)

	// Verify supported-calendar-component-set
	compSet, exists := property["supported-calendar-component-set"]
	assert.True(t, exists)
	compSetProp, ok := compSet.(*props.SupportedCalendarComponentSet)
	assert.True(t, ok)
	assert.Equal(t, 2, len(compSetProp.Components))
	assert.Contains(t, compSetProp.Components, "VEVENT")
	assert.Contains(t, compSetProp.Components, "VTODO")
}
