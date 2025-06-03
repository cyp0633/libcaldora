package storage

import (
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// TextMatch describes a <text‑match> constraint.
type TextMatch struct {
	Collation string // "i;unicode-casemap", etc.
	MatchType string // "equals", "contains", …
	Negate    bool   // true if negate-condition="yes"
	Value     string // text to match
}

// ParamFilter describes a <param-filter> inside a prop-filter.
type ParamFilter struct {
	Name         string     // e.g. "LANGUAGE", "PARTSTAT"
	IsNotDefined bool       // <is-not-defined/>
	TextMatch    *TextMatch // optional
}

// PropFilter describes a <prop‑filter> inside a comp-filter.
type PropFilter struct {
	Name         string        // e.g. "SUMMARY", "UID"
	IsNotDefined bool          // <is-not-defined/>
	TextMatch    *TextMatch    // optional
	ParamFilters []ParamFilter // zero or more <param-filter>
	Test         string        // "anyof" (default) or "allof"
}

// TimeRange describes a <time‑range> in a comp-filter.
type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

// Filter is now your one‑and‑only node type.
// It can represent a comp-filter, time-range, or prop-filters
type Filter struct {
	Component    string       // Name of component (e.g. "VCALENDAR", "VEVENT")
	IsNotDefined bool         // <is-not-defined/>
	TimeRange    *TimeRange   // optional <time-range>
	PropFilters  []PropFilter // zero or more <prop-filter>
	Children     []Filter     // nested <comp-filter>
	Test         string       // "anyof" (default) or "allof"
}

// Validate checks if a calendar object matches the given filter.
func (f *Filter) Validate(calObj *CalendarObject) bool {
	// Handle nil object
	if calObj == nil || len(calObj.Component) == 0 || calObj.Component[0] == nil {
		return f.IsNotDefined
	}

	// Get component name from the first component
	componentName := calObj.Component[0].Name

	// Handle is-not-defined case
	if f.IsNotDefined {
		return componentName != f.Component
	}

	// Check component name match if specified
	if f.Component != "" && componentName != f.Component {
		return false
	}

	// Check time range constraints
	if f.TimeRange != nil && !validateTimeRange(calObj.Component[0], f.TimeRange) {
		return false
	}

	// Set default test value if not specified
	test := f.Test
	if test == "" {
		test = "anyof"
	}

	// Validate property filters
	if len(f.PropFilters) > 0 {
		propResult := validatePropFilters(calObj.Component[0], f.PropFilters, test)
		if !propResult {
			return false
		}
	}

	// Validate nested component filters
	if len(f.Children) > 0 {
		childResult := validateChildren(calObj.Component[0], f.Children, test)
		if !childResult {
			return false
		}
	}

	return true
}

// validateTimeRange checks if a component falls within the specified time range
func validateTimeRange(comp *ical.Component, timeRange *TimeRange) bool {
	// If no time range constraints, always match
	if timeRange.Start == nil && timeRange.End == nil {
		return true
	}

	var start, end time.Time
	var hasStart, hasEnd bool

	// Get start time
	if dtstart, err := comp.Props.DateTime(ical.PropDateTimeStart, nil); err == nil {
		start = dtstart
		hasStart = true

		// Get end time - either from DTEND or DURATION or default
		if dtend, err := comp.Props.DateTime(ical.PropDateTimeEnd, nil); err == nil {
			end = dtend
			hasEnd = true

			// Special handling for all-day events: if start and end are the same DATE,
			// treat it as a 24-hour event ending at the start of the next day.
			startYear, startMonth, startDay := start.Date()
			endYear, endMonth, endDay := end.Date()
			if isAllDayDate(start) && startYear == endYear && startMonth == endMonth && startDay == endDay {
				end = start.AddDate(0, 0, 1)
			}
		} else if durationProp := comp.Props.Get(ical.PropDuration); durationProp != nil {
			if duration, err := durationProp.Duration(); err == nil {
				end = start.Add(duration)
				hasEnd = true
			} else {
				// Invalid duration, treat as not matching
				return false
			}
		} else {
			// Default duration:
			// For all-day events (date values), duration is 1 day.
			// For timed events, it's an instantaneous event (end == start).
			if isAllDayDate(start) {
				end = start.AddDate(0, 0, 1)
			} else {
				// RFC 5545 Section 3.6.1: "If the duration is not present,
				// the event is defined to have a zero duration."
				// However, for filtering, an instantaneous event should still match
				// if the time range includes that instant. Let's treat end = start.
				end = start
			}
			hasEnd = true
		}
	}

	// For VTODO, also check DUE property. The event time is the interval [START, max(END, DUE)]
	// If START is not present, the interval is just [DUE, DUE]
	if comp.Name == ical.CompToDo {
		if due, err := comp.Props.DateTime(ical.PropDue, nil); err == nil {
			if !hasStart {
				start = due
				end = due // Treat TODO with only DUE as instantaneous at DUE time
				hasStart = true
				hasEnd = true
			} else {
				// If END wasn't determined from DTEND/DURATION, use DUE.
				if !hasEnd {
					end = due
					hasEnd = true
				} else if due.After(end) {
					// If DUE is later than DTEND or calculated end from DURATION, use DUE as the end.
					end = due
				}
				// If DUE is before or equal to the calculated end, the interval [start, end) already covers it.
			}
		}
	}

	// If we couldn't determine the time interval, no match
	if !hasStart || !hasEnd {
		return false
	}

	// Prepare range bounds
	rangeStart := timeRange.Start
	rangeEnd := timeRange.End

	// If End < Start, ignore End (treat as only‐start)
	if rangeStart != nil && rangeEnd != nil && rangeEnd.Before(*rangeStart) {
		rangeEnd = nil
	}

	// Overlap if start ≤ rangeEnd AND end ≥ rangeStart
	// (nil bound means "–∞" or "+∞")
	cond1 := rangeEnd == nil || !start.After(*rangeEnd)    // start ≤ rangeEnd
	cond2 := rangeStart == nil || !end.Before(*rangeStart) // end ≥ rangeStart

	return cond1 && cond2
}

// isAllDayDate checks if a time represents an all-day date (time part is midnight)
func isAllDayDate(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0
}

// validatePropFilters checks if component properties match all filters
func validatePropFilters(comp *ical.Component, propFilters []PropFilter, test string) bool {
	matches := 0

	for _, pf := range propFilters {
		if validatePropFilter(comp, &pf) {
			matches++
			if test == "anyof" {
				return true // Short-circuit for anyof
			}
		} else if test == "allof" {
			return false // Short-circuit for allof
		}
	}

	return (test == "anyof" && matches > 0) || (test == "allof" && matches == len(propFilters))
}

// validatePropFilter checks if a single property filter matches
func validatePropFilter(comp *ical.Component, pf *PropFilter) bool {
	props := comp.Props.Values(pf.Name)

	// Handle is-not-defined case
	if pf.IsNotDefined {
		return len(props) == 0
	}

	// Property must exist
	if len(props) == 0 {
		return false
	}

	// If no further constraints, property existence is enough
	if pf.TextMatch == nil && len(pf.ParamFilters) == 0 {
		return true
	}

	// Set default test value
	test := pf.Test
	if test == "" {
		test = "anyof"
	}

	// Check each property instance
	for _, prop := range props {
		matchesText := true
		matchesParams := true

		// Check text match if specified
		if pf.TextMatch != nil {
			matchesText = validateTextMatch(prop.Value, pf.TextMatch)
		}

		// Check param filters if specified
		if len(pf.ParamFilters) > 0 {
			matchesParams = validateParamFilters(&prop, pf.ParamFilters, test)
		}

		// For anyof, any matching property is sufficient
		if matchesText && matchesParams && test == "anyof" {
			return true
		}
	}

	// For allof with multiple properties, this is more complex
	// The current implementation assumes that for "allof", we need at least
	// one property instance to match all constraints
	return false
}

// validateParamFilters checks if property parameters match the filters
func validateParamFilters(prop *ical.Prop, paramFilters []ParamFilter, test string) bool {
	matches := 0

	for _, pf := range paramFilters {
		if validateParamFilter(prop, &pf) {
			matches++
			if test == "anyof" {
				return true // Short-circuit for anyof
			}
		} else if test == "allof" {
			return false // Short-circuit for allof
		}
	}

	return (test == "anyof" && matches > 0) || (test == "allof" && matches == len(paramFilters))
}

// validateParamFilter checks if a single parameter filter matches
func validateParamFilter(prop *ical.Prop, pf *ParamFilter) bool {
	paramValues := prop.Params.Values(pf.Name)

	// Handle is-not-defined case
	if pf.IsNotDefined {
		return len(paramValues) == 0
	}

	// Parameter must exist
	if len(paramValues) == 0 {
		return false
	}

	// If no text match, parameter existence is enough
	if pf.TextMatch == nil {
		return true
	}

	// Check if any parameter value matches the text match
	for _, value := range paramValues {
		if validateTextMatch(value, pf.TextMatch) {
			return true
		}
	}

	return false
}

// validateTextMatch checks if text value matches the text match constraints
func validateTextMatch(value string, tm *TextMatch) bool {
	// Default match type is "contains"
	matchType := tm.MatchType
	if matchType == "" {
		matchType = "contains"
	}

	var matches bool
	caseInsensitive := tm.Collation == "i;unicode-casemap"

	if caseInsensitive {
		value = strings.ToLower(value)
		compareValue := strings.ToLower(tm.Value)

		switch matchType {
		case "equals":
			matches = value == compareValue
		case "contains":
			matches = strings.Contains(value, compareValue)
		case "starts-with":
			matches = strings.HasPrefix(value, compareValue)
		case "ends-with":
			matches = strings.HasSuffix(value, compareValue)
		default:
			matches = strings.Contains(value, compareValue)
		}
	} else {
		switch matchType {
		case "equals":
			matches = value == tm.Value
		case "contains":
			matches = strings.Contains(value, tm.Value)
		case "starts-with":
			matches = strings.HasPrefix(value, tm.Value)
		case "ends-with":
			matches = strings.HasSuffix(value, tm.Value)
		default:
			matches = strings.Contains(value, tm.Value)
		}
	}

	// Handle negation
	if tm.Negate {
		return !matches
	}
	return matches
}

// validateChildren checks if component children match the filters
func validateChildren(comp *ical.Component, children []Filter, test string) bool {
	matches := 0

	for _, childFilter := range children {
		childMatched := false

		// Get relevant child components
		var childComps []*ical.Component
		if childFilter.Component != "" {
			for _, child := range comp.Children {
				if child.Name == childFilter.Component {
					childComps = append(childComps, child)
				}
			}
		} else {
			childComps = comp.Children
		}

		// Check if any child matches the filter
		for _, childComp := range childComps {
			tempObj := &CalendarObject{Component: []*ical.Component{childComp}}
			if childFilter.Validate(tempObj) {
				childMatched = true
				break
			}
		}

		if childMatched {
			matches++
			if test == "anyof" {
				return true
			}
		} else if test == "allof" {
			return false
		}
	}

	return (test == "anyof" && matches > 0) || (test == "allof" && matches == len(children))
}
