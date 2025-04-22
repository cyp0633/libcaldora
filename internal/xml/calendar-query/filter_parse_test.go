package calendarquery

import (
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/stretchr/testify/assert"
)

// createElementFromXML is a test helper that creates an etree Element from XML string
func createElementFromXML(t *testing.T, xmlStr string) *etree.Element {
	doc := etree.NewDocument()
	err := doc.ReadFromString(xmlStr)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}
	return doc.Root()
}

func TestParseFilterElement_Nil(t *testing.T) {
	filter, err := ParseFilterElement(nil)
	assert.Nil(t, filter)
	assert.Nil(t, err)
}

func TestParseFilterElement_Empty(t *testing.T) {
	filterElem := createElementFromXML(t, `<C:filter xmlns:C="urn:ietf:params:xml:ns:caldav"></C:filter>`)
	filter, err := ParseFilterElement(filterElem)
	assert.Nil(t, filter)
	assert.Nil(t, err)
}

func TestParseFilterElement_Basic(t *testing.T) {
	filterXML := `
    <C:filter xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:comp-filter name="VCALENDAR">
            <C:comp-filter name="VEVENT"/>
        </C:comp-filter>
    </C:filter>
    `
	filterElem := createElementFromXML(t, filterXML)
	filter, err := ParseFilterElement(filterElem)

	assert.Nil(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)
	assert.Equal(t, "anyof", filter.Test)
	assert.Len(t, filter.Children, 1)
	assert.Equal(t, "VEVENT", filter.Children[0].Component)
}

func TestParseFilterElement_Complete(t *testing.T) {
	filterXML := `
    <C:filter xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:comp-filter name="VCALENDAR" test="allof">
            <C:comp-filter name="VEVENT">
                <C:time-range start="20240101T000000Z" end="20240131T235959Z"/>
                <C:prop-filter name="SUMMARY">
                    <C:text-match collation="i;unicode-casemap" match-type="contains">Meeting</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="LOCATION">
                    <C:is-not-defined/>
                </C:prop-filter>
            </C:comp-filter>
        </C:comp-filter>
    </C:filter>
    `
	filterElem := createElementFromXML(t, filterXML)
	filter, err := ParseFilterElement(filterElem)

	assert.Nil(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)
	assert.Equal(t, "allof", filter.Test)
	assert.Len(t, filter.Children, 1)

	eventFilter := filter.Children[0]
	assert.Equal(t, "VEVENT", eventFilter.Component)
	assert.NotNil(t, eventFilter.TimeRange)

	// Check time range
	expectedStart, _ := time.Parse("20060102T150405Z", "20240101T000000Z")
	expectedEnd, _ := time.Parse("20060102T150405Z", "20240131T235959Z")
	assert.Equal(t, expectedStart, *eventFilter.TimeRange.Start)
	assert.Equal(t, expectedEnd, *eventFilter.TimeRange.End)

	// Check property filters
	assert.Len(t, eventFilter.PropFilters, 2)

	// Check summary filter
	assert.Equal(t, "SUMMARY", eventFilter.PropFilters[0].Name)
	assert.NotNil(t, eventFilter.PropFilters[0].TextMatch)
	assert.Equal(t, "Meeting", eventFilter.PropFilters[0].TextMatch.Value)
	assert.Equal(t, "i;unicode-casemap", eventFilter.PropFilters[0].TextMatch.Collation)
	assert.Equal(t, "contains", eventFilter.PropFilters[0].TextMatch.MatchType)
	assert.False(t, eventFilter.PropFilters[0].TextMatch.Negate)

	// Check location filter
	assert.Equal(t, "LOCATION", eventFilter.PropFilters[1].Name)
	assert.True(t, eventFilter.PropFilters[1].IsNotDefined)
}

func TestParseCompFilter_IsNotDefined(t *testing.T) {
	compFilterXML := `
    <C:comp-filter name="VEVENT" xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:is-not-defined/>
    </C:comp-filter>
    `
	compFilterElem := createElementFromXML(t, compFilterXML)
	filter := parseCompFilter(compFilterElem)

	assert.NotNil(t, filter)
	assert.Equal(t, "VEVENT", filter.Component)
	assert.True(t, filter.IsNotDefined)
	assert.Nil(t, filter.TimeRange)
	assert.Empty(t, filter.PropFilters)
	assert.Empty(t, filter.Children)
}

func TestParsePropFilter_WithParamFilter(t *testing.T) {
	propFilterXML := `
    <C:prop-filter name="ATTENDEE" test="allof" xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:param-filter name="PARTSTAT">
            <C:text-match>ACCEPTED</C:text-match>
        </C:param-filter>
        <C:param-filter name="ROLE">
            <C:is-not-defined/>
        </C:param-filter>
    </C:prop-filter>
    `
	propFilterElem := createElementFromXML(t, propFilterXML)
	propFilter := parsePropFilter(propFilterElem)

	assert.Equal(t, "ATTENDEE", propFilter.Name)
	assert.Equal(t, "allof", propFilter.Test)
	assert.Len(t, propFilter.ParamFilters, 2)

	// First param filter with text match
	assert.Equal(t, "PARTSTAT", propFilter.ParamFilters[0].Name)
	assert.NotNil(t, propFilter.ParamFilters[0].TextMatch)
	assert.Equal(t, "ACCEPTED", propFilter.ParamFilters[0].TextMatch.Value)

	// Second param filter with is-not-defined
	assert.Equal(t, "ROLE", propFilter.ParamFilters[1].Name)
	assert.True(t, propFilter.ParamFilters[1].IsNotDefined)
}

func TestParseTextMatch_AllOptions(t *testing.T) {
	textMatchXML := `
    <C:text-match xmlns:C="urn:ietf:params:xml:ns:caldav" 
        collation="i;octet" 
        match-type="equals" 
        negate-condition="yes">Test Value</C:text-match>
    `
	textMatchElem := createElementFromXML(t, textMatchXML)
	textMatch := parseTextMatch(textMatchElem)

	assert.NotNil(t, textMatch)
	assert.Equal(t, "i;octet", textMatch.Collation)
	assert.Equal(t, "equals", textMatch.MatchType)
	assert.True(t, textMatch.Negate)
	assert.Equal(t, "Test Value", textMatch.Value)
}

func TestParseTimeRange(t *testing.T) {
	timeRangeXML := `
    <C:time-range xmlns:C="urn:ietf:params:xml:ns:caldav" 
        start="20240101T120000Z" 
        end="20240102T120000Z"/>
    `
	timeRangeElem := createElementFromXML(t, timeRangeXML)
	timeRange := parseTimeRange(timeRangeElem)

	assert.NotNil(t, timeRange)

	expectedStart, _ := time.Parse("20060102T150405Z", "20240101T120000Z")
	expectedEnd, _ := time.Parse("20060102T150405Z", "20240102T120000Z")

	assert.NotNil(t, timeRange.Start)
	assert.Equal(t, expectedStart, *timeRange.Start)

	assert.NotNil(t, timeRange.End)
	assert.Equal(t, expectedEnd, *timeRange.End)
}

func TestParseTimeRange_InvalidFormat(t *testing.T) {
	timeRangeXML := `
    <C:time-range xmlns:C="urn:ietf:params:xml:ns:caldav" 
        start="invalid-date" 
        end="also-invalid"/>
    `
	timeRangeElem := createElementFromXML(t, timeRangeXML)
	timeRange := parseTimeRange(timeRangeElem)

	assert.NotNil(t, timeRange)
	assert.Nil(t, timeRange.Start)
	assert.Nil(t, timeRange.End)
}

func TestGetElementsIgnoreNS(t *testing.T) {
	xml := `
    <root xmlns:A="ns-a" xmlns:B="ns-b">
        <A:element>First</A:element>
        <B:element>Second</B:element>
        <element>Third</element>
    </root>
    `
	root := createElementFromXML(t, xml)
	elements := getElementsIgnoreNS(root, "element")

	assert.Len(t, elements, 3)
	assert.Equal(t, "First", elements[0].Text())
	assert.Equal(t, "Second", elements[1].Text())
	assert.Equal(t, "Third", elements[2].Text())
}

func TestFindElementIgnoreNS(t *testing.T) {
	xml := `
    <root xmlns:A="ns-a" xmlns:B="ns-b">
        <A:first>First Element</A:first>
        <B:second>Second Element</B:second>
    </root>
    `
	root := createElementFromXML(t, xml)

	firstElem := findElementIgnoreNS(root, "first")
	assert.NotNil(t, firstElem)
	assert.Equal(t, "First Element", firstElem.Text())

	secondElem := findElementIgnoreNS(root, "second")
	assert.NotNil(t, secondElem)
	assert.Equal(t, "Second Element", secondElem.Text())

	// Test non-existent element
	missingElem := findElementIgnoreNS(root, "missing")
	assert.Nil(t, missingElem)
}

func TestComplexFilter(t *testing.T) {
	// This test uses the example from the attachment
	filterXML := `
    <C:filter xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:comp-filter name="VCALENDAR">
            <C:comp-filter name="VEVENT">
                <C:time-range start="20240101T000000Z" end="20240131T235959Z"/>
                <C:prop-filter name="SUMMARY">
                    <C:text-match collation="i;unicode-casemap" negate-condition="no">Meeting</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="CATEGORIES">
                    <C:text-match collation="i;unicode-casemap">Work</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="STATUS">
                    <C:text-match>CONFIRMED</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="ATTENDEE">
                    <C:text-match>mailto:user@example.com</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="ORGANIZER">
                    <C:text-match>mailto:organizer@example.com</C:text-match>
                </C:prop-filter>
                <C:prop-filter name="RECURRENCE-ID">
                    <C:time-range start="20240115T000000Z" end="20240115T235959Z"/>
                </C:prop-filter>
                <C:prop-filter name="LOCATION">
                    <C:is-not-defined/>
                </C:prop-filter>
            </C:comp-filter>
        </C:comp-filter>
    </C:filter>
    `
	filterElem := createElementFromXML(t, filterXML)
	filter, err := ParseFilterElement(filterElem)

	assert.Nil(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, "VCALENDAR", filter.Component)

	// Check VEVENT filter
	assert.Len(t, filter.Children, 1)
	vevent := filter.Children[0]
	assert.Equal(t, "VEVENT", vevent.Component)

	// Check time range
	assert.NotNil(t, vevent.TimeRange)
	expectedStart, _ := time.Parse("20060102T150405Z", "20240101T000000Z")
	expectedEnd, _ := time.Parse("20060102T150405Z", "20240131T235959Z")
	assert.Equal(t, expectedStart, *vevent.TimeRange.Start)
	assert.Equal(t, expectedEnd, *vevent.TimeRange.End)

	// Check property filters
	assert.Len(t, vevent.PropFilters, 7)

	// Verify each property filter
	propFilters := make(map[string]storage.PropFilter)
	for _, pf := range vevent.PropFilters {
		propFilters[pf.Name] = pf
	}

	// Check SUMMARY filter
	assert.Contains(t, propFilters, "SUMMARY")
	assert.Equal(t, "Meeting", propFilters["SUMMARY"].TextMatch.Value)
	assert.Equal(t, "i;unicode-casemap", propFilters["SUMMARY"].TextMatch.Collation)
	assert.False(t, propFilters["SUMMARY"].TextMatch.Negate)

	// Check CATEGORIES filter
	assert.Contains(t, propFilters, "CATEGORIES")
	assert.Equal(t, "Work", propFilters["CATEGORIES"].TextMatch.Value)

	// Check STATUS filter
	assert.Contains(t, propFilters, "STATUS")
	assert.Equal(t, "CONFIRMED", propFilters["STATUS"].TextMatch.Value)

	// Check ATTENDEE filter
	assert.Contains(t, propFilters, "ATTENDEE")
	assert.Equal(t, "mailto:user@example.com", propFilters["ATTENDEE"].TextMatch.Value)

	// Check ORGANIZER filter
	assert.Contains(t, propFilters, "ORGANIZER")
	assert.Equal(t, "mailto:organizer@example.com", propFilters["ORGANIZER"].TextMatch.Value)

	// Check LOCATION filter with is-not-defined
	assert.Contains(t, propFilters, "LOCATION")
	assert.True(t, propFilters["LOCATION"].IsNotDefined)

	// Check RECURRENCE-ID filter
	assert.Contains(t, propFilters, "RECURRENCE-ID")
	// This one doesn't have a TextMatch but should have a TimeRange
}
