package davclient

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
)

// ObjectFilter is the interface for filtering calendar objects
type ObjectFilter interface {
	TimeRange(start, end time.Time) ObjectFilter
	HasAlarm() ObjectFilter
	ObjectType(objType string) ObjectFilter
	Priority(priority int) ObjectFilter
	Categories(categories ...string) ObjectFilter
	Status(status string) ObjectFilter
	NotStatus(status string) ObjectFilter
	Summary(summary string) ObjectFilter
	Description(desc string) ObjectFilter
	Location(location string) ObjectFilter
	Organizer(organizer string) ObjectFilter
	Limit(limit int) ObjectFilter
	Do() ([]ical.Event, error)
}

// calendarQuerier is an interface for the calendar query operations needed by objectFilter
type calendarQuerier interface {
	executeCalendarQuery(*calendarQuery) ([]ical.Event, error)
}

// Filter represents the main filter builder
type objectFilter struct {
	client      calendarQuerier
	timeRange   *TimeRange
	hasAlarm    bool
	objectType  string
	priority    *int
	categories  []string
	status      string
	summary     string
	description string
	location    string
	organizer   string
	notStatus   string
	limit       int
	err         error
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Implementation of ObjectFilter methods
func (f *objectFilter) TimeRange(start, end time.Time) ObjectFilter {
	f.timeRange = &TimeRange{Start: start, End: end}
	return f
}

func (f *objectFilter) HasAlarm() ObjectFilter {
	f.hasAlarm = true
	return f
}

func (f *objectFilter) ObjectType(objType string) ObjectFilter {
	f.objectType = objType
	return f
}

func (f *objectFilter) Priority(priority int) ObjectFilter {
	f.priority = &priority
	return f
}

func (f *objectFilter) Categories(categories ...string) ObjectFilter {
	f.categories = categories
	return f
}

func (f *objectFilter) Status(status string) ObjectFilter {
	f.status = status
	return f
}

func (f *objectFilter) NotStatus(status string) ObjectFilter {
	f.notStatus = status
	return f
}

func (f *objectFilter) Summary(summary string) ObjectFilter {
	f.summary = summary
	return f
}

func (f *objectFilter) Description(desc string) ObjectFilter {
	f.description = desc
	return f
}

func (f *objectFilter) Location(location string) ObjectFilter {
	f.location = location
	return f
}

func (f *objectFilter) Organizer(organizer string) ObjectFilter {
	f.organizer = organizer
	return f
}

func (f *objectFilter) Limit(limit int) ObjectFilter {
	f.limit = limit
	return f
}

// buildCalendarQuery converts the filter to CalDAV XML
func (f *objectFilter) buildCalendarQuery() (*calendarQuery, error) {
	query := &calendarQuery{
		XMLName: xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-query"},
		Prop: prop{
			GetETag:      &struct{}{},
			CalendarData: &struct{}{},
		},
		Filter: filter{
			CompFilter: compFilter{
				Name: "VCALENDAR",
				CompFilter: &compFilter{
					Name: f.objectType,
				},
			},
		},
	}

	// Build inner comp-filter for VEVENT/VTODO
	innerFilter := query.Filter.CompFilter.CompFilter

	// Add time range if specified
	if f.timeRange != nil {
		innerFilter.TimeRange = &timeRange{
			Start: f.timeRange.Start.UTC().Format("20060102T150405Z"),
			End:   f.timeRange.End.UTC().Format("20060102T150405Z"),
		}
	}

	// Add prop filters
	var propFilters []propFilter

	if f.summary != "" {
		propFilters = append(propFilters, propFilter{
			Name:      "SUMMARY",
			TextMatch: &textMatch{Text: f.summary},
		})
	}

	if f.description != "" {
		propFilters = append(propFilters, propFilter{
			Name:      "DESCRIPTION",
			TextMatch: &textMatch{Text: f.description},
		})
	}

	if f.location != "" {
		propFilters = append(propFilters, propFilter{
			Name:      "LOCATION",
			TextMatch: &textMatch{Text: f.location},
		})
	}

	if f.status != "" {
		propFilters = append(propFilters, propFilter{
			Name:      "STATUS",
			TextMatch: &textMatch{Text: f.status},
		})
	}

	if f.notStatus != "" {
		propFilters = append(propFilters, propFilter{
			Name: "STATUS",
			TextMatch: &textMatch{
				Text:            f.notStatus,
				NegateCondition: true,
			},
		})
	}

	if f.priority != nil {
		propFilters = append(propFilters, propFilter{
			Name:      "PRIORITY",
			TextMatch: &textMatch{Text: fmt.Sprintf("%d", *f.priority)},
		})
	}

	if len(f.categories) > 0 {
		propFilters = append(propFilters, propFilter{
			Name:      "CATEGORIES",
			TextMatch: &textMatch{Text: f.categories[0]}, // TODO: Support multiple categories
		})
	}

	if f.organizer != "" {
		propFilters = append(propFilters, propFilter{
			Name:      "ORGANIZER",
			TextMatch: &textMatch{Text: f.organizer},
		})
	}

	if len(propFilters) > 0 {
		innerFilter.PropFilters = propFilters
	}

	return query, nil
}

// XML structs for calendar-query
type calendarQuery struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Prop    prop     `xml:"DAV: prop"`
	Filter  filter   `xml:"urn:ietf:params:xml:ns:caldav filter"`
}

type prop struct {
	GetETag      *struct{} `xml:"DAV: getetag"`
	CalendarData *struct{} `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
}

type filter struct {
	CompFilter compFilter `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
}

type compFilter struct {
	Name        string       `xml:"name,attr"`
	Test        string       `xml:"test,attr,omitempty"`
	TimeRange   *timeRange   `xml:"urn:ietf:params:xml:ns:caldav time-range,omitempty"`
	CompFilter  *compFilter  `xml:"urn:ietf:params:xml:ns:caldav comp-filter,omitempty"`
	PropFilters []propFilter `xml:"urn:ietf:params:xml:ns:caldav prop-filter,omitempty"`
}

type propFilter struct {
	Name      string     `xml:"name,attr"`
	TextMatch *textMatch `xml:"urn:ietf:params:xml:ns:caldav text-match,omitempty"`
}

type textMatch struct {
	Text            string `xml:",chardata"`
	NegateCondition bool   `xml:"negate-condition,attr,omitempty"`
}

type timeRange struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

// Do executes the filter and returns the matching events
func (f *objectFilter) Do() ([]ical.Event, error) {
	if f.err != nil {
		return nil, f.err
	}

	query, err := f.buildCalendarQuery()
	if err != nil {
		return nil, fmt.Errorf("failed to build calendar query: %w", err)
	}

	// Execute the calendar query through the client
	events, err := f.client.executeCalendarQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute calendar query: %w", err)
	}

	// Apply limit if specified
	if f.limit > 0 && len(events) > f.limit {
		events = events[:f.limit]
	}

	return events, nil
}
