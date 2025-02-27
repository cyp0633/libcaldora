package httpclient

import (
	"fmt"
	"net/http"
)

// DoDELETE sends a DELETE request with If-Match header for optimistic locking
func (c *httpClientWrapper) DoDELETE(urlStr string, etag string) error {
	c.logger.Debug("starting DELETE request",
		"url", urlStr,
		"etag", etag)

	resolvedURL, err := c.resolveURL(urlStr)
	if err != nil {
		c.logger.Debug("failed to resolve URL", "url", urlStr, "error", err)
		return fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}

	c.logger.Debug("resolved URL", "url", resolvedURL.String())

	req, err := http.NewRequest(http.MethodDelete, resolvedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}

	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Debug("request failed", "error", err)
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received response", "status", resp.Status)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		c.logger.Debug("unexpected status code",
			"status_code", resp.StatusCode,
			"status", resp.Status)
		return fmt.Errorf("DELETE request failed with status %d", resp.StatusCode)
	}

	c.logger.Debug("DELETE request complete", "status", resp.Status)
	return nil
}
