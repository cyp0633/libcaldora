package davclient

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

// normalizeURL extracts the path from a URL for comparison
func normalizeURL(urlStr string) string {
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		if parsed, err := url.Parse(urlStr); err == nil {
			return parsed.Path
		}
	}
	return urlStr
}

// TestRealServerOperations tests the client against real CalDAV servers
// Set these environment variables to run:
// - CALDAV_SERVER_URL (e.g., "https://caldav.fastmail.com", "https://caldav.icloud.com")
// - CALDAV_USERNAME
// - CALDAV_PASSWORD
// - CALDAV_CALENDAR_URL (optional, will auto-discover if not set)
func TestRealServerOperations(t *testing.T) {
	serverURL := os.Getenv("CALDAV_SERVER_URL")
	username := os.Getenv("CALDAV_USERNAME")
	password := os.Getenv("CALDAV_PASSWORD")
	calendarURL := os.Getenv("CALDAV_CALENDAR_URL")

	if serverURL == "" || username == "" || password == "" {
		t.Skip("Real server test requires CALDAV_SERVER_URL, CALDAV_USERNAME, and CALDAV_PASSWORD environment variables")
	}

	ctx := context.Background()

	// Setup debug logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("Discovery", func(t *testing.T) {
		t.Logf("Testing calendar discovery against %s", serverURL)

		// Create a custom HTTP client with better connection handling
		customClient := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives:   true, // Disable keep-alives to prevent connection drops
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 1,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
			},
		}

		calendars, err := FindCalendarsWithConfig(ctx, serverURL, username, password, &Config{
			Logger:   logger,
			Resolver: &net.Resolver{},
			Client:   customClient,
		})
		if err != nil {
			t.Fatalf("Calendar discovery failed: %v", err)
		}

		if len(calendars) == 0 {
			t.Fatal("No calendars found")
		}

		t.Logf("Found %d calendars:", len(calendars))
		for i, cal := range calendars {
			t.Logf("  Calendar %d: %s (%s)", i+1, cal.Name, cal.URI)
			t.Logf("    Color: %s", cal.Color)
			t.Logf("    ReadOnly: %v", cal.ReadOnly)
		}

		// Use first writable calendar if no specific URL provided
		if calendarURL == "" {
			for _, cal := range calendars {
				if !cal.ReadOnly {
					// Construct absolute URL if we got a relative one
					if strings.HasPrefix(cal.URI, "/") {
						calendarURL = serverURL + cal.URI
					} else {
						calendarURL = cal.URI
					}
					break
				}
			}
		}

		if calendarURL == "" {
			t.Fatal("No writable calendar found")
		}
		t.Logf("Using calendar: %s", calendarURL)
	})

	if calendarURL == "" {
		t.Fatal("No calendar URL available for testing")
	}

	// Create client for CRUD operations
	client, err := NewDAVClient(Options{
		Username:    username,
		Password:    password,
		CalendarURL: calendarURL,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("ETag_Operations", func(t *testing.T) {
		// Test calendar ETag
		etag, err := client.GetCalendarEtag()
		if err != nil {
			t.Errorf("Failed to get calendar ETag: %v", err)
		} else {
			t.Logf("Calendar ETag: %s", etag)
		}

		// Test object ETags
		etagFilter := client.GetObjectETags()
		objects, err := etagFilter.Do()
		if err != nil {
			t.Errorf("Failed to get object ETags: %v", err)
		} else {
			t.Logf("Found %d objects with ETags", len(objects))
		}
	})

	var testEventURL string
	var testEventETag string

	t.Run("Create_Event", func(t *testing.T) {
		// Create a test event
		event := ical.NewEvent()
		eventID := uuid.New().String()
		event.Props.SetText("UID", eventID)
		event.Props.SetText("SUMMARY", "libcaldora Test Event")
		event.Props.SetText("DESCRIPTION", "Created by libcaldora integration test")

		// Add required DTSTAMP property
		event.Props.SetDateTime("DTSTAMP", time.Now().UTC())

		// Set time to tomorrow at 10 AM
		tomorrow := time.Now().Add(24 * time.Hour)
		startTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, time.UTC)
		endTime := startTime.Add(time.Hour)

		event.Props.SetDateTime("DTSTART", startTime)
		event.Props.SetDateTime("DTEND", endTime)

		objectURL, etag, err := client.CreateCalendarObject(calendarURL, event)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}

		testEventURL = objectURL
		testEventETag = etag

		t.Logf("Created event at: %s", objectURL)
		t.Logf("Event ETag: %s", etag)

		if testEventURL == "" {
			t.Error("Event URL is empty")
		}
		if testEventETag == "" {
			t.Error("Event ETag is empty")
		}
	})

	t.Run("Query_Events", func(t *testing.T) {
		// Test basic query
		filter := client.GetAllEvents()
		events, err := filter.Do()
		if err != nil {
			t.Fatalf("Failed to query events: %v", err)
		}

		t.Logf("Found %d total events", len(events))

		// Find our test event by normalizing URLs for comparison
		var foundTestEvent bool
		normalizedTestURL := normalizeURL(testEventURL)
		t.Logf("Looking for test event URL: %s (normalized: %s)", testEventURL, normalizedTestURL)

		for _, event := range events {
			normalizedEventURL := normalizeURL(event.URL)
			t.Logf("  Checking event URL: %s (normalized: %s)", event.URL, normalizedEventURL)
			if normalizedEventURL == normalizedTestURL {
				foundTestEvent = true
				t.Logf("Found our test event: %s", event.Event.Props.Get("SUMMARY"))
				break
			}
		}

		if !foundTestEvent {
			t.Error("Could not find the test event we just created")
		}
	})

	t.Run("Query_With_Filters", func(t *testing.T) {
		// Test time range query
		start := time.Now()
		end := start.Add(48 * time.Hour)

		filter := client.GetAllEvents().TimeRange(start, end)
		events, err := filter.Do()
		if err != nil {
			t.Errorf("Failed to query events with time range: %v", err)
		} else {
			t.Logf("Found %d events in next 48 hours", len(events))
		}

		// Test summary filter
		filter = client.GetAllEvents().Summary("libcaldora Test Event")
		events, err = filter.Do()
		if err != nil {
			t.Errorf("Failed to query events with summary filter: %v", err)
		} else {
			t.Logf("Found %d events matching summary filter", len(events))
		}
	})

	t.Run("Multiget_Operation", func(t *testing.T) {
		if testEventURL == "" {
			t.Skip("No test event URL available")
		}

		urls := []string{testEventURL}
		objects, err := client.GetObjectsByURLs(urls)
		if err != nil {
			t.Errorf("Failed to multiget objects: %v", err)
		} else {
			t.Logf("Retrieved %d objects via multiget", len(objects))
			if len(objects) > 0 {
				t.Logf("First object summary: %s", objects[0].Event.Props.Get("SUMMARY"))
			}
		}
	})

	t.Run("Update_Event", func(t *testing.T) {
		if testEventURL == "" {
			t.Skip("No test event URL available")
		}

		// Get the current event
		objects, err := client.GetObjectsByURLs([]string{testEventURL})
		if err != nil || len(objects) == 0 {
			t.Fatalf("Failed to get event for update: %v", err)
		}

		event := objects[0].Event
		event.Props.SetText("DESCRIPTION", "Updated by libcaldora integration test at "+time.Now().Format(time.RFC3339))

		newETag, err := client.UpdateCalendarObject(testEventURL, &event)
		if err != nil {
			t.Errorf("Failed to update event: %v", err)
		} else {
			t.Logf("Updated event, new ETag: %s", newETag)
			testEventETag = newETag
		}
	})

	t.Run("Delete_Event", func(t *testing.T) {
		if testEventURL == "" {
			t.Skip("No test event URL available")
		}

		err := client.DeleteCalendarObject(testEventURL, testEventETag)
		if err != nil {
			t.Errorf("Failed to delete event: %v", err)
		} else {
			t.Logf("Successfully deleted test event")
		}

		// Verify deletion by trying to retrieve the object
		objects, err := client.GetObjectsByURLs([]string{testEventURL})
		if err != nil {
			// An error here likely means the object was not found, which is expected after deletion
			t.Logf("Confirmed event deletion (expected error): %v", err)
		} else if len(objects) == 0 {
			t.Logf("Confirmed event deletion: no objects returned")
		} else {
			t.Error("Event still exists after deletion")
		}
	})
}
