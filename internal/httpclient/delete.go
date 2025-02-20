package httpclient

import (
	"fmt"
	"net/http"
)

// DoDELETE sends a DELETE request with If-Match header for optimistic locking
func (c *httpClientWrapper) DoDELETE(urlStr string, etag string) error {
	resolvedURL, err := c.resolveURL(urlStr)
	if err != nil {
		return fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}

	req, err := http.NewRequest(http.MethodDelete, resolvedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}

	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE request failed with status %d", resp.StatusCode)
	}

	return nil
}
