package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/davserver/server"
	"github.com/emersion/go-ical"
)

var testTime = time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

func newTestServer(t *testing.T) (*MemoryProvider, *httptest.Server) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	provider := NewMemoryProvider(logger)

	handler := server.New(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/calendar/",
		Logger:    logger,
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))

	return provider, ts
}

func TestServer(t *testing.T) {
	provider, ts := newTestServer(t)
	defer ts.Close()

	// Add test event
	cal := ical.NewCalendar()
	event := ical.NewEvent()
	cal.Children = append(cal.Children, event.Component)

	event.Props.SetText("SUMMARY", "Test Event")
	event.Props.SetDateTime("DTSTART", testTime)
	event.Props.SetDateTime("DTEND", testTime.Add(time.Hour))

	testObject := &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        "test.ics",
			Type:        interfaces.ResourceTypeCalendarObject,
			ContentType: ical.MIMEType,
			ETag:        "test-etag",
		},
		Data: cal,
	}

	err := provider.PutCalendarObject(context.Background(), "test.ics", testObject)
	if err != nil {
		t.Fatalf("Failed to put calendar object: %v", err)
	}

	// Test PROPFIND request
	req, err := http.NewRequest("PROPFIND", ts.URL+"/calendar/test.ics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("Expected status 207, got %d", resp.StatusCode)
	}

	// Test GET request
	resp, err = http.Get(ts.URL + "/calendar/test.ics")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !strings.Contains(string(data), "Test Event") {
		t.Error("Response does not contain event summary")
	}
}
