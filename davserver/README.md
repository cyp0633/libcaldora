# libcaldora CalDAV Server

This package provides a framework for building CalDAV servers in Go. It's designed to be flexible and extensible, allowing you to use any HTTP server framework and storage backend.

## Features

- Framework-agnostic HTTP handling
- Flexible storage interface
- Support for essential CalDAV operations:
  - Calendar discovery
  - Event retrieval/creation/modification/deletion
  - Calendar querying and filtering
  - iCalendar format handling
- Standard CalDAV property support
- Error handling with CalDAV-compliant responses
- Structured logging with slog

## Quick Start

Here's a minimal example using the built-in HTTP server and in-memory storage:

```go
package main

import (
    "log/slog"
    "net/http"
    "os"
    
    "github.com/cyp0633/libcaldora/davserver/handler"
    "github.com/cyp0633/libcaldora/davserver/interfaces"
)

func main() {
    // Setup logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    
    // Create a storage provider
    provider := NewMemoryProvider(logger)
    
    // Create the CalDAV handler
    h := handler.NewDefaultHandler(interfaces.HandlerConfig{
        Provider:  provider,
        URLPrefix: "/calendars/",
        Logger:    logger,
    })
    
    // Start the server
    log.Fatal(http.ListenAndServe(":8080", h))
}
```

## Logging

The server uses Go's standard `log/slog` package for structured logging. You can configure logging at any level:

```go
// Debug-level logging to stdout
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
    AddSource: true,
}))

// JSON format logging to a file
file, _ := os.Create("caldav.log")
logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// Use your own slog.Handler implementation
logger := slog.New(customHandler)
```

Log levels:
- DEBUG: XML parsing details, request processing steps
- INFO: Successful operations, request summary
- WARN: Non-critical issues (e.g., invalid request format)
- ERROR: Operation failures, data errors

## Integration with Other Frameworks

### Chi Router Example

```go
package main

import (
    "log/slog"
    "net/http"
    "os"
    
    "github.com/go-chi/chi/v5"
    "github.com/cyp0633/libcaldora/davserver/handler"
    "github.com/cyp0633/libcaldora/davserver/interfaces"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    r := chi.NewRouter()
    
    provider := NewMemoryProvider(logger)
    h := handler.NewDefaultHandler(interfaces.HandlerConfig{
        Provider:  provider,
        URLPrefix: "/calendars/",
        Logger:    logger,
    })
    
    r.Mount("/calendars", h)
    http.ListenAndServe(":8080", r)
}
```

### Gin Framework Example

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/gin-gonic/gin"
    "github.com/cyp0633/libcaldora/davserver/handler"
    "github.com/cyp0633/libcaldora/davserver/interfaces"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    r := gin.Default()
    
    provider := NewMemoryProvider(logger)
    h := handler.NewDefaultHandler(interfaces.HandlerConfig{
        Provider:  provider,
        URLPrefix: "/calendars/",
        Logger:    logger,
    })
    
    r.Any("/calendars/*path", gin.WrapH(h))
    r.Run(":8080")
}
```

## Implementing a Storage Provider

Create a custom storage provider by implementing the `interfaces.CalendarProvider` interface:

```go
type CalendarProvider interface {
    // Get properties for a resource at the given path
    GetResourceProperties(ctx context.Context, path string) (*ResourceProperties, error)
    
    // Get calendar information
    GetCalendar(ctx context.Context, path string) (*Calendar, error)
    
    // Calendar object operations
    GetCalendarObject(ctx context.Context, path string) (*CalendarObject, error)
    ListCalendarObjects(ctx context.Context, path string) ([]CalendarObject, error)
    PutCalendarObject(ctx context.Context, path string, object *CalendarObject) error
    DeleteCalendarObject(ctx context.Context, path string) error
    
    // Optional optimized query methods
    Query(ctx context.Context, calendarPath string, filter *QueryFilter) ([]CalendarObject, error)
    MultiGet(ctx context.Context, paths []string) ([]CalendarObject, error)
}
```

See `examples/memory/main.go` for a complete example of a memory-based storage provider with logging.

## Customizing the Handler

The default handler can be customized through the `HandlerConfig`:

```go
type HandlerConfig struct {
    // Storage provider implementation
    Provider CalendarProvider
    
    // Base path where the CalDAV server is mounted
    URLPrefix string
    
    // List of allowed HTTP methods (optional)
    AllowedMethods []string
    
    // Custom response headers (optional)
    CustomHeaders map[string]string
    
    // Logger for request/response logging (optional)
    Logger *slog.Logger
}
```

## Error Handling

The server uses standard HTTP status codes and returns CalDAV-compliant error responses:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<D:error xmlns:D="DAV:">
    <C:error xmlns:C="urn:ietf:params:xml:ns:caldav">
        <C:calendar-collection-location-ok/>
    </C:error>
</D:error>
```

All errors are logged with appropriate context and stack traces when available.

## Testing

The package includes integration tests that verify CalDAV protocol compliance. Run them with:

```bash
go test -v ./...
```

Test logs are written to stderr by default but can be redirected:

```go
logger := slog.New(slog.NewTextHandler(testLogFile, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
