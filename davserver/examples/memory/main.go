package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cyp0633/libcaldora/davserver/handler"
	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/emersion/go-ical"
)

// MemoryProvider implements the CalendarProvider interface using in-memory storage
type MemoryProvider struct {
	mu        sync.RWMutex
	calendars map[string]*interfaces.Calendar
	objects   map[string]*interfaces.CalendarObject
	logger    *slog.Logger
}

func NewMemoryProvider(logger *slog.Logger) *MemoryProvider {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	return &MemoryProvider{
		calendars: make(map[string]*interfaces.Calendar),
		objects:   make(map[string]*interfaces.CalendarObject),
		logger:    logger,
	}
}

func (p *MemoryProvider) GetResourceProperties(ctx context.Context, path string) (*interfaces.ResourceProperties, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if cal, ok := p.calendars[path]; ok {
		p.logger.Debug("retrieved calendar properties", "path", path)
		return cal.Properties, nil
	}
	if obj, ok := p.objects[path]; ok {
		p.logger.Debug("retrieved object properties", "path", path)
		return obj.Properties, nil
	}
	p.logger.Debug("resource not found", "path", path)
	return nil, os.ErrNotExist
}

func (p *MemoryProvider) GetCalendar(ctx context.Context, path string) (*interfaces.Calendar, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if cal, ok := p.calendars[path]; ok {
		p.logger.Debug("retrieved calendar", "path", path)
		return cal, nil
	}
	p.logger.Debug("calendar not found", "path", path)
	return nil, os.ErrNotExist
}

func (p *MemoryProvider) GetCalendarObject(ctx context.Context, path string) (*interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if obj, ok := p.objects[path]; ok {
		p.logger.Debug("retrieved calendar object", "path", path)
		return obj, nil
	}
	p.logger.Debug("calendar object not found", "path", path)
	return nil, os.ErrNotExist
}

func (p *MemoryProvider) ListCalendarObjects(ctx context.Context, path string) ([]interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	for _, obj := range p.objects {
		if obj.Properties.Path == path {
			objects = append(objects, *obj)
		}
	}
	p.logger.Debug("listed calendar objects", "path", path, "count", len(objects))
	return objects, nil
}

func (p *MemoryProvider) PutCalendarObject(ctx context.Context, path string, obj *interfaces.CalendarObject) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.objects[path] = obj
	p.logger.Info("stored calendar object", "path", path)
	return nil
}

func (p *MemoryProvider) DeleteCalendarObject(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.objects, path)
	p.logger.Info("deleted calendar object", "path", path)
	return nil
}

func (p *MemoryProvider) Query(ctx context.Context, calendarPath string, filter *interfaces.QueryFilter) ([]interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	for _, obj := range p.objects {
		if filter.CompFilter != "" {
			// TODO: Implement component filtering
		}
		objects = append(objects, *obj)
	}
	p.logger.Debug("queried calendar objects",
		"path", calendarPath,
		"filter", filter.CompFilter,
		"results", len(objects))
	return objects, nil
}

func (p *MemoryProvider) MultiGet(ctx context.Context, paths []string) ([]interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	for _, path := range paths {
		if obj, ok := p.objects[path]; ok {
			objects = append(objects, *obj)
		}
	}
	p.logger.Debug("multi-get calendar objects", "paths", len(paths), "found", len(objects))
	return objects, nil
}

func main() {
	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	// Create a memory provider and add a test calendar
	provider := NewMemoryProvider(logger)
	provider.calendars["/calendars/user1/calendar1"] = &interfaces.Calendar{
		Properties: &interfaces.ResourceProperties{
			Path:        "/calendars/user1/calendar1",
			Type:        interfaces.ResourceTypeCalendar,
			DisplayName: "Test Calendar",
			Color:       "#4A90E2",
		},
		TimeZone: "UTC",
	}

	// Create calendar handler with the memory provider
	h := handler.NewDefaultHandler(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/calendars/",
		CustomHeaders: map[string]string{
			"Server": "libcaldora/0.1.0",
		},
		Logger: logger,
	})

	// Create and add a test event
	cal := &ical.Calendar{
		Component: ical.NewComponent(ical.CompCalendar),
	}
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//libcaldora//NONSGML v1.0//EN")

	// Create event component
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropSummary, "Test Event")
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	event.Props.SetText(ical.PropUID, "test-event-1")

	// Add event to calendar
	cal.Component.Children = append(cal.Component.Children, event)

	provider.objects["/calendars/user1/calendar1/test-event-1.ics"] = &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        "/calendars/user1/calendar1/test-event-1.ics",
			Type:        interfaces.ResourceTypeCalendarObject,
			DisplayName: "Test Event",
			ContentType: "text/calendar; charset=utf-8",
			ETag:        "\"1\"",
		},
		Data: cal,
	}

	// Start HTTP server
	addr := ":8080"
	logger.Info("starting CalDAV server", "address", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
