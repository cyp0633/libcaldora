package httpclient

import (
	"bytes"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
)

func TestBuildPropfindXML(t *testing.T) {
	tests := []struct {
		name     string
		props    []string
		wantXML  string
		wantFail bool
	}{
		{
			name:    "empty props",
			props:   []string{},
			wantXML: `<?xml version="1.0" encoding="UTF-8"?><D:propfind xmlns:D="DAV:"><D:prop></D:prop></D:propfind>`,
		},
		{
			name:    "single prop",
			props:   []string{"resourcetype"},
			wantXML: `<?xml version="1.0" encoding="UTF-8"?><D:propfind xmlns:D="DAV:"><D:prop><D:resourcetype/></D:prop></D:propfind>`,
		},
		{
			name:    "multiple props",
			props:   []string{"resourcetype", "displayname", "calendar-color"},
			wantXML: `<?xml version="1.0" encoding="UTF-8"?><D:propfind xmlns:D="DAV:"><D:prop><D:resourcetype/><D:displayname/><IC:calendar-color xmlns:IC="http://apple.com/ns/ical/"/></D:prop></D:propfind>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPropfindXML(tt.props...)

			// Normalize XML for comparison
			var gotBuf, wantBuf bytes.Buffer
			if err := xml.NewDecoder(bytes.NewReader(got)).Decode(&gotBuf); err != nil && !tt.wantFail {
				t.Errorf("buildPropfindXML() invalid XML: %v", err)
			}
			if err := xml.NewDecoder(bytes.NewReader([]byte(tt.wantXML))).Decode(&wantBuf); err != nil && !tt.wantFail {
				t.Errorf("test case has invalid XML: %v", err)
			}

			if !tt.wantFail && gotBuf.String() != wantBuf.String() {
				t.Errorf("buildPropfindXML() = %v, want %v", gotBuf.String(), wantBuf.String())
			}
		})
	}
}

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestDoPROPFIND(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		depth      int
		props      []string
		wantErr    bool
		wantResult *PropfindResponse
	}{
		{
			name: "calendar response",
			response: `<?xml version="1.0" encoding="UTF-8"?>
                <D:multistatus xmlns:D="DAV:">
                    <D:response>
                        <D:href>/calendars/user/calendar1/</D:href>
                        <D:propstat>
                            <D:prop>
                                <D:resourcetype><C:calendar xmlns:C="urn:ietf:params:xml:ns:caldav"/></D:resourcetype>
                                <D:displayname>My Calendar</D:displayname>
                                <IC:calendar-color xmlns:IC="http://apple.com/ns/ical/">#FF0000</IC:calendar-color>
                            </D:prop>
                            <D:status>HTTP/1.1 200 OK</D:status>
                        </D:propstat>
                    </D:response>
                </D:multistatus>`,
			depth: 0,
			props: []string{"resourcetype", "displayname", "calendar-color"},
			wantResult: &PropfindResponse{
				Resources: map[string]ResourceProps{
					"/calendars/user/calendar1/": {
						IsCalendar:  true,
						DisplayName: "My Calendar",
						Color:       "#FF0000",
						CanWrite:    false,
					},
				},
			},
		},
		{
			name: "calendar with write-properties privilege",
			response: `<?xml version="1.0" encoding="UTF-8"?>
                <D:multistatus xmlns:D="DAV:">
                    <D:response>
                        <D:href>/calendars/user/calendar1/</D:href>
                        <D:propstat>
                            <D:prop>
                                <D:resourcetype><C:calendar xmlns:C="urn:ietf:params:xml:ns:caldav"/></D:resourcetype>
                                <D:displayname>My Calendar</D:displayname>
                                <D:current-user-privilege-set>
                                    <D:privilege><D:read/></D:privilege>
                                    <D:privilege><D:write-properties/></D:privilege>
                                </D:current-user-privilege-set>
                            </D:prop>
                            <D:status>HTTP/1.1 200 OK</D:status>
                        </D:propstat>
                    </D:response>
                </D:multistatus>`,
			depth: 0,
			props: []string{"resourcetype", "displayname", "current-user-privilege-set"},
			wantResult: &PropfindResponse{
				Resources: map[string]ResourceProps{
					"/calendars/user/calendar1/": {
						IsCalendar:  true,
						DisplayName: "My Calendar",
						CanWrite:    true,
					},
				},
			},
		},
		{
			name: "calendar read only",
			response: `<?xml version="1.0" encoding="UTF-8"?>
                <D:multistatus xmlns:D="DAV:">
                    <D:response>
                        <D:href>/calendars/user/calendar1/</D:href>
                        <D:propstat>
                            <D:prop>
                                <D:resourcetype><C:calendar xmlns:C="urn:ietf:params:xml:ns:caldav"/></D:resourcetype>
                                <D:displayname>Shared Calendar</D:displayname>
                                <D:current-user-privilege-set>
                                    <D:privilege><D:read/></D:privilege>
                                </D:current-user-privilege-set>
                            </D:prop>
                            <D:status>HTTP/1.1 200 OK</D:status>
                        </D:propstat>
                    </D:response>
                </D:multistatus>`,
			depth: 0,
			props: []string{"resourcetype", "displayname", "current-user-privilege-set"},
			wantResult: &PropfindResponse{
				Resources: map[string]ResourceProps{
					"/calendars/user/calendar1/": {
						IsCalendar:  true,
						DisplayName: "Shared Calendar",
						CanWrite:    false,
					},
				},
			},
		},
		{
			name: "error response",
			response: `<?xml version="1.0" encoding="UTF-8"?>
                <D:error xmlns:D="DAV:">
                    <D:status>HTTP/1.1 403 Forbidden</D:status>
                </D:error>`,
			depth:   0,
			props:   []string{"resourcetype"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusMultiStatus,
					Body:       io.NopCloser(bytes.NewBufferString(tt.response)),
				},
			}

			client := &http.Client{Transport: mock}
			baseURL, _ := url.Parse("http://example.com")
			wrapper := &httpClientWrapper{
				client:  client,
				baseURL: *baseURL,
				logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			got, err := wrapper.DoPROPFIND("http://example.com", tt.depth, tt.props...)

			if (err != nil) != tt.wantErr {
				t.Errorf("DoPROPFIND() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Compare results
				for href, want := range tt.wantResult.Resources {
					got, ok := got.Resources[href]
					if !ok {
						t.Errorf("DoPROPFIND() missing resource %s", href)
						continue
					}
					if got != want {
						t.Errorf("DoPROPFIND() resource %s = %v, want %v", href, got, want)
					}
				}
			}
		})
	}
}
