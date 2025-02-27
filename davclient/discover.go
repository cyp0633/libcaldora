package davclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/cyp0633/libcaldora/internal/httpclient"
)

type CalendarInfo struct {
	URI      string
	Name     string
	Color    string
	ReadOnly bool
}

// DNSResolver interface for mocking DNS lookups in tests
type DNSResolver interface {
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// Config holds configuration for FindCalendars
type Config struct {
	Resolver DNSResolver
	Client   *http.Client
	Logger   *slog.Logger
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		Resolver: &net.Resolver{},
		Client:   http.DefaultClient,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// find calendar list based on location, logic from thunderbird
func FindCalendars(ctx context.Context, location string, username string, password string) (calendars []CalendarInfo, err error) {
	return FindCalendarsWithConfig(ctx, location, username, password, DefaultConfig())
}

// FindCalendarsWithConfig allows injecting custom configuration for testing
func FindCalendarsWithConfig(ctx context.Context, location string, username string, password string, cfg *Config) ([]CalendarInfo, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	logger.Debug("starting calendar discovery", "location", location, "username", username)
	calendars := make([]CalendarInfo, 0)

	// Validate URL
	if location == "" {
		return nil, fmt.Errorf("invalid URL")
	}

	baseURL, err := url.Parse(location)
	if err != nil || baseURL.Host == "" || baseURL.Scheme == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid URL")
	}

	// Try all discovery methods
	possibleLocations := []string{}

	// 1. Try direct location if path is specified
	if baseURL.Path != "/" && baseURL.Path != "" {
		possibleLocations = append(possibleLocations, location)
	}

	// 2. DNS SRV
	// Try both secure and non-secure
	for _, prefix := range []string{"_caldavs._tcp.", "_caldav._tcp."} {
		host := prefix + baseURL.Hostname()
		logger.Debug("looking up SRV records", "host", host)
		_, addrs, err := cfg.Resolver.LookupSRV(ctx, "", "", host)
		if err != nil {
			logger.Debug("SRV lookup failed", "host", host, "error", err)
			continue
		}

		// Check for TXT records for path
		var path string
		txts, _ := cfg.Resolver.LookupTXT(ctx, host)
		for _, txt := range txts {
			if len(txt) > 5 && txt[:5] == "path=" {
				path = txt[5:]
				break
			}
		}

		// Construct URLs from SRV records
		for _, addr := range addrs {
			scheme := "http"
			if prefix == "_caldavs._tcp." {
				scheme = "https"
			}

			serverURL := fmt.Sprintf("%s://%s:%d%s",
				scheme,
				addr.Target,
				addr.Port,
				path,
			)
			possibleLocations = append(possibleLocations, serverURL)
			logger.Debug("added SRV location", "url", serverURL)
		}
	}

	// 3. well-known URL
	wellKnownURL := baseURL.JoinPath(".well-known", "caldav")
	possibleLocations = append(possibleLocations, wellKnownURL.String())
	logger.Debug("added well-known location", "url", wellKnownURL.String())

	// 4. root path
	rootURL := baseURL.JoinPath("/")
	possibleLocations = append(possibleLocations, rootURL.String())
	logger.Debug("added root location", "url", rootURL.String())

	// Set up the client once before the loop
	client := cfg.Client
	if client == nil {
		client = &http.Client{}
	}

	// Preserve existing transport if present
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = httpclient.NewBasicAuthTransport(username, password, transport)

	wrapper, err := httpclient.NewHttpClientWrapper(client, *baseURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client wrapper: %v", err)
	}

	// Try each possible location to find the principal URL
	var principalURL string
	for _, possibleLocation := range possibleLocations {
		logger.Debug("trying location for principal URL", "url", possibleLocation)
		// Get current-user-principal
		resp, err := wrapper.DoPROPFIND(possibleLocation, 0, "current-user-principal")
		if err != nil {
			logger.Debug("PROPFIND failed", "url", possibleLocation, "error", err)
			if strings.Contains(err.Error(), "network error") {
				return nil, fmt.Errorf("network error")
			}
			continue
		}

		if resp != nil && resp.CurrentUserPrincipal != "" {
			// Convert relative principal URL to absolute if needed
			if !strings.HasPrefix(resp.CurrentUserPrincipal, "http://") && !strings.HasPrefix(resp.CurrentUserPrincipal, "https://") {
				baseURL, _ := url.Parse(possibleLocation)
				relativeURL, _ := url.Parse(resp.CurrentUserPrincipal)
				principalURL = baseURL.ResolveReference(relativeURL).String()
			} else {
				principalURL = resp.CurrentUserPrincipal
			}
			logger.Debug("found principal URL", "url", principalURL)
			break
		}
	}

	if principalURL == "" {
		logger.Debug("no principal URL found")
		return nil, fmt.Errorf("could not find current-user-principal")
	}

	// Get calendar home from principal URL
	logger.Debug("fetching calendar home", "principal_url", principalURL)
	resp, err := wrapper.DoPROPFIND(principalURL, 0, "calendar-home-set")
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar-home-set: %v", err)
	}

	if resp == nil || resp.CalendarHomeSet == "" {
		logger.Debug("no calendar home found")
		return nil, fmt.Errorf("no calendar-home-set found")
	}

	// Convert relative calendar home URL to absolute if needed
	calendarHome := resp.CalendarHomeSet
	if !strings.HasPrefix(calendarHome, "http://") && !strings.HasPrefix(calendarHome, "https://") {
		baseURL, _ := url.Parse(principalURL)
		relativeURL, _ := url.Parse(calendarHome)
		calendarHome = baseURL.ResolveReference(relativeURL).String()
	}
	logger.Debug("found calendar home", "url", calendarHome)

	// List calendars from calendar home
	logger.Debug("fetching calendars", "calendar_home", calendarHome)
	resp, err = wrapper.DoPROPFIND(calendarHome, 1,
		"resourcetype",
		"displayname",
		"calendar-color",
		"current-user-privilege-set")
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %v", err)
	}

	if resp == nil || resp.Resources == nil {
		logger.Debug("no calendars found")
		return nil, fmt.Errorf("no calendars found")
	}

	// Process each resource in the response and collect calendars
	for uri, resource := range resp.Resources {
		if resource.IsCalendar {
			// Convert absolute URI to relative if needed
			calendarURI := uri
			if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
				// Parse the calendar home and current URI
				calendarHomeURL, _ := url.Parse(calendarHome)
				currentURL, _ := url.Parse(uri)

				// If they have the same host and scheme, make it relative
				if currentURL.Host == calendarHomeURL.Host && currentURL.Scheme == calendarHomeURL.Scheme {
					// Remove the common base path to get the relative path
					calendarURI = strings.TrimPrefix(currentURL.Path, calendarHomeURL.Path)
					if !strings.HasPrefix(calendarURI, "/") {
						calendarURI = "/" + calendarURI
					}
				}
			}

			// Only add if it's a calendar resource
			calendars = append(calendars, CalendarInfo{
				URI:      calendarURI,
				Name:     resource.DisplayName,
				Color:    resource.Color,
				ReadOnly: !resource.CanWrite,
			})
			logger.Debug("found calendar", "uri", calendarURI, "name", resource.DisplayName)
		}
	}

	// Sort calendars by URI for consistent ordering
	sort.Slice(calendars, func(i, j int) bool {
		return calendars[i].URI < calendars[j].URI
	})

	logger.Debug("calendar discovery complete", "count", len(calendars))
	return calendars, nil
}
