package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/davserver/server"
	"github.com/emersion/go-ical"
)

var testTime = time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

func newTestServer(t *testing.T) (*MemoryProvider, *httptest.Server) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	provider := NewMemoryProvider(logger)

	handler := server.New(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/", // Use root as prefix since we want to handle both /principals and /calendar
		Logger:    logger,
	})

	ts := httptest.NewServer(handler)

	return provider, ts
}

func TestCalendarDiscovery(t *testing.T) {
	provider, ts := newTestServer(t)
	defer ts.Close()

	// Test current-user-principal discovery
	principal, err := provider.GetCurrentUserPrincipal(context.Background())
	if err != nil {
		t.Fatalf("Failed to get current user principal: %v", err)
	}
	if principal != "/principals/user/" {
		t.Errorf("Expected principal path /principals/user/, got %s", principal)
	}

	// Test calendar home set discovery
	calendarHome, err := provider.GetCalendarHomeSet(context.Background(), principal)
	if err != nil {
		t.Fatalf("Failed to get calendar home set: %v", err)
	}
	if calendarHome != "/calendar/" {
		t.Errorf("Expected calendar home /calendar/, got %s", calendarHome)
	}

	// Test PROPFIND on principal URL
	req, err := http.NewRequest("PROPFIND", ts.URL+"/principals/user/", nil)
	req.Header.Set("Depth", "1")
	if err != nil {
		t.Fatalf("Failed to create principal request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute principal request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("Expected status 207 for principal request, got %d", resp.StatusCode)
	}

	// Test PROPFIND on calendar home URL
	req, err = http.NewRequest("PROPFIND", ts.URL+"/calendar/", nil)
	req.Header.Set("Depth", "1")
	if err != nil {
		t.Fatalf("Failed to create calendar home request: %v", err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute calendar home request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("Expected status 207 for calendar home request, got %d", resp.StatusCode)
	}

	// Test resource properties for principal URL
	props, err := provider.GetResourceProperties(context.Background(), "/principals/user/")
	if err != nil {
		t.Fatalf("Failed to get principal properties: %v", err)
	}
	if props.PrincipalURL != "/principals/user/" {
		t.Errorf("Expected principal URL /principals/user/, got %s", props.PrincipalURL)
	}
	if props.CalendarHomeURL != "/calendar/" {
		t.Errorf("Expected calendar home URL /calendar/, got %s", props.CalendarHomeURL)
	}
}

func TestCalendarOperations(t *testing.T) {
	provider, ts := newTestServer(t)
	defer ts.Close()

	// Add test event
	cal := ical.NewCalendar()
	cal.Props.SetText("PRODID", "-//libcaldora//NONSGML v1.0//EN")
	cal.Props.SetText("VERSION", "2.0")

	event := ical.NewEvent()
	event.Props.SetText("SUMMARY", "Test Event")
	event.Props.SetDateTime("DTSTART", testTime)
	event.Props.SetDateTime("DTEND", testTime.Add(time.Hour))
	event.Props.SetDateTime("DTSTAMP", time.Now()) // Required by iCalendar spec
	event.Props.SetText("UID", "test-event")       // Required by iCalendar spec
	cal.Children = append(cal.Children, event.Component)

	testObject := &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        "/calendar/test.ics", // Full path including calendar home
			Type:        interfaces.ResourceTypeCalendarObject,
			ContentType: ical.MIMEType,
			ETag:        "test-etag",
		},
		Data: cal,
	}

	err := provider.PutCalendarObject(context.Background(), "/calendar/test.ics", testObject)
	if err != nil {
		t.Fatalf("Failed to put calendar object: %v", err)
	}

	// Test PROPFIND request
	req, err := http.NewRequest("PROPFIND", ts.URL+"/calendar/test.ics", nil)
	req.Header.Set("Depth", "0") // Individual resource
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
