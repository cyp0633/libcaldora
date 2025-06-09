package recurrence

import (
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// ExtractRecurrenceInfoFromComponent extracts recurrence information from an iCal component
func ExtractRecurrenceInfoFromComponent(comp *ical.Component) RecurrenceInfo {
	info := RecurrenceInfo{}

	// Extract RRULE
	if rruleProp := comp.Props.Get(ical.PropRecurrenceRule); rruleProp != nil && rruleProp.Value != "" {
		info.RRULE = rruleProp.Value
	}

	// Extract RDATE
	if rdateProp := comp.Props.Get(ical.PropRecurrenceDates); rdateProp != nil && rdateProp.Value != "" {
		info.RDATE = parseRecurrenceDates(rdateProp.Value, rdateProp.Params)
	}

	// Extract EXDATE
	if exdateProp := comp.Props.Get(ical.PropExceptionDates); exdateProp != nil && exdateProp.Value != "" {
		info.EXDATE = parseExceptionDates(exdateProp.Value, exdateProp.Params)
	}

	// Extract RECURRENCE-ID (for exception instances)
	if recurrenceIdProp := comp.Props.Get("RECURRENCE-ID"); recurrenceIdProp != nil && recurrenceIdProp.Value != "" {
		if recId, err := parseDateTime(recurrenceIdProp.Value, recurrenceIdProp.Params); err == nil {
			info.RecurrenceID = &recId
		}
	}

	return info
}

// ExtractBasicTimeInfoFromComponent extracts start and end times from an iCal component
func ExtractBasicTimeInfoFromComponent(comp *ical.Component) (start, end time.Time, hasTime bool) {
	// Get start time
	if dtstart, err := comp.Props.DateTime(ical.PropDateTimeStart, nil); err == nil {
		start = dtstart
		hasTime = true

		// Get end time - either from DTEND or DURATION or default
		if dtend, err := comp.Props.DateTime(ical.PropDateTimeEnd, nil); err == nil {
			end = dtend

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
			} else {
				// Invalid duration, fall back to default
				hasTime = false
				return
			}
		} else {
			// Default duration:
			// For all-day events (date values), duration is 1 day.
			// For timed events, it's an instantaneous event (end == start).
			if isAllDayDate(start) {
				end = start.AddDate(0, 0, 1)
			} else {
				end = start
			}
		}
	}

	// For VTODO, also check DUE property
	if comp.Name == ical.CompToDo {
		if due, err := comp.Props.DateTime(ical.PropDue, nil); err == nil {
			if !hasTime {
				start = due
				end = due
				hasTime = true
			} else {
				// If DUE is later than calculated end, use DUE as the end
				if due.After(end) {
					end = due
				}
			}
		}
	}

	return start, end, hasTime
}

// parseRecurrenceDates parses RDATE property value into time.Time slice
func parseRecurrenceDates(value string, params map[string][]string) []time.Time {
	if value == "" {
		return nil
	}

	var rdates []time.Time
	rdateStrings := strings.Split(value, ",")

	// Check if this is a date-only RDATE (VALUE=DATE parameter)
	isDateOnly := false
	if params != nil {
		if valueParam := params["VALUE"]; len(valueParam) > 0 && strings.ToUpper(valueParam[0]) == "DATE" {
			isDateOnly = true
		}
	}

	for _, rdateStr := range rdateStrings {
		rdateStr = strings.TrimSpace(rdateStr)
		if rdateStr == "" {
			continue
		}

		var rdate time.Time
		var err error

		if isDateOnly {
			// Parse as date-only and store with 00:00:00 time in UTC
			rdate, err = time.Parse("20060102", rdateStr)
			if err == nil {
				// Store as midnight UTC for date-only RDATEs
				rdate = time.Date(rdate.Year(), rdate.Month(), rdate.Day(), 0, 0, 0, 0, time.UTC)
			}
		} else {
			// Parse the iCalendar date-time format
			rdate, err = time.Parse("20060102T150405Z", rdateStr)
			if err != nil {
				// Try parsing as date-only format as fallback
				rdate, err = time.Parse("20060102", rdateStr)
				if err == nil {
					// If we had to fall back to date parsing, treat as date-only
					rdate = time.Date(rdate.Year(), rdate.Month(), rdate.Day(), 0, 0, 0, 0, time.UTC)
				}
			}
		}

		if err == nil {
			rdates = append(rdates, rdate)
		}
	}

	return rdates
}

// parseExceptionDates parses EXDATE property value into time.Time slice
func parseExceptionDates(value string, params map[string][]string) []time.Time {
	if value == "" {
		return nil
	}

	var exdates []time.Time
	exdateStrings := strings.Split(value, ",")

	// Check if this is a date-only EXDATE (VALUE=DATE parameter)
	isDateOnly := false
	if params != nil {
		if valueParam := params["VALUE"]; len(valueParam) > 0 && strings.ToUpper(valueParam[0]) == "DATE" {
			isDateOnly = true
		}
	}

	for _, exdateStr := range exdateStrings {
		exdateStr = strings.TrimSpace(exdateStr)
		if exdateStr == "" {
			continue
		}

		var exdate time.Time
		var err error

		if isDateOnly {
			// Parse as date-only and store with 00:00:00 time in UTC
			exdate, err = time.Parse("20060102", exdateStr)
			if err == nil {
				// Store as midnight UTC for date-only EXDATEs
				exdate = time.Date(exdate.Year(), exdate.Month(), exdate.Day(), 0, 0, 0, 0, time.UTC)
			}
		} else {
			// Parse the iCalendar date-time format
			exdate, err = time.Parse("20060102T150405Z", exdateStr)
			if err != nil {
				// Try parsing as date-only format as fallback
				exdate, err = time.Parse("20060102", exdateStr)
				if err == nil {
					// If we had to fall back to date parsing, treat as date-only
					exdate = time.Date(exdate.Year(), exdate.Month(), exdate.Day(), 0, 0, 0, 0, time.UTC)
				}
			}
		}

		if err == nil {
			exdates = append(exdates, exdate)
		}
	}

	return exdates
}

// parseDateTime parses a date-time string from iCalendar properties
func parseDateTime(value string, params map[string][]string) (time.Time, error) {
	// Check if this is a date-only value (VALUE=DATE parameter)
	isDateOnly := false
	if params != nil {
		if valueParam := params["VALUE"]; len(valueParam) > 0 && strings.ToUpper(valueParam[0]) == "DATE" {
			isDateOnly = true
		}
	}

	if isDateOnly {
		// Parse as date-only and store with 00:00:00 time in UTC
		t, err := time.Parse("20060102", value)
		if err == nil {
			// Store as midnight UTC for date-only values
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
		return t, err
	} else {
		// Try parsing as iCalendar date-time format first
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			// Try parsing as date-only format as fallback
			t, err = time.Parse("20060102", value)
			if err == nil {
				// If we had to fall back to date parsing, treat as date-only
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			}
		}
		return t, err
	}
}

// isAllDayDate checks if a time represents an all-day date (time part is midnight)
func isAllDayDate(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0
}

// SafeTimeDeref safely dereferences a time pointer, returning zero time if nil
func SafeTimeDeref(t *time.Time, defaultTime time.Time) time.Time {
	if t == nil {
		return defaultTime
	}
	return *t
}
