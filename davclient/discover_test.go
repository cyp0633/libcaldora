package davclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
)

type mockTransport struct{}

type dummyResolver struct{}

func (d *dummyResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	return "", nil, fmt.Errorf("dummy: no SRV records")
}

func (d *dummyResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return []string{}, nil
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create mock response based on the request URL and method
	var respBody string
	switch req.URL.Path {
	case "/":
		// Always return the current-user-principal response regardless of Depth header
		respBody = `<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/</href>
    <propstat>
      <prop>
        <current-user-principal>
          <href>/cyp0633/</href>
        </current-user-principal>
        <resourcetype>
          <collection/>
        </resourcetype>
        <owner/>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`
	case "/cyp0633/", "/cyp0633":
		if req.Header.Get("Depth") == "0" {
			respBody = `<?xml version='1.0' encoding='utf-8'?>
<multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<response>
<href>/cyp0633/</href>
<propstat>
<prop>
<C:calendar-home-set>
<href>/cyp0633/</href>
</C:calendar-home-set>
</prop>
<status>HTTP/1.1 200 OK</status>
</propstat>
</response>
</multistatus>`
		} else if req.Header.Get("Depth") == "1" {
			respBody = `<multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"
xmlns:ICAL="http://apple.com/ns/ical/">
<response>
<href>/cyp0633/</href>
<propstat>
<prop>
<resourcetype>
<principal />
<collection />
</resourcetype>
<current-user-privilege-set>
<privilege>
<read />
</privilege>
<privilege>
<all />
</privilege>
<privilege>
<write />
</privilege>
</current-user-privilege-set>
</prop>
<status>HTTP/1.1 200 OK</status>
</propstat>
</response>
<response>
<href>/cyp0633/7f7d579c-cb19-047a-d5e5-c0894aaed9cd/</href>
<propstat>
<prop>
<resourcetype>
<C:calendar />
<collection />
</resourcetype>
<displayname>7f7d579c-cb19-047a-d5e5-c0894aaed9cd</displayname>
<current-user-privilege-set>
<privilege>
<read />
</privilege>
<privilege>
<all />
</privilege>
<privilege>
<write />
</privilege>
</current-user-privilege-set>
</prop>
<status>HTTP/1.1 200 OK</status>
</propstat>
</response>
<response>
<href>/cyp0633/b860fa1c-fd49-82f9-d43b-8336bfd3a506/</href>
<propstat>
<prop>
<resourcetype>
<C:calendar />
<collection />
</resourcetype>
<displayname>test2</displayname>
<ICAL:calendar-color>#008080ff</ICAL:calendar-color>
<current-user-privilege-set>
<privilege>
<read />
</privilege>
<privilege>
<all />
</privilege>
<privilege>
<write />
</privilege>
</current-user-privilege-set>
</prop>
<status>HTTP/1.1 200 OK</status>
</propstat>
</response>
</multistatus>`
		}
	}

	if respBody == "" {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
		}, nil
	}

	resp := &http.Response{
		StatusCode: http.StatusMultiStatus,
		Body:       io.NopCloser(bytes.NewBufferString(respBody)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/xml")
	return resp, nil
}

func TestFindCalendars(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockTransport{},
	}

	baseURL := "http://example.com"
	cfg := &Config{
		Client:   mockClient,
		Resolver: &dummyResolver{},
	}

	calendars, err := FindCalendarsWithConfig(context.Background(), baseURL, "testuser", "testpass", cfg)
	if err != nil {
		t.Fatalf("FindCalendars failed: %v", err)
	}

	// Verify the number of calendars found
	if len(calendars) != 2 {
		t.Errorf("Expected 2 calendars, got %d", len(calendars))
	}

	// Expected relative URIs for both calendars
	calendar1URI := "/cyp0633/7f7d579c-cb19-047a-d5e5-c0894aaed9cd/"
	calendar2URI := "/cyp0633/b860fa1c-fd49-82f9-d43b-8336bfd3a506/"

	// Find both calendars
	var calendar1, calendar2 *CalendarInfo
	for _, cal := range calendars {
		if cal.URI == calendar1URI {
			calendar1 = &cal
		} else if cal.URI == calendar2URI {
			calendar2 = &cal
		}
	}

	// Verify first calendar
	if calendar1 == nil {
		t.Errorf("Calendar with URI %s not found", calendar1URI)
	} else {
		if calendar1.Name != "7f7d579c-cb19-047a-d5e5-c0894aaed9cd" {
			t.Errorf("Expected name '7f7d579c-cb19-047a-d5e5-c0894aaed9cd', got '%s'", calendar1.Name)
		}
		if calendar1.Color != "" {
			t.Errorf("Expected no color, got '%s'", calendar1.Color)
		}
		if calendar1.ReadOnly {
			t.Error("Expected calendar to be writable")
		}
	}

	// Verify second calendar
	if calendar2 == nil {
		t.Errorf("Calendar with URI %s not found", calendar2URI)
	} else {
		if calendar2.Name != "test2" {
			t.Errorf("Expected name 'test2', got '%s'", calendar2.Name)
		}
		if calendar2.Color != "#008080ff" {
			t.Errorf("Expected color '#008080ff', got '%s'", calendar2.Color)
		}
		if calendar2.ReadOnly {
			t.Error("Expected calendar to be writable")
		}
	}
}

func TestFindCalendarsManual(t *testing.T) {
	t.Skip("Manual test - run with credentials set in environment variables")

	username := os.Getenv("CALDAV_USERNAME")
	password := os.Getenv("CALDAV_PASSWORD")
	if username == "" || password == "" {
		t.Fatal("CALDAV_USERNAME and CALDAV_PASSWORD environment variables must be set")
	}

	ctx := context.Background()
	location := "https://caldav.icloud.com.cn"

	// Enable debug logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(logHandler)

	cfg := &Config{
		Client: http.DefaultClient,
		Logger: logger,
	}

	calendars, err := FindCalendarsWithConfig(ctx, location, username, password, cfg)
	if err != nil {
		t.Fatalf("FindCalendars failed: %v", err)
	}

	t.Logf("Found %d calendars:", len(calendars))
	for i, cal := range calendars {
		t.Logf("Calendar %d:", i+1)
		t.Logf("  URI: %s", cal.URI)
		t.Logf("  Name: %s", cal.Name)
		t.Logf("  Color: %s", cal.Color)
		t.Logf("  ReadOnly: %v", cal.ReadOnly)
	}
}

func TestFindCalendarsInvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		location string
	}{
		{"Empty URL", ""},
		{"Invalid URL", "not-a-url"},
		{"Missing scheme", "example.com"},
		{"Invalid scheme", "ftp://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FindCalendars(context.Background(), tt.location, "user", "pass")
			if err == nil {
				t.Error("Expected error for invalid URL, got nil")
			}
		})
	}
}
