package davclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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

	// Verify the first calendar
	baseURLParsed, _ := url.Parse(baseURL)
	expectedURI1 := baseURLParsed.ResolveReference(&url.URL{Path: "/cyp0633/7f7d579c-cb19-047a-d5e5-c0894aaed9cd/"}).String()
	if calendars[0].URI != expectedURI1 {
		t.Errorf("Expected URI %s, got %s", expectedURI1, calendars[0].URI)
	}
	if calendars[0].Name != "7f7d579c-cb19-047a-d5e5-c0894aaed9cd" {
		t.Errorf("Expected name '7f7d579c-cb19-047a-d5e5-c0894aaed9cd', got '%s'", calendars[0].Name)
	}
	if calendars[0].Color != "" {
		t.Errorf("Expected no color, got '%s'", calendars[0].Color)
	}
	if calendars[0].ReadOnly {
		t.Error("Expected calendar to be writable")
	}

	// Verify the second calendar
	expectedURI2 := baseURLParsed.ResolveReference(&url.URL{Path: "/cyp0633/b860fa1c-fd49-82f9-d43b-8336bfd3a506/"}).String()
	if calendars[1].URI != expectedURI2 {
		t.Errorf("Expected URI %s, got %s", expectedURI2, calendars[1].URI)
	}
	if calendars[1].Name != "test2" {
		t.Errorf("Expected name 'test2', got '%s'", calendars[1].Name)
	}
	if calendars[1].Color != "#008080ff" {
		t.Errorf("Expected color '#008080ff', got '%s'", calendars[1].Color)
	}
	if calendars[1].ReadOnly {
		t.Error("Expected calendar to be writable")
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
