package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/davserver/handler"
	"github.com/cyp0633/libcaldora/davserver/interfaces"
	davxml "github.com/cyp0633/libcaldora/davserver/protocol/xml"
	"github.com/emersion/go-ical"
)

func setupTestServer() (*httptest.Server, *MemoryProvider) {
	// Send test logs to a buffer for checking
	testLog := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(io.MultiWriter(testLog, os.Stderr), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	provider := NewMemoryProvider(logger)

	// Add test calendar
	provider.calendars["/calendars/user1/calendar1"] = &interfaces.Calendar{
		Properties: &interfaces.ResourceProperties{
			Path:        "/calendars/user1/calendar1",
			Type:        interfaces.ResourceTypeCalendar,
			DisplayName: "Test Calendar",
			Color:       "#4A90E2",
		},
		TimeZone: "UTC",
	}

	// Create handler
	h := handler.NewDefaultHandler(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/calendars/",
		Logger:    logger,
	})

	return httptest.NewServer(h), provider
}

func TestPropfindCalendar(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	// Create PROPFIND request
	body := `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:resourcetype/>
    <D:displayname/>
    <C:calendar-color/>
  </D:prop>
</D:propfind>`

	req, err := http.NewRequest("PROPFIND", server.URL+"/calendars/user1/calendar1", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Depth", "0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("expected status 207, got %d", resp.StatusCode)
	}

	var ms davxml.MultistatusResponse
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		t.Fatal(err)
	}

	if len(ms.Response) != 1 {
		t.Errorf("expected 1 response, got %d", len(ms.Response))
	}

	response := ms.Response[0]
	if response.Propstat.Status != "HTTP/1.1 200 OK" {
		t.Errorf("expected OK status, got %s", response.Propstat.Status)
	}

	props := response.Propstat.Prop
	if props.DisplayName != "Test Calendar" {
		t.Errorf("expected displayname 'Test Calendar', got %s", props.DisplayName)
	}
	if props.CalendarColor != "#4A90E2" {
		t.Errorf("expected color '#4A90E2', got %s", props.CalendarColor)
	}
}

func TestPutCalendarObject(t *testing.T) {
	server, provider := setupTestServer()
	defer server.Close()

	// Create test event
	cal := &ical.Calendar{
		Component: ical.NewComponent(ical.CompCalendar),
	}
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//test//NONSGML v1.0//EN")

	// Create event component
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, "test-event-1")
	event.Props.SetText(ical.PropSummary, "Test Event")
	event.Props.SetText(ical.PropDateTimeStamp, time.Now().UTC().Format("20060102T150405Z"))

	// Add event to calendar
	cal.Component.Children = append(cal.Component.Children, event)

	// Encode event to iCalendar format
	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(cal); err != nil {
		t.Fatal(err)
	}

	// Send PUT request
	req, err := http.NewRequest("PUT", server.URL+"/calendars/user1/calendar1/test-event-1.ics", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// Verify event was stored
	obj, err := provider.GetCalendarObject(context.Background(), "/calendars/user1/calendar1/test-event-1.ics")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Properties.DisplayName != "Test Event" {
		t.Errorf("expected displayname 'Test Event', got %s", obj.Properties.DisplayName)
	}

	// Verify event content
	if summary, err := obj.Data.Component.Children[0].Props.Text(ical.PropSummary); err != nil {
		t.Fatal(err)
	} else if summary != "Test Event" {
		t.Errorf("expected summary 'Test Event', got %s", summary)
	}
}

func TestDeleteCalendarObject(t *testing.T) {
	server, provider := setupTestServer()
	defer server.Close()

	// Create test event for deletion
	cal := &ical.Calendar{
		Component: ical.NewComponent(ical.CompCalendar),
	}
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//test//NONSGML v1.0//EN")

	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, "test-event-1")
	event.Props.SetText(ical.PropSummary, "Test Event")
	event.Props.SetText(ical.PropDateTimeStamp, time.Now().UTC().Format("20060102T150405Z"))

	cal.Component.Children = append(cal.Component.Children, event)

	// Add event to provider
	eventPath := "/calendars/user1/calendar1/test-event-1.ics"
	provider.objects[eventPath] = &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        eventPath,
			Type:        interfaces.ResourceTypeCalendarObject,
			DisplayName: "Test Event",
		},
		Data: cal,
	}

	// Send DELETE request
	req, err := http.NewRequest("DELETE", server.URL+eventPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}

	// Verify event was deleted
	if _, err := provider.GetCalendarObject(context.Background(), eventPath); err == nil {
		t.Error("expected event to be deleted")
	}
}
