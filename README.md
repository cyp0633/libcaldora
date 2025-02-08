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

// Create a new client
client := davclient.NewDAVClient(httpClient, calendarURL)

// Get all events
filter := client.GetAllEvents()

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
