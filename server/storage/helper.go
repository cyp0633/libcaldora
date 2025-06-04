package storage

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// ICalCompToICS converts ical.Component(s) (event or other calendar components) to an ICS string.
func ICalCompToICS(components []*ical.Component, removeCalendarWrapper bool) (string, error) {
	if len(components) == 0 {
		return "", fmt.Errorf("no components provided")
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Caldora//Go Calendar//EN")

	// Add all components to the calendar
	for _, component := range components {
		if component == nil {
			continue // Skip nil components
		}
		// Ensure DTSTAMP is present
		if component.Props.Get(ical.PropDateTimeStamp) == nil {
			component.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
		}
		cal.Children = append(cal.Children, component)
	}

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return "", fmt.Errorf("failed to encode calendar: %w", err)
	}

	icsString := buf.String()

	if removeCalendarWrapper {
		// For multiple components, we need to extract all of them
		// Determine line ending type used in the ICS file
		lineEnding := "\n"
		if strings.Contains(icsString, "\r\n") {
			lineEnding = "\r\n"
		}

		// Split by line endings to process line by line
		lines := strings.Split(icsString, lineEnding)

		var extractedLines []string
		var inComponent bool
		var componentDepth int

		// Find and extract all component sections (VEVENT, VTODO, etc.)
		for _, line := range lines {
			if strings.HasPrefix(line, "BEGIN:") && line != "BEGIN:VCALENDAR" {
				inComponent = true
				componentDepth = 1
				extractedLines = append(extractedLines, line)
			} else if inComponent && strings.HasPrefix(line, "BEGIN:") {
				componentDepth++
				extractedLines = append(extractedLines, line)
			} else if inComponent && strings.HasPrefix(line, "END:") {
				extractedLines = append(extractedLines, line)
				componentDepth--
				if componentDepth == 0 {
					inComponent = false
				}
			} else if inComponent {
				extractedLines = append(extractedLines, line)
			}
		}

		if len(extractedLines) > 0 {
			return strings.Join(extractedLines, lineEnding), nil
		}
	}

	return icsString, nil
}

// ICSToICalComp parses an ICS string and returns ical.Component(s) (event or other calendar components).
// It automatically adds the VCALENDAR wrapper if not present in the input.
func ICSToICalComp(ics string) ([]*ical.Component, error) {
	// Add the VCALENDAR wrapper if not present
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

	// If no components are found, return an error
	if len(cal.Children) == 0 {
		return nil, fmt.Errorf("no components found in calendar")
	}

	// Return all components including VTIMEZONE
	return cal.Children, nil
}
