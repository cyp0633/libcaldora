package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/davserver/server"
	"github.com/emersion/go-ical"
)

// MemoryProvider implements CalendarProvider interface using in-memory storage
type MemoryProvider struct {
	mu      sync.RWMutex
	objects map[string]*interfaces.CalendarObject
	logger  *slog.Logger
}

func NewMemoryProvider(logger *slog.Logger) *MemoryProvider {
	return &MemoryProvider{
		objects: make(map[string]*interfaces.CalendarObject),
		logger:  logger,
	}
}

func (p *MemoryProvider) GetResourceProperties(ctx context.Context, path string) (*interfaces.ResourceProperties, error) {
	if path == "" {
		return &interfaces.ResourceProperties{
			Type:                interfaces.ResourceTypeCalendar,
			DisplayName:         "Test Calendar",
			Color:               "#4f6bed",
			SupportedComponents: []string{"VEVENT"},
		}, nil
	}

	p.mu.RLock()
	obj, ok := p.objects[path]
	p.mu.RUnlock()

	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return obj.Properties, nil
}

func (p *MemoryProvider) GetCalendar(ctx context.Context, path string) (*interfaces.Calendar, error) {
	if path != "" {
		return nil, interfaces.ErrNotFound
	}
	return &interfaces.Calendar{
		Properties: &interfaces.ResourceProperties{
			Type:                interfaces.ResourceTypeCalendar,
			DisplayName:         "Test Calendar",
			Color:               "#4f6bed",
			SupportedComponents: []string{"VEVENT"},
		},
	}, nil
}

func (p *MemoryProvider) GetCalendarObject(ctx context.Context, path string) (*interfaces.CalendarObject, error) {
	p.mu.RLock()
	obj, ok := p.objects[path]
	p.mu.RUnlock()

	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return obj, nil
}

func (p *MemoryProvider) ListCalendarObjects(ctx context.Context, path string) ([]interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	for _, obj := range p.objects {
		objects = append(objects, *obj) // Objects in storage already have ETags
	}
	return objects, nil
}

// generateETag generates a new ETag based on calendar data and timestamp
func (p *MemoryProvider) generateETag(object *interfaces.CalendarObject) string {
	if object.Data == nil {
		return ""
	}

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(object.Data); err != nil {
		p.logger.Error("failed to generate ETag", "error", err)
		return ""
	}
	// Include LastModified in hash to ensure uniqueness even for identical content
	timestamp := object.Properties.LastModified.UnixNano()
	buf.Write(fmt.Appendf(nil, "-%d", timestamp))

	hash := sha256.Sum256(buf.Bytes())
	return `"` + base64.URLEncoding.EncodeToString(hash[:]) + `"`
}

func (p *MemoryProvider) PutCalendarObject(ctx context.Context, path string, object *interfaces.CalendarObject) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	object.Properties.LastModified = time.Now()
	object.Properties.ETag = p.generateETag(object)
	p.objects[path] = object
	return nil
}

func (p *MemoryProvider) DeleteCalendarObject(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.objects, path)
	return nil
}

func (p *MemoryProvider) Query(ctx context.Context, calendarPath string, filter *interfaces.QueryFilter) ([]interfaces.CalendarObject, error) {
	p.logger.Debug("executing query",
		"calendar_path", calendarPath,
		"filter", filter)

	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []interfaces.CalendarObject
	for _, obj := range p.objects {
		// Simple filtering based on component type
		for _, comp := range obj.Data.Children {
			if comp.Name == filter.CompFilter {
				result = append(result, *obj) // Objects in storage already have ETags
				break
			}
		}
	}
	return result, nil
}

func (p *MemoryProvider) MultiGet(ctx context.Context, paths []string) ([]interfaces.CalendarObject, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	for _, path := range paths {
		if obj, ok := p.objects[path]; ok {
			objects = append(objects, *obj) // Objects in storage already have ETags
		}
	}
	return objects, nil
}

func main() {
	// Set up logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create provider
	provider := NewMemoryProvider(logger)

	// Add a test event
	cal := ical.NewCalendar()
	event := ical.NewEvent()
	cal.Children = append(cal.Children, event.Component)

	event.Props.SetText("SUMMARY", "Test Event")
	event.Props.SetDateTime("DTSTART", time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
	event.Props.SetDateTime("DTEND", time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC))

	testObject := &interfaces.CalendarObject{
		Properties: &interfaces.ResourceProperties{
			Path:        "test.ics",
			Type:        interfaces.ResourceTypeCalendarObject,
			ContentType: ical.MIMEType,
			ETag:        "test-etag",
		},
		Data: cal,
	}
	provider.PutCalendarObject(context.Background(), "test.ics", testObject)

	// Create server
	handler := server.New(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/calendar/",
		Logger:    logger,
	})

	// Start server
	logger.Info("starting server on :8080")
	http.Handle("/calendar/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
