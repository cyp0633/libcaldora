package calendarmultiget

import (
	"testing"

	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/stretchr/testify/assert"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name            string
		xml             string
		expectedProps   []string
		expectedURIs    []string
		expectedSuccess bool
	}{
		{
			name: "Valid calendar-multiget request",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
  </D:prop>
  <D:href>/calendars/user/calendar/event1.ics</D:href>
  <D:href>/calendars/user/calendar/event2.ics</D:href>
</C:calendar-multiget>`,
			expectedProps:   []string{"getetag", "calendar-data"},
			expectedURIs:    []string{"/calendars/user/calendar/event1.ics", "/calendars/user/calendar/event2.ics"},
			expectedSuccess: true,
		},
		{
			name: "Valid calendar-multiget with more properties",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
    <D:displayname/>
    <D:resourcetype/>
  </D:prop>
  <D:href>/calendars/user/calendar/event1.ics</D:href>
</C:calendar-multiget>`,
			expectedProps:   []string{"getetag", "calendar-data", "displayname", "resourcetype"},
			expectedURIs:    []string{"/calendars/user/calendar/event1.ics"},
			expectedSuccess: true,
		},
		{
			name: "Empty calendar-multiget request",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
  </D:prop>
</C:calendar-multiget>`,
			expectedProps:   []string{},
			expectedURIs:    []string{},
			expectedSuccess: true,
		},
		{
			name: "Missing prop element",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:href>/calendars/user/calendar/event1.ics</D:href>
</C:calendar-multiget>`,
			expectedProps:   []string{},
			expectedURIs:    []string{},
			expectedSuccess: false,
		},
		{
			name: "Invalid XML",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <D:href>/calendars/user/calendar/event1.ics`,
			expectedProps:   []string{},
			expectedURIs:    []string{},
			expectedSuccess: false,
		},
		{
			name: "Unknown properties",
			xml: `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:nonexistent-property/>
  </D:prop>
  <D:href>/calendars/user/calendar/event1.ics</D:href>
</C:calendar-multiget>`,
			expectedProps:   []string{"getetag"},
			expectedURIs:    []string{"/calendars/user/calendar/event1.ics"},
			expectedSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propsMap, hrefs := ParseRequest(tt.xml)

			// Check if we got the expected result
			if !tt.expectedSuccess {
				if len(propsMap) > 0 || len(hrefs) > 0 {
					t.Errorf("Expected failure but got non-empty results: propsMap=%v, hrefs=%v", propsMap, hrefs)
				}
				return
			}

			// Check that all expected properties are present in the propsMap
			for _, propName := range tt.expectedProps {
				propResult, exists := propsMap[propName]
				if !assert.True(t, exists, "Property %s should exist in the response map", propName) {
					continue
				}

				// Verify that the property result is a success (not an error)
				if !assert.True(t, propResult.IsOk(), "Property %s should be a success result", propName) {
					continue
				}

				// Verify that the property has the correct encoder type
				encoder := propResult.MustGet()
				expectedEncoder, exists := props.PropNameToStruct[propName]

				if !assert.True(t, exists, "Property %s should have an expected encoder type", propName) {
					continue
				}

				assert.IsType(t, expectedEncoder, encoder, "Property %s has wrong encoder type", propName)
			}

			// Check that we don't have extra properties
			assert.Equal(t, len(tt.expectedProps), len(propsMap), "Number of properties should match expected")

			// Check hrefs
			assert.Equal(t, tt.expectedURIs, hrefs, "URIs should match expected")
		})
	}
}

func TestParseRequestWithNamespacedProperties(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
  </D:prop>
  <D:href>/calendars/user/calendar/event1.ics</D:href>
</C:calendar-multiget>`

	propsMap, hrefs := ParseRequest(xml)

	// Check that the namespaced properties are correctly parsed
	assert.Len(t, propsMap, 2)
	assert.Contains(t, propsMap, "getetag")
	assert.Contains(t, propsMap, "calendar-data")

	// Verify the property encoders
	getetag, exists := propsMap["getetag"]
	assert.True(t, exists)
	assert.True(t, getetag.IsOk())
	assert.IsType(t, &props.GetEtag{}, getetag.MustGet())

	calendarData, exists := propsMap["calendar-data"]
	assert.True(t, exists)
	assert.True(t, calendarData.IsOk())
	assert.IsType(t, &props.CalendarData{}, calendarData.MustGet())

	// Check hrefs
	assert.Equal(t, []string{"/calendars/user/calendar/event1.ics"}, hrefs)
}
