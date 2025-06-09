package recurrence

import (
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

// Engine provides unified recurrence expansion and validation logic
type Engine struct {
	// Future: can add caching, configuration, etc.
}

// NewEngine creates a new recurrence engine instance
func NewEngine() *Engine {
	return &Engine{}
}

// HasOccurrenceInRange checks if a recurring event has any occurrence in the time range
// This is a performance-optimized method that doesn't do full expansion
func (e *Engine) HasOccurrenceInRange(
	masterStart, masterEnd time.Time,
	recurrence RecurrenceInfo,
	rangeStart, rangeEnd time.Time,
) (bool, error) {
	// Fast path: check master event first (if no RRULE, this is the only occurrence)
	// Use proper time range overlap logic: start <= rangeEnd AND end >= rangeStart
	if !masterStart.After(rangeEnd) && !masterEnd.Before(rangeStart) {
		// Check if this occurrence is not excluded by EXDATE
		if !e.isExcluded(masterStart, recurrence.EXDATE) {
			return true, nil
		}
	}

	// Check RRULE occurrences if present
	if recurrence.RRULE != "" {
		hasRRuleOccurrence, err := e.hasRRuleOccurrenceInRange(
			masterStart, recurrence.RRULE, recurrence.EXDATE, rangeStart, rangeEnd)
		if err != nil {
			return false, fmt.Errorf("failed to check RRULE occurrences: %w", err)
		}
		if hasRRuleOccurrence {
			return true, nil
		}
	}

	// Check RDATE occurrences
	duration := masterEnd.Sub(masterStart)
	for _, rdate := range recurrence.RDATE {
		rdateEnd := rdate.Add(duration)
		// Use proper time range overlap logic: start <= rangeEnd AND end >= rangeStart
		if !rdate.After(rangeEnd) && !rdateEnd.Before(rangeStart) && !e.isExcluded(rdate, recurrence.EXDATE) {
			return true, nil
		}
	}

	return false, nil
}

// hasRRuleOccurrenceInRange checks if an RRULE has any occurrence in range (optimized)
func (e *Engine) hasRRuleOccurrenceInRange(
	masterStart time.Time, rruleStr string, exdates []time.Time, rangeStart, rangeEnd time.Time) (bool, error) {

	// For performance, we limit the expansion to check only the first few occurrences
	// This is a reasonable trade-off for the "has occurrence" check
	limitedRangeEnd := rangeEnd
	if rangeEnd.Sub(rangeStart) > 90*24*time.Hour { // If range > 90 days, limit check
		limitedRangeEnd = rangeStart.Add(90 * 24 * time.Hour)
	}

	occurrences, err := e.expandRRule(masterStart, rruleStr, rangeStart, limitedRangeEnd)
	if err != nil {
		return false, err
	}

	// Check if any occurrence is not excluded
	for _, occurrence := range occurrences {
		if !e.isExcluded(occurrence, exdates) {
			return true, nil
		}
	}

	// If we limited the range and found nothing, try the full range with a reasonable limit
	if limitedRangeEnd.Before(rangeEnd) && len(occurrences) > 0 {
		fullOccurrences, err := e.expandRRule(masterStart, rruleStr, rangeStart, rangeEnd)
		if err != nil {
			return false, err
		}

		// Check up to 100 occurrences for performance
		limit := len(fullOccurrences)
		if limit > 100 {
			limit = 100
		}

		for i := 0; i < limit; i++ {
			if !e.isExcluded(fullOccurrences[i], exdates) {
				return true, nil
			}
		}
	}

	return false, nil
}

// expandRRule expands an RRULE within the given time range
func (e *Engine) expandRRule(masterStart time.Time, rruleStr string, rangeStart, rangeEnd time.Time) ([]time.Time, error) {
	// Build the full RRULE string for parsing
	dtstart := masterStart.UTC().Format("20060102T150405Z")
	fullRRule := fmt.Sprintf("DTSTART:%s\nRRULE:%s", dtstart, rruleStr)

	// Parse the RRULE
	ruleSet, err := rrule.StrToRRuleSet(fullRRule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RRULE '%s': %w", rruleStr, err)
	}

	// Get occurrences in the time range
	// Note: rrule-go's Between method is inclusive of start, exclusive of end
	occurrences := ruleSet.Between(rangeStart, rangeEnd, true)

	return occurrences, nil
}

// isExcluded checks if a given time is in the EXDATE list
func (e *Engine) isExcluded(t time.Time, exdates []time.Time) bool {
	for _, exdate := range exdates {
		// Handle both exact timestamp matches and date-only matches
		if t.Equal(exdate) {
			return true
		}

		// For date-only exceptions (stored as midnight UTC), check if the occurrence
		// falls on the same date when normalized to midnight UTC
		if exdate.Hour() == 0 && exdate.Minute() == 0 && exdate.Second() == 0 && exdate.Location() == time.UTC {
			occurrenceAtMidnight := time.Date(
				t.Year(), t.Month(), t.Day(),
				0, 0, 0, 0, time.UTC,
			)
			if occurrenceAtMidnight.Equal(exdate) {
				return true
			}
		}
	}
	return false
}
