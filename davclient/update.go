package davclient // UpdateCalendarObject updates a calendar object at the specified URL with optimistic locking using etags
import "fmt"

func (c *davClient) UpdateCalendarObject(objectURL string, data []byte) (etag string, err error) {
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
