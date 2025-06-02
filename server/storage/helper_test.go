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
		components            []ical.Component
		removeCalendarWrapper bool
		want                  []string // substrings that should be in result
		dontWant              []string // substrings that should not be in result when wrapper is removed
		wantErr               bool
	}{
		{
			name: "single event with wrapper",
			components: []ical.Component{func() ical.Component {
				e := ical.NewEvent()
				e.Props.SetText(ical.PropSummary, "Test Event")
				e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
				e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
				e.Props.SetText(ical.PropUID, "test-event-1")
				return *e.Component
			}()},
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
			name: "single event without wrapper",
			components: []ical.Component{func() ical.Component {
				e := ical.NewEvent()
				e.Props.SetText(ical.PropSummary, "Test Event")
				e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
				e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
				e.Props.SetText(ical.PropUID, "test-event-1")
				return *e.Component
			}()},
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
		{
			name: "multiple events with wrapper",
			components: []ical.Component{
				func() ical.Component {
					e := ical.NewEvent()
					e.Props.SetText(ical.PropSummary, "Event 1")
					e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
					e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
					e.Props.SetText(ical.PropUID, "event-1")
					return *e.Component
				}(),
				func() ical.Component {
					e := ical.NewEvent()
					e.Props.SetText(ical.PropSummary, "Event 2")
					e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC))
					e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 2, 13, 0, 0, 0, time.UTC))
					e.Props.SetText(ical.PropUID, "event-2")
					return *e.Component
				}(),
			},
			removeCalendarWrapper: false,
			want: []string{
				"BEGIN:VCALENDAR",
				"VERSION:2.0",
				"PRODID:-//Caldora//Go Calendar//EN",
				"BEGIN:VEVENT",
				"SUMMARY:Event 1",
				"UID:event-1",
				"END:VEVENT",
				"BEGIN:VEVENT",
				"SUMMARY:Event 2",
				"UID:event-2",
				"END:VEVENT",
				"END:VCALENDAR",
			},
			wantErr: false,
		},
		{
			name: "multiple events without wrapper",
			components: []ical.Component{
				func() ical.Component {
					e := ical.NewEvent()
					e.Props.SetText(ical.PropSummary, "Event 1")
					e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
					e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))
					e.Props.SetText(ical.PropUID, "event-1")
					return *e.Component
				}(),
				func() ical.Component {
					e := ical.NewEvent()
					e.Props.SetText(ical.PropSummary, "Event 2")
					e.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC))
					e.Props.SetDateTime(ical.PropDateTimeEnd, time.Date(2024, 1, 2, 13, 0, 0, 0, time.UTC))
					e.Props.SetText(ical.PropUID, "event-2")
					return *e.Component
				}(),
			},
			removeCalendarWrapper: true,
			want: []string{
				"BEGIN:VEVENT",
				"SUMMARY:Event 1",
				"UID:event-1",
				"END:VEVENT",
				"BEGIN:VEVENT",
				"SUMMARY:Event 2",
				"UID:event-2",
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
		{
			name:                  "empty components list",
			components:            []ical.Component{},
			removeCalendarWrapper: false,
			want:                  []string{},
			wantErr:               true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ICalCompToICS(tt.components, tt.removeCalendarWrapper)
			if (err != nil) != tt.wantErr {
				t.Errorf("ICalCompToICS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				for _, want := range tt.want {
					if !strings.Contains(got, want) {
						t.Errorf("ICalCompToICS() = %v\nwant substring: %v", got, want)
					}
				}
				// Check that unwanted strings are not present (for removeCalendarWrapper case)
				if tt.removeCalendarWrapper && tt.dontWant != nil {
					for _, dontWant := range tt.dontWant {
						if strings.Contains(got, dontWant) {
							t.Errorf("ICalCompToICS() = %v\nshould not contain: %v", got, dontWant)
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
		check   func(*testing.T, []*ical.Component) // Check against components
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
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 1 {
					t.Errorf("expected 1 component, got %d", len(components))
					return
				}
				c := components[0]
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
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 2 {
					t.Errorf("expected 2 components, got %d", len(components))
					return
				}

				summary1, err := components[0].Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary from first component: %v", err)
				}
				if summary1 != "Event 1" {
					t.Errorf("got first summary = %v, want Event 1", summary1)
				}

				summary2, err := components[1].Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary from second component: %v", err)
				}
				if summary2 != "Event 2" {
					t.Errorf("got second summary = %v, want Event 2", summary2)
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
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 1 {
					t.Errorf("expected 1 component, got %d", len(components))
					return
				}
				c := components[0]
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
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 1 {
					t.Errorf("expected 1 component, got %d", len(components))
					return
				}
				c := components[0]
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
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 1 {
					t.Errorf("expected 1 component, got %d", len(components))
					return
				}
				c := components[0]
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
		{
			name: "event with timezone information",
			ics: `BEGIN:VCALENDAR
PRODID:-//Mozilla.org/NONSGML Mozilla Calendar V1.1//EN
VERSION:2.0
BEGIN:VTIMEZONE
TZID:Asia/Shanghai
X-TZINFO:Asia/Shanghai[2025a]
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+080543
TZNAME:Asia/Shanghai(STD)
DTSTART:19010101T000000
RDATE:19010101T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19190413T000000
RDATE:19190413T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19191001T000000
RDATE:19191001T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19400601T000000
RDATE:19400601T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19401013T000000
RDATE:19401013T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19410315T000000
RDATE:19410315T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19411102T000000
RDATE:19411102T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19420131T000000
RDATE:19420131T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19450902T000000
RDATE:19450902T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19460515T000000
RDATE:19460515T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19461001T000000
RDATE:19461001T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19470415T000000
RDATE:19470415T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19471101T000000
RDATE:19471101T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19480501T000000
RDATE:19480501T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19481001T000000
RDATE:19481001T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19490501T000000
RDATE:19490501T000000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19490528T000000
RDATE:19490528T000000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19860504T020000
RDATE:19860504T020000
END:DAYLIGHT
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19870412T020000
RDATE:19870412T020000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19860914T020000
RRULE:FREQ=YEARLY;BYMONTH=9;BYDAY=2SU;UNTIL=19880911T020000
END:STANDARD
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19880417T020000
RRULE:FREQ=YEARLY;BYMONTH=4;BYDAY=3SU;UNTIL=19900415T020000
END:DAYLIGHT
BEGIN:DAYLIGHT
TZOFFSETTO:+090000
TZOFFSETFROM:+080000
TZNAME:Asia/Shanghai(DST)
DTSTART:19910414T020000
RDATE:19910414T020000
END:DAYLIGHT
BEGIN:STANDARD
TZOFFSETTO:+080000
TZOFFSETFROM:+090000
TZNAME:Asia/Shanghai(STD)
DTSTART:19890917T020000
RRULE:FREQ=YEARLY;BYMONTH=9;BYDAY=3SU;UNTIL=19910915T020000
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
CREATED:20250424T082302Z
LAST-MODIFIED:20250424T082307Z
DTSTAMP:20250424T082307Z
UID:cd7e9aa6-17ef-4eb3-a9b3-51383f8aaca3
SUMMARY:test
DTSTART;TZID=Asia/Shanghai:20250424T174500
DTEND;TZID=Asia/Shanghai:20250424T184500
TRANSP:OPAQUE
LOCATION:test
END:VEVENT
END:VCALENDAR`,
			check: func(t *testing.T, components []*ical.Component) {
				if len(components) != 1 {
					t.Errorf("expected 1 component, got %d", len(components))
					return
				}
				c := components[0]
				// Check summary
				summary, err := c.Props.Text(ical.PropSummary)
				if err != nil {
					t.Errorf("failed to get summary: %v", err)
				}
				if summary != "test" {
					t.Errorf("got summary = %v, want test", summary)
				}

				// Check UID
				uid, err := c.Props.Text(ical.PropUID)
				if err != nil {
					t.Errorf("failed to get UID: %v", err)
				}
				if uid != "cd7e9aa6-17ef-4eb3-a9b3-51383f8aaca3" {
					t.Errorf("got UID = %v, want cd7e9aa6-17ef-4eb3-a9b3-51383f8aaca3", uid)
				}

				// Check location
				location, err := c.Props.Text(ical.PropLocation)
				if err != nil {
					t.Errorf("failed to get location: %v", err)
				}
				if location != "test" {
					t.Errorf("got location = %v, want test", location)
				}

				// Check that start time has timezone parameter
				dtstart := c.Props.Get(ical.PropDateTimeStart)
				if dtstart == nil {
					t.Error("DTSTART property not found")
				} else {
					tzid := dtstart.Params.Get("TZID")
					if tzid == "" {
						t.Error("TZID parameter not found in DTSTART")
					} else if tzid != "Asia/Shanghai" {
						t.Errorf("got TZID = %v, want Asia/Shanghai", tzid[0])
					}
				}
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ICSToICalComp(tt.ics)
			if (err != nil) != (tt.wantErr != "") {
				t.Errorf("ICSToICalComp() error = %v, wantErr %v", err, tt.wantErr != "")
				return
			}
			if err != nil {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ICSToICalComp() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
