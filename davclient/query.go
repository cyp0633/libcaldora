package davclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
)

// CalendarMultiGet represents a CalDAV calendar-multiget REPORT request
type calendarMultiGet struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-multiget"`
	Prop    struct {
		GetETag      struct{} `xml:"DAV: getetag"`
		CalendarData struct{} `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
	} `xml:"DAV: prop"`
	Hrefs []string `xml:"DAV: href"`
}

// CalendarObject represents a calendar object with its metadata
type CalendarObject struct {
	Event ical.Event
	URL   string
	ETag  string
}

// GetAllEvents returns a filter for querying all events
func (c *davClient) GetAllEvents() ObjectFilter {
	return &objectFilter{
		client:     c,
		objectType: "VEVENT", // Default to VEVENT
	}
}

// GetObjectETags returns only the ETags of calendar objects matching the filter criteria
func (c *davClient) GetObjectETags() ObjectFilter {
	return &objectFilter{
		client:     c,
		objectType: "VEVENT",
		etagOnly:   true,
	}
}

// GetObjectsByURLs retrieves calendar objects and their ETags for the specified URLs
func (c *davClient) GetObjectsByURLs(urls []string) ([]CalendarObject, error) {
	query := &calendarMultiGet{
		Hrefs: urls,
	}
	return c.executeCalendarMultiGet(query)
}

// GetCalendarEtag retrieves only the etag of the calendar to check for updates
func (c *davClient) GetCalendarEtag() (string, error) {
	resp, err := c.httpClient.DoPROPFIND(c.calendarURL, 0, "getetag")
	if err != nil {
		return "", fmt.Errorf("failed to get calendar etag: %w", err)
	}

	// Look for the calendar resource
	for _, props := range resp.Resources {
		if props.IsCalendar {
			return props.Etag, nil
		}
		// If there's only one resource and it has an etag, return it regardless of IsCalendar
		if len(resp.Resources) == 1 && props.Etag != "" {
			return props.Etag, nil
		}
	}

	return "", fmt.Errorf("no calendar found at %s", c.calendarURL)
}

// executeCalendarQuery sends a CalDAV REPORT query request and returns calendar objects with metadata
func (c *davClient) executeCalendarQuery(query *calendarQuery) ([]CalendarObject, error) {
	return c.doReport(query)
}

// executeCalendarMultiGet sends a CalDAV calendar-multiget REPORT request and returns calendar objects with metadata
func (c *davClient) executeCalendarMultiGet(query *calendarMultiGet) ([]CalendarObject, error) {
	return c.doReport(query)
}

// doReport sends a CalDAV REPORT request and processes the response
func (c *davClient) doReport(query interface{}) ([]CalendarObject, error) {
	resp, err := c.httpClient.DoREPORT(c.calendarURL, 1, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute calendar query: %w", err)
	}

	var objects []CalendarObject
	for _, response := range resp.Responses {
		if response.PropStat.Status != "HTTP/1.1 200 OK" {
			continue
		}

		if response.PropStat.Prop.CalendarData != "" {
			// Parse iCalendar data
			calendar, err := ical.NewDecoder(bytes.NewReader([]byte(response.PropStat.Prop.CalendarData))).Decode()
			if err != nil {
				return nil, fmt.Errorf("failed to parse iCalendar data: %w", err)
			}

			// Extract events and metadata
			for _, event := range calendar.Events() {
				objects = append(objects, CalendarObject{
					Event: event,
					URL:   response.Href,
					ETag:  response.PropStat.Prop.ETag,
				})
			}
		} else {
			// ETag-only case
			objects = append(objects, CalendarObject{
				URL:  response.Href,
				ETag: response.PropStat.Prop.ETag,
			})
		}
	}

	return objects, nil
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
