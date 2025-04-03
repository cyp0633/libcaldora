package storage

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

func IcalEventToICS(event ical.Event) (string, error) {
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
	return buf.String(), nil
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
