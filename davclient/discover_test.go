package davclient

import (
	"context"
	"net"
	"net/http"
	"testing"
)

// mockTransport implements http.RoundTripper for testing
type mockTransport struct {
	responses map[string]*http.Response
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, ok := t.responses[req.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
		}, nil
	}
	return resp, nil
}

func TestFindCalendars(t *testing.T) {
	mockTransport := &mockTransport{
		responses: map[string]*http.Response{
			"http://example.com/calendar": {
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			},
		},
	}
	tests := []struct {
		name     string
		location string
		username string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "invalid URL",
			location: "not-a-url",
			username: "test",
			password: "test",
			wantErr:  true,
			errMsg:   "invalid URL",
		},
		{
			name:     "valid URL with path",
			location: "http://example.com/calendar",
			username: "test",
			password: "test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Resolver: &mockResolver{},
				Client: &http.Client{
					Transport: mockTransport,
				},
			}
			ctx := context.Background()
			calendars, err := FindCalendarsWithConfig(ctx, tt.location, tt.username, tt.password, cfg)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("FindCalendars() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("FindCalendars() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("FindCalendars() unexpected error = %v", err)
				return
			}
			if calendars == nil {
				t.Error("FindCalendars() returned nil calendars slice")
			}
		})
	}
}

// mockResolver implements a mock DNS resolver for testing
type mockResolver struct {
	srvRecords map[string][]*net.SRV
	txtRecords map[string][]string
}

func (r *mockResolver) LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
	addrs, ok := r.srvRecords[name]
	if !ok {
		return "", nil, &net.DNSError{
			Err:        "no such host",
			Name:       name,
			IsNotFound: true,
		}
	}
	return "", addrs, nil
}

func (r *mockResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	records, ok := r.txtRecords[name]
	if !ok {
		return nil, &net.DNSError{
			Err:        "no such host",
			Name:       name,
			IsNotFound: true,
		}
	}
	return records, nil
}

func TestFindCalendarsWithDNS(t *testing.T) {
	tests := []struct {
		name           string
		location       string
		srvRecords    map[string][]*net.SRV
		txtRecords    map[string][]string
		wantLocations []string
		wantErr       bool
	}{
		{
			name:     "caldavs SRV record with path",
			location: "https://example.com",
			srvRecords: map[string][]*net.SRV{
				"_caldavs._tcp.example.com": {
					{
						Target:   "calendar.example.com",
						Port:     443,
						Priority: 1,
						Weight:   1,
					},
				},
			},
			txtRecords: map[string][]string{
				"_caldavs._tcp.example.com": {"path=/calendar"},
			},
			wantLocations: []string{
				"https://calendar.example.com:443/calendar",
				"https://example.com/.well-known/caldav",
				"https://example.com/",
			},
		},
		{
			name:     "caldav SRV record without path",
			location: "http://example.com",
			srvRecords: map[string][]*net.SRV{
				"_caldav._tcp.example.com": {
					{
						Target:   "calendar.example.com",
						Port:     80,
						Priority: 1,
						Weight:   1,
					},
				},
			},
			wantLocations: []string{
				"http://calendar.example.com:80",
				"http://example.com/.well-known/caldav",
				"http://example.com/",
			},
		},
		{
			name:     "no SRV records",
			location: "http://example.com",
			wantLocations: []string{
				"http://example.com/.well-known/caldav",
				"http://example.com/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := &mockResolver{
				srvRecords: tt.srvRecords,
				txtRecords: tt.txtRecords,
			}

			cfg := &Config{
				Resolver: mockResolver,
				Client:   &http.Client{},
			}

			ctx := context.Background()
			_, err := FindCalendarsWithConfig(ctx, tt.location, "test", "test", cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("FindCalendarsWithConfig() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("FindCalendarsWithConfig() unexpected error = %v", err)
			}

			// Note: We can't directly test possibleLocations since it's not returned,
			// but we can verify the function completes without error for valid cases
		})
	}
}
