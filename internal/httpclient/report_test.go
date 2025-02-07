package httpclient

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockQuery struct {
	XMLName xml.Name `xml:"query"`
	Value   string   `xml:"value"`
}

func TestDoREPORT(t *testing.T) {
	tests := []struct {
		name          string
		query         interface{}
		serverHandler func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		validateResp  func(*ReportResponse) bool
	}{
		{
			name: "successful request",
			query: &mockQuery{
				Value: "test",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "REPORT" {
					t.Errorf("expected REPORT method, got %s", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/xml; charset=utf-8" {
					t.Errorf("expected Content-Type application/xml, got %s", ct)
				}
				if depth := r.Header.Get("Depth"); depth != "0" {
					t.Errorf("expected Depth 0, got %s", depth)
				}

				// Return valid response
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
					<D:multistatus xmlns:D="DAV:">
						<D:response>
							<D:href>/calendar/event1.ics</D:href>
							<D:propstat>
								<D:prop>
<C:calendar-data xmlns:C="urn:ietf:params:xml:ns:caldav">BEGIN:VCALENDAR...</C:calendar-data>
<D:getetag>"123"</D:getetag>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`))
			},
			wantErr: false,
			validateResp: func(resp *ReportResponse) bool {
				return len(resp.Responses) == 1 &&
					resp.Responses[0].Href == "/calendar/event1.ics" &&
					resp.Responses[0].PropStat.Status == "HTTP/1.1 200 OK" &&
					resp.Responses[0].PropStat.Prop.CalendarData == "BEGIN:VCALENDAR..." &&
					resp.Responses[0].PropStat.Prop.ETag == `"123"`
			},
		},
		{
			name:  "invalid query",
			query: make(chan int), // Channels cannot be marshaled to XML
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called")
			},
			wantErr: true,
		},
		{
			name: "server error",
			query: &mockQuery{
				Value: "test",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "invalid response XML",
			query: &mockQuery{
				Value: "test",
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid XML`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			// Create client
			client := &httpClientWrapper{
				client: server.Client(),
			}

			// Execute request
			resp, err := client.DoREPORT(server.URL, 0, tt.query)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("DoREPORT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Validate response if needed
			if !tt.wantErr && tt.validateResp != nil {
				if !tt.validateResp(resp) {
					t.Error("response validation failed")
				}
			}
		})
	}
}
