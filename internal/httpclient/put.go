package httpclient

import (
	"bytes"
	"fmt"
	"net/http"
)

func (c *httpClientWrapper) DoPUT(urlStr string, etag string, data []byte) (newEtag string, err error) {
	resolvedURL, err := c.resolveURL(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}

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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Header.Get("ETag"), nil
}
