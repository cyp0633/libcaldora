package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
)

func TestIcalEventToICS(t *testing.T) {
	tests := []struct {
		name    string
		event   ical.Event
		want    []string // substrings that should be in result
		wantErr bool
	}{
		{
			name: "basic event",
			event: func() ical.Event {
				e := ical.NewEvent()
				e.Props.SetText(ical.PropSummary, "Test Event")
				e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
				e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
				e.Props.SetText(ical.PropUID, "test-event-1")
				return *e
			}(),
			want: []string{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IcalEventToICS(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("IcalEventToICS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				for _, want := range tt.want {
					if !strings.Contains(got, want) {
						t.Errorf("IcalEventToICS() = %v\nwant substring: %v", got, want)
					}
				}
			}
		})
	}
}

func TestICSToICalEvent(t *testing.T) {
	tests := []struct {
		name    string
		ics     string
		check   func(*testing.T, *ical.Event)
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
			check: func(t *testing.T, e *ical.Event) {
				summary, err := e.Props.Text(ical.PropSummary)
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
			wantErr: "no events found in calendar",
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
			wantErr: "multiple events found in calendar",
		},
		{
			name:    "invalid format",
			ics:     "invalid ics format",
			check:   nil,
			wantErr: "failed to decode calendar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ICSToICalEvent(tt.ics)
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
