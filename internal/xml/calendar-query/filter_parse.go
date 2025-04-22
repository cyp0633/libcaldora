package calendarquery

import (
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/server/storage"
)

// ParseFilterElement parses a <filter> element into a Filter structure
func ParseFilterElement(filterElem *etree.Element) (*storage.Filter, error) {
	if filterElem == nil {
		return nil, nil
	}

	// Find comp-filter elements
	compFilters := getElementsIgnoreNS(filterElem, "comp-filter")
	if len(compFilters) == 0 {
		return nil, nil
	}

	// Parse the first comp-filter (should be VCALENDAR)
	return parseCompFilter(compFilters[0]), nil
}

// parseCompFilter recursively parses a comp-filter element
func parseCompFilter(compFilterElem *etree.Element) *storage.Filter {
	filter := &storage.Filter{
		Component: compFilterElem.SelectAttrValue("name", ""),
		Test:      compFilterElem.SelectAttrValue("test", "anyof"),
	}

	// Check for is-not-defined
	if findElementIgnoreNS(compFilterElem, "is-not-defined") != nil {
		filter.IsNotDefined = true
		return filter // If is-not-defined is present, other elements should not be
	}

	// Parse time-range if present
	timeRangeElem := findElementIgnoreNS(compFilterElem, "time-range")
	if timeRangeElem != nil {
		filter.TimeRange = parseTimeRange(timeRangeElem)
	}

	// Parse prop-filters
	propFilterElems := getElementsIgnoreNS(compFilterElem, "prop-filter")
	for _, propFilterElem := range propFilterElems {
		propFilter := parsePropFilter(propFilterElem)
		filter.PropFilters = append(filter.PropFilters, propFilter)
	}

	// Parse nested comp-filters
	nestedCompFilterElems := getElementsIgnoreNS(compFilterElem, "comp-filter")
	for _, nestedElem := range nestedCompFilterElems {
		nestedFilter := parseCompFilter(nestedElem)
		filter.Children = append(filter.Children, *nestedFilter)
	}

	return filter
}

// parsePropFilter parses a prop-filter element
func parsePropFilter(propFilterElem *etree.Element) storage.PropFilter {
	propFilter := storage.PropFilter{
		Name: propFilterElem.SelectAttrValue("name", ""),
		Test: propFilterElem.SelectAttrValue("test", "anyof"),
	}

	// Check for is-not-defined
	if findElementIgnoreNS(propFilterElem, "is-not-defined") != nil {
		propFilter.IsNotDefined = true
		return propFilter // If is-not-defined is present, other elements should not be
	}

	// Parse text-match
	textMatchElem := findElementIgnoreNS(propFilterElem, "text-match")
	if textMatchElem != nil {
		propFilter.TextMatch = parseTextMatch(textMatchElem)
	}

	// Parse param-filters
	paramFilterElems := getElementsIgnoreNS(propFilterElem, "param-filter")
	for _, paramFilterElem := range paramFilterElems {
		paramFilter := parseParamFilter(paramFilterElem)
		propFilter.ParamFilters = append(propFilter.ParamFilters, paramFilter)
	}

	return propFilter
}

// parseParamFilter parses a param-filter element
func parseParamFilter(paramFilterElem *etree.Element) storage.ParamFilter {
	paramFilter := storage.ParamFilter{
		Name: paramFilterElem.SelectAttrValue("name", ""),
	}

	// Check for is-not-defined
	if findElementIgnoreNS(paramFilterElem, "is-not-defined") != nil {
		paramFilter.IsNotDefined = true
		return paramFilter
	}

	// Parse text-match
	textMatchElem := findElementIgnoreNS(paramFilterElem, "text-match")
	if textMatchElem != nil {
		paramFilter.TextMatch = parseTextMatch(textMatchElem)
	}

	return paramFilter
}

// parseTextMatch parses a text-match element
func parseTextMatch(textMatchElem *etree.Element) *storage.TextMatch {
	return &storage.TextMatch{
		Collation: textMatchElem.SelectAttrValue("collation", "i;unicode-casemap"),
		MatchType: textMatchElem.SelectAttrValue("match-type", "contains"),
		Negate:    textMatchElem.SelectAttrValue("negate-condition", "no") == "yes",
		Value:     textMatchElem.Text(),
	}
}

// parseTimeRange parses a time-range element
func parseTimeRange(timeRangeElem *etree.Element) *storage.TimeRange {
	timeRange := &storage.TimeRange{}

	startStr := timeRangeElem.SelectAttrValue("start", "")
	if startStr != "" {
		start, err := time.Parse("20060102T150405Z", startStr)
		if err == nil {
			timeRange.Start = &start
		}
	}

	endStr := timeRangeElem.SelectAttrValue("end", "")
	if endStr != "" {
		end, err := time.Parse("20060102T150405Z", endStr)
		if err == nil {
			timeRange.End = &end
		}
	}

	return timeRange
}

// Helper functions to handle namespaces

// getElementsIgnoreNS returns all child elements with the given local name, ignoring namespace
func getElementsIgnoreNS(parent *etree.Element, localName string) []*etree.Element {
	var elements []*etree.Element
	for _, child := range parent.ChildElements() {
		// Strip namespace prefix if present
		tagName := child.Tag
		if strings.Contains(tagName, ":") {
			tagName = strings.Split(tagName, ":")[1]
		}

		if strings.EqualFold(tagName, localName) {
			elements = append(elements, child)
		}
	}
	return elements
}

// findElementIgnoreNS finds the first child element with the given local name, ignoring namespace
func findElementIgnoreNS(parent *etree.Element, localName string) *etree.Element {
	elements := getElementsIgnoreNS(parent, localName)
	if len(elements) > 0 {
		return elements[0]
	}
	return nil
}
