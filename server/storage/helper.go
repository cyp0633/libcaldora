package storage

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// ICalCompToICS converts a ical.Component (event or other calendar component) to an ICS string.
func ICalCompToICS(component ical.Component, removeCalendarWrapper bool) (string, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Caldora//Go Calendar//EN")

	// Ensure DTSTAMP is present
	if component.Props.Get(ical.PropDateTimeStamp) == nil {
		component.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	}

	cal.Children = append(cal.Children, &component) // Adding the component directly

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return "", fmt.Errorf("failed to encode calendar: %w", err)
	}

	icsString := buf.String()

	if removeCalendarWrapper {
		// Determine line ending type used in the ICS file
		lineEnding := "\n"
		if strings.Contains(icsString, "\r\n") {
			lineEnding = "\r\n"
		}

		// Split by line endings to process line by line
		lines := strings.Split(icsString, lineEnding)

		startIdx := -1
		endIdx := -1

		// Find the VEVENT section
		for i, line := range lines {
			if line == "BEGIN:VEVENT" {
				startIdx = i
			} else if line == "END:VEVENT" {
				endIdx = i
				break // We only want the first EVENT
			}
		}

		// If we found the VEVENT section
		if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
			// Extract just the VEVENT lines (including BEGIN:VEVENT and END:VEVENT)
			eventLines := lines[startIdx : endIdx+1]
			return strings.Join(eventLines, lineEnding), nil
		}
	}

	return icsString, nil
}

// ICSToICalComp parses an ICS string and returns an ical.Component (event or other calendar component).
// It automatically adds the VCALENDAR wrapper if not present in the input.
func ICSToICalComp(ics string) (*ical.Component, error) {
	// Add the VCALENDAR wrapper
	if !strings.HasPrefix(strings.TrimSpace(ics), "BEGIN:VCALENDAR") {
		ics = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Caldora//Go Calendar//EN\r\n" + ics + "\r\nEND:VCALENDAR"
	}

	r := strings.NewReader(ics)
	dec := ical.NewDecoder(r)

	// Decode the calendar
	cal, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode calendar: %w", err)
	}

	// Check if there are multiple components in the calendar
	if len(cal.Children) > 1 {
		return nil, fmt.Errorf("multiple components found in calendar")
	}

	// If no components are found, return an error
	if len(cal.Children) == 0 {
		return nil, fmt.Errorf("no components found in calendar")
	}

	// Return the first component (event or other)
	return cal.Children[0], nil
}
