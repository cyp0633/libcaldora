package calendarquery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRequest_Complete(t *testing.T) {
	// Complete calendar-query with both prop and filter
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
    <D:displayname/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="20240101T000000Z" end="20240131T235959Z"/>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Check properties were parsed correctly
	assert.Len(t, propsMap, 3)
	assert.Contains(t, propsMap, "getetag")
	assert.Contains(t, propsMap, "calendar-data")
	assert.Contains(t, propsMap, "displayname")

	// Check filter was parsed correctly
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)
	assert.Len(t, filter.Children, 1)
	assert.Equal(t, "VEVENT", filter.Children[0].Component)

	// Check time range
	assert.NotNil(t, filter.Children[0].TimeRange)
	expectedStart, _ := time.Parse("20060102T150405Z", "20240101T000000Z")
	expectedEnd, _ := time.Parse("20060102T150405Z", "20240131T235959Z")
	assert.Equal(t, expectedStart, *filter.Children[0].TimeRange.Start)
	assert.Equal(t, expectedEnd, *filter.Children[0].TimeRange.End)
}

func TestParseRequest_OnlyProps(t *testing.T) {
	// Calendar-query with only prop section
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
    <D:resourcetype/>
  </D:prop>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Check properties were parsed correctly
	assert.Len(t, propsMap, 3)
	assert.Contains(t, propsMap, "getetag")
	assert.Contains(t, propsMap, "calendar-data")
	assert.Contains(t, propsMap, "resourcetype")

	// Check filter is nil
	assert.Nil(t, filter)
}

func TestParseRequest_OnlyFilter(t *testing.T) {
	// Calendar-query with only filter section
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:prop-filter name="SUMMARY">
          <C:text-match>Meeting</C:text-match>
        </C:prop-filter>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Check properties map is empty
	assert.Empty(t, propsMap)

	// Check filter was parsed correctly
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)
	assert.Len(t, filter.Children, 1)
	assert.Equal(t, "VEVENT", filter.Children[0].Component)
	assert.Len(t, filter.Children[0].PropFilters, 1)
	assert.Equal(t, "SUMMARY", filter.Children[0].PropFilters[0].Name)
	assert.Equal(t, "Meeting", filter.Children[0].PropFilters[0].TextMatch.Value)
}

func TestParseRequest_InvalidXML(t *testing.T) {
	// Invalid XML
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="20240101T000000Z" end="20240131T235959Z"/>`

	propsMap, filter, err := ParseRequest(xml)

	// Check error occurred
	assert.Error(t, err)

	// Maps and filter should be empty or nil
	assert.Empty(t, propsMap)
	assert.Nil(t, filter)
}

func TestParseRequest_Empty(t *testing.T) {
	// Empty XML
	xml := ``

	propsMap, filter, err := ParseRequest(xml)

	// Check error occurred
	assert.Error(t, err)

	// Maps and filter should be empty or nil
	assert.Empty(t, propsMap)
	assert.Nil(t, filter)
}

func TestParseRequest_MixedProps(t *testing.T) {
	// XML with both known and unknown properties
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
    <D:unknown-prop/>
    <C:unsupported-prop/>
  </D:prop>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Only known properties should be in the map
	assert.Len(t, propsMap, 2)
	assert.Contains(t, propsMap, "getetag")
	assert.Contains(t, propsMap, "calendar-data")

	// Unknown properties should not be included
	assert.NotContains(t, propsMap, "unknown-prop")
	assert.NotContains(t, propsMap, "unsupported-prop")

	// No filter
	assert.Nil(t, filter)
}

func TestParseRequest_ComplexFilter(t *testing.T) {
	// Complex filter with nested elements and multiple prop-filters
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="20240101T000000Z" end="20240131T235959Z"/>
        <C:prop-filter name="SUMMARY">
          <C:text-match collation="i;unicode-casemap" match-type="contains">Meeting</C:text-match>
        </C:prop-filter>
        <C:prop-filter name="ATTENDEE">
          <C:param-filter name="PARTSTAT">
            <C:text-match>ACCEPTED</C:text-match>
          </C:param-filter>
        </C:prop-filter>
        <C:prop-filter name="LOCATION">
          <C:is-not-defined/>
        </C:prop-filter>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Check properties
	assert.Len(t, propsMap, 2)

	// Check filter structure
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)

	vevent := filter.Children[0]
	assert.Equal(t, "VEVENT", vevent.Component)

	// Check time range
	assert.NotNil(t, vevent.TimeRange)

	// Check prop filters
	assert.Len(t, vevent.PropFilters, 3)

	// Map prop filters for easier testing
	propFilters := make(map[string]int)
	for i, pf := range vevent.PropFilters {
		propFilters[pf.Name] = i
	}

	// Check SUMMARY filter
	summaryIdx := propFilters["SUMMARY"]
	assert.Equal(t, "Meeting", vevent.PropFilters[summaryIdx].TextMatch.Value)
	assert.Equal(t, "contains", vevent.PropFilters[summaryIdx].TextMatch.MatchType)

	// Check ATTENDEE filter with param filter
	attendeeIdx := propFilters["ATTENDEE"]
	assert.Len(t, vevent.PropFilters[attendeeIdx].ParamFilters, 1)
	assert.Equal(t, "PARTSTAT", vevent.PropFilters[attendeeIdx].ParamFilters[0].Name)
	assert.Equal(t, "ACCEPTED", vevent.PropFilters[attendeeIdx].ParamFilters[0].TextMatch.Value)

	// Check LOCATION filter with is-not-defined
	locationIdx := propFilters["LOCATION"]
	assert.True(t, vevent.PropFilters[locationIdx].IsNotDefined)
}

func TestParseRequest_ExtraNamespaces(t *testing.T) {
	// Test with additional namespaces that shouldn't affect parsing
	xml := `<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:X="http://example.org/ns/">
  <D:prop>
    <D:getetag/>
    <X:custom-prop/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR"/>
  </C:filter>
</C:calendar-query>`

	propsMap, filter, err := ParseRequest(xml)

	// Check no error occurred
	assert.NoError(t, err)

	// Only standard properties should be parsed
	assert.Len(t, propsMap, 1)
	assert.Contains(t, propsMap, "getetag")

	// Filter should be parsed correctly
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)
}
