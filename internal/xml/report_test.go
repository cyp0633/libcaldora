package xml

import (
	"reflect"
	"testing"
	"time"

	"github.com/beevik/etree"
)

func TestReportRequest_Parse(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    *ReportRequest
		wantErr bool
	}{
		{
			name:    "empty document",
			xml:     "",
			wantErr: true,
		},
		{
			name:    "invalid root tag",
			xml:     `<?xml version="1.0" encoding="utf-8"?><wrong/>`,
			wantErr: true,
		},
		{
			name: "calendar-query with time range",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<C:calendar-query xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
<D:prop>
<D:getetag/>
<C:calendar-data/>
</D:prop>
<C:filter>
<C:comp-filter name="VCALENDAR">
<C:comp-filter name="VEVENT">
<C:time-range start="20250320T000000Z" end="20250322T235959Z"/>
</C:comp-filter>
</C:comp-filter>
</C:filter>
</C:calendar-query>`,
			want: &ReportRequest{
				Query: &CalendarQuery{
					Props: []string{"getetag", "calendar-data"},
					Filter: Filter{
						ComponentName: "VCALENDAR",
						SubFilter: &Filter{
							ComponentName: "VEVENT",
							TimeRange: &TimeRange{
								Start: parseTime("20250320T000000Z"),
								End:   parseTime("20250322T235959Z"),
							},
						},
					},
				},
			},
		},
		{
			name: "calendar-multiget",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:prop>
<D:getetag/>
<C:calendar-data/>
</D:prop>
<D:href>/calendars/user1/calendar1/1.ics</D:href>
<D:href>/calendars/user1/calendar1/2.ics</D:href>
</C:calendar-multiget>`,
			want: &ReportRequest{
				MultiGet: &CalendarMultiget{
					Props: []string{"getetag", "calendar-data"},
					Hrefs: []string{
						"/calendars/user1/calendar1/1.ics",
						"/calendars/user1/calendar1/2.ics",
					},
				},
			},
		},
		{
			name: "free-busy query",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<C:free-busy-query xmlns:C="urn:ietf:params:xml:ns:caldav">
<C:time-range start="20250320T000000Z" end="20250322T235959Z"/>
</C:free-busy-query>`,
			want: &ReportRequest{
				FreeBusy: &FreeBusyQuery{
					TimeRange: TimeRange{
						Start: parseTime("20250320T000000Z"),
						End:   parseTime("20250322T235959Z"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			if tt.xml != "" {
				err := doc.ReadFromString(tt.xml)
				if err != nil {
					t.Fatalf("failed to parse test XML: %v", err)
				}
			}

			var got ReportRequest
			err := got.Parse(doc)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportRequest.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if tt.want == nil {
					t.Error("ReportRequest.Parse() succeeded but want error")
					return
				}
				if !reflect.DeepEqual(&got, tt.want) {
					t.Errorf("ReportRequest.Parse() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestReportRequest_ToXML(t *testing.T) {
	tests := []struct {
		name    string
		request ReportRequest
		want    string
	}{
		{
			name: "calendar-query with time range",
			request: ReportRequest{
				Query: &CalendarQuery{
					Props: []string{"getetag", "calendar-data"},
					Filter: Filter{
						ComponentName: "VCALENDAR",
						SubFilter: &Filter{
							ComponentName: "VEVENT",
							TimeRange: &TimeRange{
								Start: parseTime("20250320T000000Z"),
								End:   parseTime("20250322T235959Z"),
							},
						},
					},
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:prop>
<D:getetag/>
<C:calendar-data/>
</D:prop>
<C:filter>
<C:comp-filter name="VCALENDAR">
<C:comp-filter name="VEVENT">
<C:time-range start="20250320T000000Z" end="20250322T235959Z"/>
</C:comp-filter>
</C:comp-filter>
</C:filter>
</C:calendar-query>`,
		},
		{
			name: "calendar-multiget",
			request: ReportRequest{
				MultiGet: &CalendarMultiget{
					Props: []string{"getetag", "calendar-data"},
					Hrefs: []string{
						"/calendars/user1/calendar1/1.ics",
						"/calendars/user1/calendar1/2.ics",
					},
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:prop>
<D:getetag/>
<C:calendar-data/>
</D:prop>
<D:href>/calendars/user1/calendar1/1.ics</D:href>
<D:href>/calendars/user1/calendar1/2.ics</D:href>
</C:calendar-multiget>`,
		},
		{
			name: "free-busy query",
			request: ReportRequest{
				FreeBusy: &FreeBusyQuery{
					TimeRange: TimeRange{
						Start: parseTime("20250320T000000Z"),
						End:   parseTime("20250322T235959Z"),
					},
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?>
<C:free-busy-query xmlns:C="urn:ietf:params:xml:ns:caldav">
<C:time-range start="20250320T000000Z" end="20250322T235959Z"/>
</C:free-busy-query>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.request.ToXML()
			got, err := doc.WriteToString()
			if err != nil {
				t.Fatalf("failed to serialize XML: %v", err)
			}

			// Use a better comparison approach with more logging
			gotNorm := normalizeXML(got)
			wantNorm := normalizeXML(tt.want)

			if gotNorm != wantNorm {
				// Print normalized versions to help debugging
				t.Errorf("ReportRequest.ToXML() normalized comparison failed\nGot (normalized): %s\nWant (normalized): %s\n\nOriginal:\nGot: %s\nWant: %s",
					gotNorm, wantNorm, got, tt.want)

				// Check for common issues
				for i := 0; i < len(gotNorm) && i < len(wantNorm); i++ {
					if gotNorm[i] != wantNorm[i] {
						t.Errorf("First difference at position %d: got '%c' (ASCII %d), want '%c' (ASCII %d)",
							i, gotNorm[i], gotNorm[i], wantNorm[i], wantNorm[i])
						// Show context around the difference
						start := max(0, i-10)
						end := min(len(gotNorm), i+10)
						t.Errorf("Context: '...%s...'", gotNorm[start:end])
						break
					}
				}
			}
		})
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper function to parse time in CalDAV format
func parseTime(s string) *time.Time {
	t, _ := time.Parse("20060102T150405Z", s)
	return &t
}
