package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cyp0633/libcaldora/server"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

const (
	// Server configuration
	serverAddr   = ":8080"
	caldavPrefix = "/caldav/"
	serverRealm  = "libcaldora Example Server"
	maxDepth     = 3 // Maximum depth for PROPFIND/REPORT operations
)

func main() {
	// Initialize memory storage with sample data
	memStorage := setupStorage()

	// Create the CalDAV handler with our storage
	handler := server.NewCaldavHandler(caldavPrefix, serverRealm, memStorage, maxDepth, nil)

	// Register the handler with the HTTP server
	http.Handle(caldavPrefix, handler)

	http.HandleFunc("/.well-known/caldav", handler.ServeWellKnown)

	http.HandleFunc("/", handleRoot)

	// Start the HTTP server
	log.Printf("Starting CalDAV server on %s", serverAddr)
	log.Printf("CalDAV endpoint: http://localhost%s", serverAddr+caldavPrefix)
	log.Printf("Well-known CalDAV endpoint: http://localhost%s/.well-known/caldav", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// handleRoot provides a basic landing page with instructions
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>libcaldora Example Server</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        code { background: #f4f4f4; padding: 2px 4px; border-radius: 4px; }
        pre { background: #f4f4f4; padding: 10px; border-radius: 4px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>libcaldora Example CalDAV Server</h1>
    <p>This is a demonstration of the libcaldora CalDAV server.</p>
    
    <h2>Available Users</h2>
    <ul>
        <li><strong>Username:</strong> alice, <strong>Password:</strong> password</li>
        <li><strong>Username:</strong> bob, <strong>Password:</strong> password</li>
    </ul>
    
    <h2>CalDAV URL</h2>
    <p>The CalDAV server is available at: <code>http://localhost%s</code></p>
    
    <h2>Connect with a CalDAV Client</h2>
    <p>You can connect using any CalDAV client with these settings:</p>
    <ul>
        <li><strong>Server URL:</strong> <code>http://localhost%s</code></li>
        <li><strong>Username:</strong> alice or bob</li>
        <li><strong>Password:</strong> password</li>
    </ul>
</body>
</html>
`
	fmt.Fprintf(w, html, serverAddr+caldavPrefix, serverAddr+caldavPrefix)
}

// setupStorage initializes storage with sample users and calendars
func setupStorage() *MemoryStorage {
	memStorage := NewMemoryStorage()

	// Add users
	memStorage.RegisterUser("alice", "Alice Smith")
	memStorage.RegisterUser("bob", "Bob Johnson")

	// Create sample calendars for Alice
	createCalendarForUser(memStorage, "alice", "default", "Default", "#0000FF")
	createCalendarForUser(memStorage, "alice", "work", "Work", "#FF0000")

	// Create sample calendars for Bob
	createCalendarForUser(memStorage, "bob", "default", "Default", "#00FF00")
	createCalendarForUser(memStorage, "bob", "family", "Family", "#800080")

	// Create and add sample events for Alice's calendars
	now := time.Now()

	// Events for Alice's default calendar
	aliceEvent1 := createEvent("alice", "default", "Meeting with Team", "Conference Room A",
		now.Add(24*time.Hour), now.Add(25*time.Hour))
	aliceEvent2 := createEvent("alice", "default", "Doctor Appointment", "Medical Center",
		now.Add(48*time.Hour), now.Add(49*time.Hour))

	memStorage.AddEvent("alice", "default", aliceEvent1)
	memStorage.AddEvent("alice", "default", aliceEvent2)

	// Events for Alice's work calendar
	aliceWorkEvent1 := createEvent("alice", "work", "Project Review", "Office",
		now.Add(3*24*time.Hour), now.Add(3*24*time.Hour+2*time.Hour))
	aliceWorkEvent2 := createEvent("alice", "work", "Client Meeting", "Client HQ",
		now.Add(5*24*time.Hour), now.Add(5*24*time.Hour+3*time.Hour))

	memStorage.AddEvent("alice", "work", aliceWorkEvent1)
	memStorage.AddEvent("alice", "work", aliceWorkEvent2)

	// Events for Bob's calendars
	bobEvent1 := createEvent("bob", "default", "Grocery Shopping", "Supermarket",
		now.Add(6*time.Hour), now.Add(7*time.Hour))
	bobEvent2 := createEvent("bob", "default", "Gym", "Fitness Center",
		now.Add(30*time.Hour), now.Add(32*time.Hour))

	memStorage.AddEvent("bob", "default", bobEvent1)
	memStorage.AddEvent("bob", "default", bobEvent2)

	bobFamilyEvent1 := createEvent("bob", "family", "Family Dinner", "Home",
		now.Add(4*24*time.Hour), now.Add(4*24*time.Hour+3*time.Hour))
	bobFamilyEvent2 := createEvent("bob", "family", "Movie Night", "Cinema",
		now.Add(6*24*time.Hour), now.Add(6*24*time.Hour+4*time.Hour))

	memStorage.AddEvent("bob", "family", bobFamilyEvent1)
	memStorage.AddEvent("bob", "family", bobFamilyEvent2)

	return memStorage
}

// createCalendarForUser creates a calendar and adds it to storage
func createCalendarForUser(ms *MemoryStorage, userID, calendarID, name, color string) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//libcaldora//Example Server//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropName, name)
	cal.Props.SetText(ical.PropColor, color)

	calendar := &storage.Calendar{
		SupportedComponents: []string{"VEVENT"},
		ETag:                fmt.Sprintf("etag-calendar-%s-%s", name, uuid.New().String()[:8]),
		CTag:                fmt.Sprintf("ctag-%s-%d", name, time.Now().Unix()),
		CalendarData:        cal,
		Path:                fmt.Sprintf("/%s/cal/%s/", userID, calendarID),
	}

	// Add calendar to storage
	err := ms.CreateCalendar(userID, calendar)
	if err != nil {
		log.Printf("Error creating calendar %s for user %s: %v", calendarID, userID, err)
	}
}

// createEvent is a helper function to create a calendar event
func createEvent(userID, calendarID, summary, location string, start, end time.Time) storage.CalendarObject {
	eventUID := uuid.New().String()
	eventID := fmt.Sprintf("%s.ics", eventUID[:8])

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, eventUID)
	event.Props.SetText(ical.PropSummary, summary)
	event.Props.SetText(ical.PropLocation, location)
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	event.Props.SetDateTime(ical.PropDateTimeStart, start)
	event.Props.SetDateTime(ical.PropDateTimeEnd, end)

	return storage.CalendarObject{
		Path:      fmt.Sprintf("/%s/cal/%s/%s", userID, calendarID, eventID),
		ETag:      fmt.Sprintf("etag-%s-%d", eventUID[:8], time.Now().Unix()),
		Component: event.Component,
	}
}
