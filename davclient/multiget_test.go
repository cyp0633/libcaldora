package davclient

import (
	"testing"

	"github.com/cyp0633/libcaldora/internal/httpclient"
)

func TestGetObjectsByURLs(t *testing.T) {
	tests := []struct {
		name      string
		responses []struct {
			Href     string `xml:"DAV: href"`
			PropStat struct {
				Prop struct {
					CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
					ETag         string `xml:"DAV: getetag"`
				} `xml:"DAV: prop"`
				Status string `xml:"DAV: status"`
			} `xml:"DAV: propstat"`
		}
		urls        []string
		wantObjects int
		wantErr     bool
	}{
		{
			name: "successful multiget with two events",
			urls: []string{
				"/calendars/user/calendar/1.ics",
				"/calendars/user/calendar/2.ics",
			},
			responses: []struct {
				Href     string `xml:"DAV: href"`
				PropStat struct {
					Prop struct {
						CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
						ETag         string `xml:"DAV: getetag"`
					} `xml:"DAV: prop"`
					Status string `xml:"DAV: status"`
				} `xml:"DAV: propstat"`
			}{
				{
					Href: "/calendars/user/calendar/1.ics",
					PropStat: struct {
						Prop struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						} `xml:"DAV: prop"`
						Status string `xml:"DAV: status"`
					}{
						Status: "HTTP/1.1 200 OK",
						Prop: struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						}{
							CalendarData: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:event1\r\nSUMMARY:Test Event 1\r\nEND:VEVENT\r\nEND:VCALENDAR",
							ETag:         `"123"`,
						},
					},
				},
				{
					Href: "/calendars/user/calendar/2.ics",
					PropStat: struct {
						Prop struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						} `xml:"DAV: prop"`
						Status string `xml:"DAV: status"`
					}{
						Status: "HTTP/1.1 200 OK",
						Prop: struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						}{
							CalendarData: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:event2\r\nSUMMARY:Test Event 2\r\nEND:VEVENT\r\nEND:VCALENDAR",
							ETag:         `"456"`,
						},
					},
				},
			},
			wantObjects: 2,
			wantErr:     false,
		},
		{
			name: "skip non-200 responses",
			urls: []string{"/calendars/user/calendar/1.ics"},
			responses: []struct {
				Href     string `xml:"DAV: href"`
				PropStat struct {
					Prop struct {
						CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
						ETag         string `xml:"DAV: getetag"`
					} `xml:"DAV: prop"`
					Status string `xml:"DAV: status"`
				} `xml:"DAV: propstat"`
			}{
				{
					PropStat: struct {
						Prop struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						} `xml:"DAV: prop"`
						Status string `xml:"DAV: status"`
					}{
						Status: "HTTP/1.1 404 Not Found",
					},
				},
			},
			wantObjects: 0,
			wantErr:     false,
		},
		{
			name: "invalid calendar data",
			urls: []string{"/calendars/user/calendar/1.ics"},
			responses: []struct {
				Href     string `xml:"DAV: href"`
				PropStat struct {
					Prop struct {
						CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
						ETag         string `xml:"DAV: getetag"`
					} `xml:"DAV: prop"`
					Status string `xml:"DAV: status"`
				} `xml:"DAV: propstat"`
			}{
				{
					PropStat: struct {
						Prop struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						} `xml:"DAV: prop"`
						Status string `xml:"DAV: status"`
					}{
						Status: "HTTP/1.1 200 OK",
						Prop: struct {
							CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
							ETag         string `xml:"DAV: getetag"`
						}{
							CalendarData: "invalid calendar data",
						},
					},
				},
			},
			wantObjects: 0,
			wantErr:     true,
		},
		{
			name:        "empty URLs list",
			urls:        []string{},
			responses:   nil,
			wantObjects: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				reportResponse: &httpclient.ReportResponse{
					Responses: tt.responses,
				},
			}

			client := &davClient{
				httpClient:  mockClient,
				calendarURL: "/calendar",
			}

			objects, err := client.GetObjectsByURLs(tt.urls)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetObjectsByURLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(objects) != tt.wantObjects {
				t.Errorf("GetObjectsByURLs() got %v objects, want %v", len(objects), tt.wantObjects)
				return
			}

			if tt.wantObjects > 0 {
				// Check first event's data
				summary1, _ := objects[0].Event.Props.Text("SUMMARY")
				if summary1 != "Test Event 1" {
					t.Errorf("First event summary = %v, want %v", summary1, "Test Event 1")
				}

				// Check second event's data
				if tt.wantObjects > 1 {
					summary2, _ := objects[1].Event.Props.Text("SUMMARY")
					if summary2 != "Test Event 2" {
						t.Errorf("Second event summary = %v, want %v", summary2, "Test Event 2")
					}
				}
			}
		})
	}
}
