package davclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

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
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		Resolver: &net.Resolver{},
		Client:   http.DefaultClient,
	}
}

// find calendar list based on location, logic from thunderbird
func FindCalendars(ctx context.Context, location string, username string, password string) (calendars []CalendarInfo, err error) {
	return FindCalendarsWithConfig(ctx, location, username, password, DefaultConfig())
}

// FindCalendarsWithConfig allows injecting custom configuration for testing
func FindCalendarsWithConfig(ctx context.Context, location string, username string, password string, cfg *Config) ([]CalendarInfo, error) {
	calendars := make([]CalendarInfo, 0)
	baseURL, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
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
		_, addrs, err := cfg.Resolver.LookupSRV(ctx, "", "", host)
		if err != nil {
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
		}
	}

	// 3. well-known URL
	wellKnownURL := baseURL.JoinPath(".well-known", "caldav")
	possibleLocations = append(possibleLocations, wellKnownURL.String())

	// 4. root path
	rootURL := baseURL.JoinPath("/")
	possibleLocations = append(possibleLocations, rootURL.String())

	for _, possibleLocation := range possibleLocations {
		client := cfg.Client
		if client == nil {
			client = &http.Client{}
		}
		client.Transport = &httpclient.BasicAuthTransport{
			Username:  username,
			Password:  password,
			Transport: http.DefaultTransport,
		}
		wrapper, err := httpclient.NewHttpClientWrapper(client, *baseURL)
		if err != nil {
			return nil, err
		}
		_, err = wrapper.DoPROPFIND(possibleLocation, 1,
			"resourcetype",
			"displayname",
			"calendar-color",
			"current-user-privilege-set")
		if err != nil {
			continue // Try next location if this one fails
		}

		// TODO: Parse PROPFIND response and append to calendars slice
		// For now, just add a placeholder calendar for testing
		calendars = append(calendars, CalendarInfo{
			URI:      possibleLocation,
			Name:     "Test Calendar",
			Color:    "#000000",
			ReadOnly: false,
		})
	}
	return calendars, nil
}
