package storage

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

func ICalEventToICS(event ical.Event, removeCalendarWrapper bool) (string, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Caldora//Go Calendar//EN")

	// Ensure DTSTAMP is present
	if event.Props.Get(ical.PropDateTimeStamp) == nil {
		event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	}

	cal.Children = append(cal.Children, event.Component)

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

func ICSToICalEvent(ics string) (*ical.Event, error) {
	r := strings.NewReader(ics)
	dec := ical.NewDecoder(r)

	cal, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode calendar: %w", err)
	}

	events := cal.Events()
	if len(events) == 0 {
		return nil, fmt.Errorf("no events found in calendar")
	}
	if len(events) > 1 {
		return nil, fmt.Errorf("multiple events found in calendar")
	}

	return &events[0], nil
}
