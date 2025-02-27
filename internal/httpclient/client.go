package httpclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// HttpClientWrapper wraps http.Client with CalDAV-specific functionality
type HttpClientWrapper interface {
	DoPROPFIND(url string, depth int, props ...string) (*PropfindResponse, error)
	DoREPORT(url string, depth int, query interface{}) (*ReportResponse, error)
	DoPUT(url string, etag string, data []byte) (newEtag string, err error)
	DoDELETE(url string, etag string) error
}

type httpClientWrapper struct {
	client  *http.Client
	baseURL url.URL
	logger  *slog.Logger
}

// resolveURL resolves a URL string against the base URL
func (c *httpClientWrapper) resolveURL(urlStr string) (*url.URL, error) {
	ref, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %q: %w", urlStr, err)
	}
	return c.baseURL.ResolveReference(ref), nil
}

// NewHttpClientWrapper creates a new client wrapper with basic auth and logging
func NewHttpClientWrapper(client *http.Client, baseURL url.URL, logger *slog.Logger) (HttpClientWrapper, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &httpClientWrapper{client: client, baseURL: baseURL, logger: logger}, nil
}
