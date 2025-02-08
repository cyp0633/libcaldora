package davclient

import (
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/internal/httpclient"
	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

func createTestEvent() *ical.Event {
	event := ical.NewEvent()
	event.Props.SetText("SUMMARY", "Test Event")
	event.Props.SetText("UID", uuid.New().String())
	event.Props.SetDateTime("DTSTAMP", time.Now().UTC())
	return event
}

func TestCreateCalendarObject(t *testing.T) {
	tests := []struct {
		name          string
		collectionURL string
		putResp       *mockPutResponse
		wantEtag      string
		wantErr       bool
		expectedErr   string
	}{
		{
			name:          "successful create with etag",
			collectionURL: "/calendar",
			putResp: &mockPutResponse{
				etag: "new-etag",
				err:  nil,
			},
			wantEtag: "new-etag",
			wantErr:  false,
		},
		{
			name:          "empty etag in PUT response gets new etag",
			collectionURL: "/calendar",
			putResp: &mockPutResponse{
				etag: "", // Empty etag in PUT response
				err:  nil,
			},
			wantEtag: "new-etag", // Should get this from PROPFIND
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				propfindResponse: &httpclient.PropfindResponse{
					Resources: map[string]httpclient.ResourceProps{},
				},
				putResponse: tt.putResp,
			}

			// Set up mock client to handle PROPFIND calls for newly created objects
			mockClient.doPropfind = func(url string, depth int, props ...string) (*httpclient.PropfindResponse, error) {
				// Return test's desired etag for any URL (since objectURL isn't known until after creation)
				return &httpclient.PropfindResponse{
					Resources: map[string]httpclient.ResourceProps{
						url: {Etag: "new-etag"},
					},
				}, nil
			}

			client := &davClient{
				httpClient: mockClient,
			}

			_, gotEtag, err := client.CreateCalendarObject(tt.collectionURL, createTestEvent())
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCalendarObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.expectedErr != "" {
				if err.Error() != tt.expectedErr {
					t.Errorf("CreateCalendarObject() error = %v, want %v", err, tt.expectedErr)
				}
			}

			if gotEtag != tt.wantEtag {
				t.Errorf("CreateCalendarObject() etag = %v, want %v", gotEtag, tt.wantEtag)
			}
		})
	}
}

func TestUpdateCalendarObject(t *testing.T) {
	tests := []struct {
		name        string
		objectURL   string
		resources   map[string]httpclient.ResourceProps
		putResp     *mockPutResponse
		wantEtag    string
		wantErr     bool
		expectedErr string
	}{
		{
			name:      "successful update with etag",
			objectURL: "/calendar/event.ics",
			resources: map[string]httpclient.ResourceProps{
				"/calendar/event.ics": {Etag: "original-etag"},
			},
			putResp: &mockPutResponse{
				etag: "updated-etag",
				err:  nil,
			},
			wantEtag: "updated-etag",
			wantErr:  false,
		},
		{
			name:        "object not found",
			objectURL:   "/calendar/nonexistent.ics",
			resources:   map[string]httpclient.ResourceProps{},
			wantEtag:    "",
			wantErr:     true,
			expectedErr: "object not found at /calendar/nonexistent.ics",
		},
		{
			name:      "empty etag in PUT response gets new etag",
			objectURL: "/calendar/event.ics",
			resources: map[string]httpclient.ResourceProps{
				"/calendar/event.ics": {Etag: "original-etag"},
			},
			putResp: &mockPutResponse{
				etag: "", // Empty etag in PUT response
				err:  nil,
			},
			wantEtag: "original-etag", // Should get this from second PROPFIND
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				propfindResponse: &httpclient.PropfindResponse{
					Resources: tt.resources,
				},
				putResponse: tt.putResp,
			}

			client := &davClient{
				httpClient: mockClient,
			}

			gotEtag, err := client.UpdateCalendarObject(tt.objectURL, createTestEvent())
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCalendarObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.expectedErr != "" {
				if err.Error() != tt.expectedErr {
					t.Errorf("UpdateCalendarObject() error = %v, want %v", err, tt.expectedErr)
				}
			}

			if gotEtag != tt.wantEtag {
				t.Errorf("UpdateCalendarObject() = %v, want %v", gotEtag, tt.wantEtag)
			}
		})
	}
}
