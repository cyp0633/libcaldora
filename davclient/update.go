package davclient

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

// normalizeURLPath extracts the path from a URL for comparison
func normalizeURLPath(urlStr string) string {
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		if parsed, err := url.Parse(urlStr); err == nil {
			return parsed.Path
		}
	}
	return urlStr
}

// eventToBytes converts an ical.Event to iCalendar format bytes
func eventToBytes(event *ical.Event) ([]byte, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText("PRODID", "-//github.com/cyp0633/libcaldora//NONSGML v1.0//EN")
	cal.Props.SetText("VERSION", "2.0")
	cal.Children = append(cal.Children, event.Component)

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(cal); err != nil {
		return nil, fmt.Errorf("failed to encode calendar: %w", err)
	}
	return buf.Bytes(), nil
}

// CreateCalendarObject creates a new calendar object in the specified collection URL
// Returns the URL of the created object and its etag
func (c *davClient) CreateCalendarObject(collectionURL string, event *ical.Event) (objectURL string, etag string, err error) {
	// Generate a UUID for the new object and construct URL correctly
	id := uuid.New().String()
	base, err := url.Parse(collectionURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse collection URL: %w", err)
	}
	ref, err := url.Parse(id + ".ics")
	if err != nil {
		return "", "", fmt.Errorf("failed to parse object URL: %w", err)
	}
	objectURL = base.ResolveReference(ref).String()

	// Create the object without an etag (new object)
	data, err := eventToBytes(event)
	if err != nil {
		return "", "", fmt.Errorf("failed to encode calendar object: %w", err)
	}
	etag, err = c.httpClient.DoPUT(objectURL, "", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to create calendar object: %w", err)
	}

	// If no etag in response, get it again
	if etag == "" {
		resp, err := c.httpClient.DoPROPFIND(objectURL, 0, "getetag")
		if err != nil {
			return objectURL, "", fmt.Errorf("failed to get new etag: %w", err)
		}

		props, ok := resp.Resources[objectURL]
		if !ok || props.Etag == "" {
			return objectURL, "", fmt.Errorf("no etag found for created object")
		}
		etag = props.Etag
	}

	return objectURL, etag, nil
}

// UpdateCalendarObject updates a calendar object at the specified URL with optimistic locking using etags
func (c *davClient) UpdateCalendarObject(objectURL string, event *ical.Event) (etag string, err error) {
	// First try to get the current etag
	resp, err := c.httpClient.DoPROPFIND(objectURL, 0, "getetag")
	if err != nil {
		return "", fmt.Errorf("failed to get object etag: %w", err)
	}

	// Normalize the object URL for comparison
	normalizedObjectURL := normalizeURLPath(objectURL)

	// Look for the object in the response (might be returned with different URL format)
	var objectEtag string
	var found bool

	for respURL, props := range resp.Resources {
		normalizedRespURL := normalizeURLPath(respURL)
		if normalizedRespURL == normalizedObjectURL {
			objectEtag = props.Etag
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("object not found at %s", objectURL)
	}

	// Try to update with current etag
	data, err := eventToBytes(event)
	if err != nil {
		return "", fmt.Errorf("failed to encode calendar object: %w", err)
	}
	etag, err = c.httpClient.DoPUT(objectURL, objectEtag, data)
	if err != nil {
		return "", fmt.Errorf("failed to update calendar object: %w", err)
	}

	// If no etag in response, get it again
	if etag == "" {
		resp, err = c.httpClient.DoPROPFIND(objectURL, 0, "getetag")
		if err != nil {
			return "", fmt.Errorf("failed to get new etag: %w", err)
		}

		// Look for the object again with normalized URL comparison
		for respURL, props := range resp.Resources {
			normalizedRespURL := normalizeURLPath(respURL)
			if normalizedRespURL == normalizedObjectURL && props.Etag != "" {
				etag = props.Etag
				break
			}
		}

		if etag == "" {
			return "", fmt.Errorf("no etag found for updated object")
		}
	}

	return etag, nil
}

// DeleteCalendarObject deletes a calendar object at the specified URL with optimistic locking using etag
func (c *davClient) DeleteCalendarObject(objectURL string, etag string) error {
	if err := c.httpClient.DoDELETE(objectURL, etag); err != nil {
		return fmt.Errorf("failed to delete calendar object: %w", err)
	}
	return nil
}
