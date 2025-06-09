package storage

import (
	"strings"
	"time"

	"github.com/cyp0633/libcaldora/server/recurrence"
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
// This now properly handles recurring events using the unified recurrence engine
func validateTimeRange(comp *ical.Component, timeRange *TimeRange) bool {
	// If no time range constraints, always match
	if timeRange.Start == nil && timeRange.End == nil {
		return true
	}

	// Extract basic time info from the component
	masterStart, masterEnd, hasBasicTime := recurrence.ExtractBasicTimeInfoFromComponent(comp)
	if !hasBasicTime {
		return false
	}

	// Extract recurrence information
	recurrenceInfo := recurrence.ExtractRecurrenceInfoFromComponent(comp)

	// Determine the query time range
	rangeStart := recurrence.SafeTimeDeref(timeRange.Start, time.Time{})
	rangeEnd := recurrence.SafeTimeDeref(timeRange.End, time.Now().AddDate(10, 0, 0)) // reasonable future limit

	// If End < Start, ignore End (treat as only‐start)
	if timeRange.Start != nil && timeRange.End != nil && timeRange.End.Before(*timeRange.Start) {
		rangeEnd = time.Now().AddDate(10, 0, 0) // Default to far future
	}

	// Use the centralized recurrence engine for RFC 4791 compliant validation
	engine := recurrence.NewEngine()

	// For performance, use the fast check that doesn't do full expansion
	hasOccurrence, err := engine.HasOccurrenceInRange(
		masterStart, masterEnd,
		recurrenceInfo,
		rangeStart, rangeEnd,
	)

	if err != nil {
		// Fallback to basic validation on error to maintain compatibility
		return validateBasicTimeRange(masterStart, masterEnd, timeRange)
	}

	return hasOccurrence
}

// validateBasicTimeRange provides fallback validation using only master event times
// This is used when recurrence expansion fails, to maintain backward compatibility
func validateBasicTimeRange(start, end time.Time, timeRange *TimeRange) bool {
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
