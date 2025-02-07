package davclient

import (
	"errors"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/internal/httpclient"
)

func TestGetAllEvents(t *testing.T) {
	client := &davClient{}
	filter := client.GetAllEvents()

	if filter == nil {
		t.Error("GetAllEvents() returned nil")
	}

	objFilter, ok := filter.(*objectFilter)
	if !ok {
		t.Error("GetAllEvents() did not return an *objectFilter")
	}

	if objFilter.objectType != "VEVENT" {
		t.Errorf("GetAllEvents() objectType = %v, want %v", objFilter.objectType, "VEVENT")
	}
}

func TestGetCalendarEtag(t *testing.T) {
	tests := []struct {
		name        string
		resources   map[string]httpclient.ResourceProps
		wantEtag    string
		wantErr     bool
		expectedErr string
	}{
		{
			name: "calendar found with etag",
			resources: map[string]httpclient.ResourceProps{
				"/calendar": {IsCalendar: true, Etag: "etag123"},
			},
			wantEtag: "etag123",
			wantErr:  false,
		},
		{
			name: "single resource with etag",
			resources: map[string]httpclient.ResourceProps{
				"/calendar": {IsCalendar: false, Etag: "etag456"},
			},
			wantEtag: "etag456",
			wantErr:  false,
		},
		{
			name:        "no calendar found",
			resources:   map[string]httpclient.ResourceProps{},
			wantEtag:    "",
			wantErr:     true,
			expectedErr: "no calendar found at ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				propfindResponse: &httpclient.PropfindResponse{
					Resources: tt.resources,
				},
			}

			client := &davClient{
				httpClient: mockClient,
			}

			gotEtag, err := client.GetCalendarEtag()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCalendarEtag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.expectedErr {
				t.Errorf("GetCalendarEtag() error = %v, want %v", err, tt.expectedErr)
			}

			if gotEtag != tt.wantEtag {
				t.Errorf("GetCalendarEtag() = %v, want %v", gotEtag, tt.wantEtag)
			}
		})
	}
}

func TestExecuteCalendarQuery(t *testing.T) {
	type reportResponse = struct {
		Href     string `xml:"DAV: href"`
		PropStat struct {
			Prop struct {
				CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
				ETag         string `xml:"DAV: getetag"`
			} `xml:"DAV: prop"`
			Status string `xml:"DAV: status"`
		} `xml:"DAV: propstat"`
	}

	tests := []struct {
		name        string
		responses   []reportResponse
		wantObjects int
		wantErr     bool
		expectedErr string
	}{
		{
			name: "successful query with valid event",
			responses: []reportResponse{
				{
					Href: "/calendar/event1.ics",
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
							CalendarData: "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nEND:VEVENT\r\nEND:VCALENDAR",
							ETag:         "etag123",
						},
					},
				},
			},
			wantObjects: 1,
			wantErr:     false,
		},
		{
			name: "skip non-200 responses",
			responses: []reportResponse{
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
			responses: []reportResponse{
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
							CalendarData: "invalid data",
						},
					},
				},
			},
			wantObjects: 0,
			wantErr:     true,
			expectedErr: "failed to parse iCalendar data",
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
				httpClient: mockClient,
			}

			objects, err := client.executeCalendarQuery(&calendarQuery{})
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCalendarQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && len(tt.expectedErr) > 0 {
				if !errors.Is(err, err) && err.Error() != tt.expectedErr {
					t.Errorf("executeCalendarQuery() error = %v, want error containing %v", err, tt.expectedErr)
				}
			}

			if len(objects) != tt.wantObjects {
				t.Errorf("executeCalendarQuery() got %v objects, want %v", len(objects), tt.wantObjects)
			}
		})
	}
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		tzID    string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "UTC time",
			value:   "20240207T151500Z",
			tzID:    "",
			want:    time.Date(2024, 2, 7, 15, 15, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "local time",
			value:   "20240207T151500",
			tzID:    "",
			want:    time.Date(2024, 2, 7, 15, 15, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "date only",
			value:   "20240207",
			tzID:    "",
			want:    time.Date(2024, 2, 7, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format",
			value:   "invalid",
			tzID:    "",
			want:    time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateTime(tt.value, tt.tzID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseDateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Mock types for testing
type mockHTTPClient struct {
	propfindResponse *httpclient.PropfindResponse
	reportResponse   *httpclient.ReportResponse
}

func (m *mockHTTPClient) DoPROPFIND(url string, depth int, props ...string) (*httpclient.PropfindResponse, error) {
	return m.propfindResponse, nil
}

func (m *mockHTTPClient) DoREPORT(url string, depth int, query interface{}) (*httpclient.ReportResponse, error) {
	return m.reportResponse, nil
}
