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
	"strings"
	"sync"
	"time"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
	"github.com/cyp0633/libcaldora/davserver/server"
	"github.com/emersion/go-ical"
)

// MemoryProvider implements CalendarProvider interface using in-memory storage
type MemoryProvider struct {
	mu              sync.RWMutex
	objects         map[string]*interfaces.CalendarObject
	calendarVersion int64 // For CTag generation
	logger          *slog.Logger
	principalPath   string // Path to user's principal collection
	calendarHome    string // Path to user's calendar home
}

func NewMemoryProvider(logger *slog.Logger) *MemoryProvider {
	return &MemoryProvider{
		objects:         make(map[string]*interfaces.CalendarObject),
		calendarVersion: time.Now().UnixNano(), // Initialize with current timestamp
		logger:          logger,
		principalPath:   "/principals/user/", // Default principal path
		calendarHome:    "/calendar/",        // Default calendar home
	}
}

// generateCTag generates a new CTag based on calendar version and contents
func (p *MemoryProvider) generateCTag() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var pathHash bytes.Buffer
	for path := range p.objects {
		pathHash.WriteString(path)
	}
	hash := sha256.Sum256(pathHash.Bytes())
	return fmt.Sprintf("%d-%s", p.calendarVersion, base64.URLEncoding.EncodeToString(hash[:8]))
}

func (p *MemoryProvider) GetCurrentUserPrincipal(ctx context.Context) (string, error) {
	return p.principalPath, nil
}

func (p *MemoryProvider) GetCalendarHomeSet(ctx context.Context, principalPath string) (string, error) {
	if principalPath != p.principalPath {
		return "", interfaces.ErrNotFound
	}
	return p.calendarHome, nil
}

func (p *MemoryProvider) GetResourceProperties(ctx context.Context, path string) (*interfaces.ResourceProperties, error) {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	// Return principal properties
	if path == p.principalPath || strings.TrimSuffix(path, "/") == strings.TrimSuffix(p.principalPath, "/") {
		return &interfaces.ResourceProperties{
			Path:            p.principalPath,
			Type:            interfaces.ResourceTypeCollection,
			DisplayName:     "User Principal",
			PrincipalURL:    p.principalPath,
			CalendarHomeURL: p.calendarHome,
		}, nil
	}

	// Return calendar home properties
	if path == p.calendarHome || strings.TrimSuffix(path, "/") == strings.TrimSuffix(p.calendarHome, "/") {
		return &interfaces.ResourceProperties{
			Path:                p.calendarHome,
			Type:                interfaces.ResourceTypeCalendar,
			DisplayName:         "Test Calendar",
			Color:               "#4f6bed",
			SupportedComponents: []string{"VEVENT"},
			CTag:                p.generateCTag(),
			CurrentUserPrivSet:  []string{"read", "write"},
		}, nil
	}

	// Return calendar object properties if path is not empty
	if path != "" {
		p.mu.RLock()
		obj, ok := p.objects[path]
		p.mu.RUnlock()

		if !ok {
			return nil, interfaces.ErrNotFound
		}
		return obj.Properties, nil
	}

	return nil, interfaces.ErrNotFound
}

func (p *MemoryProvider) GetCalendar(ctx context.Context, path string) (*interfaces.Calendar, error) {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	if path == p.calendarHome || strings.TrimSuffix(path, "/") == strings.TrimSuffix(p.calendarHome, "/") || path == "" {
		return &interfaces.Calendar{
			Properties: &interfaces.ResourceProperties{
				Path:                p.calendarHome,
				Type:                interfaces.ResourceTypeCalendar,
				DisplayName:         "Test Calendar",
				Color:               "#4f6bed",
				SupportedComponents: []string{"VEVENT"},
				CTag:                p.generateCTag(),
				CurrentUserPrivSet:  []string{"read", "write"},
			},
		}, nil
	}
	return nil, interfaces.ErrNotFound
}

func (p *MemoryProvider) GetCalendarObject(ctx context.Context, path string) (*interfaces.CalendarObject, error) {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	p.mu.RLock()
	obj, ok := p.objects[path]
	p.mu.RUnlock()

	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return obj, nil
}

// ListResources implements the ListableProvider interface
func (p *MemoryProvider) ListResources(ctx context.Context, path string) ([]*interfaces.ResourceProperties, error) {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /

	// Handle principal path - list calendar home
	if path == p.principalPath || strings.TrimSuffix(path, "/") == strings.TrimSuffix(p.principalPath, "/") {
		return []*interfaces.ResourceProperties{
			{
				Path:                strings.TrimPrefix(p.calendarHome, "/"),
				Type:                interfaces.ResourceTypeCalendar,
				DisplayName:         "Test Calendar",
				Color:               "#4f6bed",
				SupportedComponents: []string{"VEVENT"},
				CTag:                p.generateCTag(),
				CurrentUserPrivSet:  []string{"read", "write"},
			},
		}, nil
	}

	// Handle calendar home path - list calendar objects
	if path == p.calendarHome || strings.TrimSuffix(path, "/") == strings.TrimSuffix(p.calendarHome, "/") {
		p.mu.RLock()
		defer p.mu.RUnlock()

		var resources []*interfaces.ResourceProperties
		for _, obj := range p.objects {
			resources = append(resources, obj.Properties)
		}
		return resources, nil
	}

	return nil, interfaces.ErrNotFound
}

func (p *MemoryProvider) ListCalendarObjects(ctx context.Context, path string) ([]interfaces.CalendarObject, error) {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	// Only list objects in the given calendar path
	p.mu.RLock()
	defer p.mu.RUnlock()

	var objects []interfaces.CalendarObject
	calendarPath := strings.TrimSuffix(p.calendarHome, "/") + "/"
	for objPath, obj := range p.objects {
		if strings.HasPrefix(objPath, calendarPath) {
			objects = append(objects, *obj) // Objects in storage already have ETags
		}
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
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	p.mu.Lock()
	defer p.mu.Unlock()

	object.Properties.LastModified = time.Now()
	object.Properties.ETag = p.generateETag(object)
	p.objects[path] = object

	// Update calendar version on any modification
	p.calendarVersion = time.Now().UnixNano()
	return nil
}

func (p *MemoryProvider) DeleteCalendarObject(ctx context.Context, path string) error {
	path = "/" + strings.TrimPrefix(path, "/") // Normalize path to start with /
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.objects, path)

	// Update calendar version on any modification
	p.calendarVersion = time.Now().UnixNano()
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
			Path:        "/calendar/test.ics", // Full path including calendar home
			Type:        interfaces.ResourceTypeCalendarObject,
			ContentType: ical.MIMEType,
			ETag:        "test-etag",
		},
		Data: cal,
	}
	provider.PutCalendarObject(context.Background(), "/calendar/test.ics", testObject)

	// Create server
	handler := server.New(interfaces.HandlerConfig{
		Provider:  provider,
		URLPrefix: "/", // Handle all paths since we need both /principals and /calendar
		Logger:    logger,
	})

	// Start server
	logger.Info("starting server on :8080")
	http.Handle("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
