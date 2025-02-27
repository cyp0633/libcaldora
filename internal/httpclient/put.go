package httpclient

import (
	"bytes"
	"fmt"
	"net/http"
)

func (c *httpClientWrapper) DoPUT(urlStr string, etag string, data []byte) (newEtag string, err error) {
	c.logger.Debug("starting PUT request",
		"url", urlStr,
		"etag", etag,
		"data_length", len(data))

	resolvedURL, err := c.resolveURL(urlStr)
	if err != nil {
		c.logger.Debug("failed to resolve URL", "url", urlStr, "error", err)
		return "", fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}

	c.logger.Debug("resolved URL", "url", resolvedURL.String())

	req, err := http.NewRequest(http.MethodPut, resolvedURL.String(), bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if etag != "" {
		req.Header.Set("If-Match", etag)
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Debug("request failed", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	c.logger.Debug("received response", "status", resp.Status)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		c.logger.Debug("unexpected status code",
			"status_code", resp.StatusCode,
			"status", resp.Status)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	newEtag = resp.Header.Get("ETag")
	c.logger.Debug("PUT request complete",
		"status", resp.Status,
		"new_etag", newEtag)
	return newEtag, nil
}
