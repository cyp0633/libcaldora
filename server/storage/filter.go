package storage

import "time"

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
