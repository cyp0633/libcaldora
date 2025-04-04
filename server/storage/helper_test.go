package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
)

func TestICalCompToICS(t *testing.T) {
	tests := []struct {
		name                  string
		component             ical.Component
		removeCalendarWrapper bool
		want                  []string // substrings that should be in result
		dontWant              []string // substrings that should not be in result when wrapper is removed
		wantErr               bool
	}{
		{
			name: "basic event with wrapper",
			component: func() ical.Component {
				e := ical.NewEvent()
				e.Props.SetText(ical.PropSummary, "Test Event")
				e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
				e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
				e.Props.SetText(ical.PropUID, "test-event-1")
				return *e.Component
			}(),
			removeCalendarWrapper: false,
			want: []string{
				"BEGIN:VCALENDAR",
				"VERSION:2.0",
				"PRODID:-//Caldora//Go Calendar//EN",
				"BEGIN:VEVENT",
				"SUMMARY:Test Event",
				"DTSTART:20240101T100000Z",
				"DTEND:20240101T110000Z",
				"UID:test-event-1",
				"END:VEVENT",
				"END:VCALENDAR",
			},
			wantErr: false,
		},
		{
			name: "basic event without wrapper",
			component: func() ical.Component {
				e := ical.NewEvent()
				e.Props.SetText(ical.PropSummary, "Test Event")
				e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
				e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
				e.Props.SetText(ical.PropUID, "test-event-1")
				return *e.Component
			}(),
			removeCalendarWrapper: true,
			want: []string{
				"BEGIN:VEVENT",
				"SUMMARY:Test Event",
				"DTSTART:20240101T100000Z",
				"DTEND:20240101T110000Z",
				"UID:test-event-1",
				"END:VEVENT",
			},
			dontWant: []string{
				"BEGIN:VCALENDAR",
				"VERSION:2.0",
				"PRODID:-//Caldora//Go Calendar//EN",
				"END:VCALENDAR",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ICalCompToICS(tt.component, tt.removeCalendarWrapper)
			if (err != nil) != tt.wantErr {
				t.Errorf("IcalEventToICS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				for _, want := range tt.want {
					if !strings.Contains(got, want) {
						t.Errorf("IcalEventToICS() = %v\nwant substring: %v", got, want)
					}
					// Check that unwanted strings are not present (for removeCalendarWrapper case)
					if tt.removeCalendarWrapper && tt.dontWant != nil {
						for _, dontWant := range tt.dontWant {
							if strings.Contains(got, dontWant) {
								t.Errorf("IcalEventToICS() = %v\nshould not contain: %v", got, dontWant)
							}
						}
					}
				}
			}
		})
	}
}

func TestICSToICalComp(t *testing.T) {
	tests := []struct {
		name    string
		ics     string
		check   func(*testing.T, *ical.Component) // Check against component
		wantErr string
	}{
		{
			name: "valid single event",
			ics: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Caldora//Go Calendar//EN
BEGIN:VEVENT
SUMMARY:Test Event
DTSTART:20240101T100000Z
DTEND:20240101T110000Z
UID:test-event-1
END:VEVENT
END:VCALENDAR`,
			check: func(t *testing.T, c *ical.Component) {
				// Extract summary property from the component
				summary, err := c.Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary: %v", err)
				}
				if summary != "Test Event" {
					t.Errorf("got summary = %v, want Test Event", summary)
				}
			},
			wantErr: "",
		},
		{
			name: "empty calendar",
			ics: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Caldora//Go Calendar//EN
END:VCALENDAR`,
			check:   nil,
			wantErr: "no components found in calendar",
		},
		{
			name: "multiple events",
			ics: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Caldora//Go Calendar//EN
BEGIN:VEVENT
SUMMARY:Event 1
DTSTART:20240101T100000Z
DTEND:20240101T110000Z
UID:event-1
END:VEVENT
BEGIN:VEVENT
SUMMARY:Event 2
DTSTART:20240101T120000Z
DTEND:20240101T130000Z
UID:event-2
END:VEVENT
END:VCALENDAR`,
			check:   nil,
			wantErr: "multiple components found in calendar",
		},
		{
			name:    "invalid format",
			ics:     "invalid ics format",
			check:   nil,
			wantErr: "failed to decode calendar",
		},
		{
			name: "raw event without VCALENDAR wrapper",
			ics: `BEGIN:VEVENT
SUMMARY:Raw Event
DTSTART:20240101T100000Z
DTEND:20240101T110000Z
UID:raw-event-1
END:VEVENT`,
			check: func(t *testing.T, c *ical.Component) {
				// Extract summary property from the component
				summary, err := c.Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary: %v", err)
				}
				if summary != "Raw Event" {
					t.Errorf("got summary = %v, want Raw Event", summary)
				}
			},
			wantErr: "",
		},
		{
			name: "raw event with whitespace before",
			ics: `
BEGIN:VEVENT
SUMMARY:Raw Event With Whitespace
DTSTART:20240101T100000Z
DTEND:20240101T110000Z
UID:raw-event-2
END:VEVENT`,
			check: func(t *testing.T, c *ical.Component) {
				summary, err := c.Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary: %v", err)
				}
				if summary != "Raw Event With Whitespace" {
					t.Errorf("got summary = %v, want Raw Event With Whitespace", summary)
				}
			},
			wantErr: "",
		},
		{
			name: "raw todo component without wrapper",
			ics: `BEGIN:VTODO
SUMMARY:Raw Task
DUE:20240101T110000Z
UID:raw-task-1
END:VTODO`,
			check: func(t *testing.T, c *ical.Component) {
				if c.Name != "VTODO" {
					t.Errorf("expected VTODO component, got %s", c.Name)
				}
				summary, err := c.Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary: %v", err)
				}
				if summary != "Raw Task" {
					t.Errorf("got summary = %v, want Raw Task", summary)
				}
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ICSToICalComp(tt.ics)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("ICSToICalEvent() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ICSToICalEvent() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				return
			}
			if err != nil {
				t.Errorf("ICSToICalEvent() unexpected error: %v", err)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
