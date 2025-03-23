package xml

import (
	"reflect"
	"testing"

	"github.com/beevik/etree"
)

func TestMultistatusResponse_ToXML(t *testing.T) {
	tests := []struct {
		name     string
		response MultistatusResponse
		want     string
	}{
		{
			name: "single response with one property",
			response: MultistatusResponse{
				Responses: []Response{
					{
						Href: "/calendars/user/calendar1",
						PropStats: []PropStat{
							{
								Props: []Property{
									{
										Name:        "displayname",
										TextContent: "Calendar 1",
										Attributes:  make(map[string]string),
									},
								},
								Status: "HTTP/1.1 200 OK",
							},
						},
					},
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
<D:response>
<D:href>/calendars/user/calendar1</D:href>
<D:propstat>
<D:prop><D:displayname>Calendar 1</D:displayname></D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`,
		},
		{
			name: "multiple responses with different properties",
			response: MultistatusResponse{
				Responses: []Response{
					{
						Href: "/calendars/user/calendar1",
						PropStats: []PropStat{
							{
								Props: []Property{
									{
										Name:      "resourcetype",
										Namespace: DAV,
										Children: []Property{
											{
												Name:       "collection",
												Namespace:  DAV,
												Attributes: make(map[string]string),
											},
											{
												Name:       "calendar",
												Namespace:  CalDAV,
												Attributes: make(map[string]string),
											},
										},
										Attributes: make(map[string]string),
									},
								},
								Status: "HTTP/1.1 200 OK",
							},
						},
					},
					{
						Href: "/calendars/user/calendar2",
						Error: &Error{
							Tag:     "not-found",
							Message: "Resource not found",
						},
					},
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
<D:response>
<D:href>/calendars/user/calendar1</D:href>
<D:propstat>
<D:prop><D:resourcetype><D:collection/><C:calendar/></D:resourcetype></D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
<D:response>
<D:href>/calendars/user/calendar2</D:href>
<D:error><not-found>Resource not found</not-found></D:error>
</D:response>
</D:multistatus>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.response.ToXML()
			got, err := doc.WriteToString()
			if err != nil {
				t.Fatalf("failed to serialize XML: %v", err)
			}

			// Normalize spaces and newlines for comparison
			gotNorm := normalizeXML(got)
			wantNorm := normalizeXML(tt.want)

			if gotNorm != wantNorm {
				t.Errorf("MultistatusResponse.ToXML() =\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestMultistatusResponse_Parse(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    *MultistatusResponse
		wantErr bool
	}{
		{
			name: "single response with one property",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:response>
<D:href>/calendars/user/calendar1</D:href>
<D:propstat>
<D:prop><D:displayname>Calendar 1</D:displayname></D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`,
			want: &MultistatusResponse{
				Responses: []Response{
					{
						Href: "/calendars/user/calendar1",
						PropStats: []PropStat{
							{
								Props: []Property{
									{
										Name:        "displayname",
										Namespace:   DAV,
										TextContent: "Calendar 1",
										Attributes:  make(map[string]string),
									},
								},
								Status: "HTTP/1.1 200 OK",
							},
						},
					},
				},
			},
		},
		{
			name: "response with error",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response>
<D:href>/calendars/user/calendar2</D:href>
<D:error><D:resource-must-be-null/></D:error>
</D:response>
</D:multistatus>`,
			want: &MultistatusResponse{
				Responses: []Response{
					{
						Href: "/calendars/user/calendar2",
						Error: &Error{
							Tag:       "resource-must-be-null",
							Namespace: DAV,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			err := doc.ReadFromString(tt.xml)
			if err != nil {
				t.Fatalf("failed to parse test XML: %v", err)
			}

			var got MultistatusResponse
			err = got.Parse(doc)

			if (err != nil) != tt.wantErr {
				t.Errorf("MultistatusResponse.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if tt.want == nil {
					t.Error("MultistatusResponse.Parse() succeeded but want error")
					return
				}
				if !reflect.DeepEqual(&got, tt.want) {
					t.Errorf("MultistatusResponse.Parse() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}
