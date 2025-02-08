package davclient

import (
	"bytes"
	"fmt"
	"path"

	"github.com/emersion/go-ical"
	"github.com/google/uuid"
)

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
	// Generate a UUID for the new object
	id := uuid.New().String()
	objectURL = path.Join(collectionURL, id+".ics")

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

	props, ok := resp.Resources[objectURL]
	if !ok {
		return "", fmt.Errorf("object not found at %s", objectURL)
	}

	// Try to update with current etag
	data, err := eventToBytes(event)
	if err != nil {
		return "", fmt.Errorf("failed to encode calendar object: %w", err)
	}
	etag, err = c.httpClient.DoPUT(objectURL, props.Etag, data)
	if err != nil {
		return "", fmt.Errorf("failed to update calendar object: %w", err)
	}

	// If no etag in response, get it again
	if etag == "" {
		resp, err = c.httpClient.DoPROPFIND(objectURL, 0, "getetag")
		if err != nil {
			return "", fmt.Errorf("failed to get new etag: %w", err)
		}

		props, ok = resp.Resources[objectURL]
		if !ok || props.Etag == "" {
			return "", fmt.Errorf("no etag found for updated object")
		}
		etag = props.Etag
	}

	return etag, nil
}
