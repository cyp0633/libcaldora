package recurrence

import (
	"time"
)

// RecurrenceInfo contains all recurrence-related information for an event
type RecurrenceInfo struct {
	RRULE        string      // The RRULE string (without "RRULE:" prefix)
	RDATE        []time.Time // Additional recurrence dates
	EXDATE       []time.Time // Exception dates (excluded occurrences)
	RecurrenceID *time.Time  // For exception instances - which occurrence this overrides
}

// TimeOccurrence represents a single occurrence of an event in time
type TimeOccurrence struct {
	Start        time.Time  // Start time of this occurrence
	End          time.Time  // End time of this occurrence
	IsException  bool       // True if this is an exception/override instance
	RecurrenceID *time.Time // If this is an exception, the original occurrence time
}

// ExpansionOptions controls how recurrence expansion behaves
type ExpansionOptions struct {
	MaxOccurrences    int           // Maximum number of occurrences to expand (0 = unlimited)
	MaxTimeSpan       time.Duration // Maximum time span to expand (0 = unlimited)
	IncludeExceptions bool          // Whether to include exception instances in expansion
}

// DefaultExpansionOptions provides sensible defaults for expansion
var DefaultExpansionOptions = ExpansionOptions{
	MaxOccurrences:    1000,                     // Reasonable limit to prevent infinite expansion
	MaxTimeSpan:       365 * 24 * time.Hour * 2, // 2 years
	IncludeExceptions: true,
}
