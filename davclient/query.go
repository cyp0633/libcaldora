package davclient

import (
	"bytes"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
)

// GetAllEvents returns a filter for querying all events
func (c *davClient) GetAllEvents() ObjectFilter {
	return &objectFilter{
		client:     c,
		objectType: "VEVENT", // Default to VEVENT
	}
}

// executeCalendarQuery sends a CalDAV REPORT request and returns the raw response
func (c *davClient) executeCalendarQuery(query *calendarQuery) ([]ical.Event, error) {
	resp, err := c.httpClient.DoREPORT(c.calendarURL, 1, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute calendar query: %w", err)
	}

	var events []ical.Event
	for _, response := range resp.Responses {
		if response.PropStat.Status != "HTTP/1.1 200 OK" {
			continue
		}

		// Parse iCalendar data
		calendar, err := ical.NewDecoder(bytes.NewReader([]byte(response.PropStat.Prop.CalendarData))).Decode()
		if err != nil {
			return nil, fmt.Errorf("failed to parse iCalendar data: %w", err)
		}

		// Extract events from calendar
		events = append(events, calendar.Events()...)
	}

	return events, nil
}

// parseDateTime parses an iCalendar date-time string
func parseDateTime(value string, tzID string) (time.Time, error) {
	formats := []string{
		"20060102T150405Z", // UTC
		"20060102T150405",  // Local time
		"20060102",         // Date only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			if tzID != "" {
				if loc, err := time.LoadLocation(tzID); err == nil {
					return t.In(loc), nil
				}
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date-time format: %s", value)
}
