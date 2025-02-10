# libcaldora

A CalDAV client library in Go that supports automatic discovery and some essential calendar operations.

Server operations are planned.

## Features

- 🔍 Automatic CalDAV server discovery
  - Direct URL
  - DNS SRV records
  - Well-known URLs (/.well-known/caldav)
  - Root path fallback
- 📅 Complete calendar operations
  - List calendars
  - Get/Create/Update/Delete calendar events
  - Calendar synchronization support (Etag)
- 🔒 Authentication support
  - Basic authentication
  - Transport layer customization
- 🎨 Rich calendar information
  - Calendar name
  - Color
  - Access permissions

## Installation

```bash
go get github.com/cyp0633/libcaldora
```

## Usage

### Discovering Calendars

```go
import "github.com/cyp0633/libcaldora/davclient"

// Find calendars using automatic discovery
calendars, err := davclient.FindCalendars(context.Background(), "https://calendar.example.com", "username", "password")
if err != nil {
    log.Fatal(err)
}

// Print discovered calendars
for _, cal := range calendars {
    fmt.Printf("Calendar: %s (%s)\n", cal.Name, cal.URI)
    fmt.Printf("  Color: %s\n", cal.Color)
    fmt.Printf("  ReadOnly: %v\n", cal.ReadOnly)
}
```

### Calendar Operations

```go
import (
    "github.com/cyp0633/libcaldora/davclient"
    "github.com/emersion/go-ical"
)

// Create a client with basic auth
client, err := davclient.NewDAVClientWithBasicAuth("username", "password", "https://calendar.example.com")
if err != nil {
    log.Fatal(err)
}

// Or create with custom HTTP client
httpClient := &http.Client{
    Timeout: time.Second * 30,
}
client, err = davclient.NewDAVClient(httpClient, "https://calendar.example.com")
if err != nil {
    log.Fatal(err)
}

// Get all events
filter := client.GetAllEvents()

// Get calendar ETag for synchronization
etag, err := client.GetCalendarEtag()
if err != nil {
    log.Fatal(err)
}

// If calendar etag has changed, check object etags
etagFilter := client.GetObjectETags()

// Get specific objects by their URLs
urls := []string{"https://calendar.example.com/events/123", "https://calendar.example.com/events/456"}
objects, err := client.GetObjectsByURLs(urls)

// Create a new event
event := ical.NewEvent()
event.Props.SetText(ical.PropSummary, "Meeting")
objectURL, etag, err := client.CreateCalendarObject(calendarURL, event)

// Update an event
event.Props.SetText(ical.PropDescription, "Team meeting")
newEtag, err := client.UpdateCalendarObject(objectURL, event)

// Delete an event
err = client.DeleteCalendarObject(objectURL, etag)
```

### Event Filtering

The library supports rich filtering capabilities for retrieving calendar objects:

```go
// Time range filtering
filter := client.GetAllEvents().TimeRange(start, end)

// Combined filters
filter := client.GetAllEvents().
    TimeRange(start, end).
    Status("CONFIRMED").
    NotStatus("CANCELLED").
    Summary("Meeting").
    Location("Conference Room").
    Priority(1).
    Categories("Work", "Important").
    Limit(10)

// Execute filter
events, err := filter.Do()
if err != nil {
    log.Fatal(err)
}
```

## Advanced Configuration

### Custom DNS Resolver

```go
config := davclient.DefaultConfig()
config.Resolver = customResolver
calendars, err := davclient.FindCalendarsWithConfig(ctx, location, username, password, config)
```

### Custom HTTP Client

```go
config := davclient.DefaultConfig()
config.Client = &http.Client{
    Timeout: time.Second * 30,
}
calendars, err := davclient.FindCalendarsWithConfig(ctx, location, username, password, config)
```

## Thanks

- **Claude 3.5 Sonnet** on Copilot API for writing most of the project (including README)
- [**sabre.io Documentation**](https://sabre.io/dav/building-a-caldav-client/) for instructions on building a CalDAV client

## License

[MIT License](LICENSE)
