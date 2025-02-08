package httpclient

import (
	"net/http"
	"net/url"
)

// HttpClientWrapper wraps http.Client with CalDAV-specific functionality
type HttpClientWrapper interface {
	DoPROPFIND(url string, depth int, props ...string) (*PropfindResponse, error)
	DoREPORT(url string, depth int, query interface{}) (*ReportResponse, error)
	DoPUT(url string, etag string, data []byte) (newEtag string, err error)
}

type httpClientWrapper struct {
	client  *http.Client
	baseURL url.URL
}

// NewHttpClientWrapper creates a new client wrapper with basic auth
func NewHttpClientWrapper(client *http.Client, baseURL url.URL) (HttpClientWrapper, error) {
	return &httpClientWrapper{client: client, baseURL: baseURL}, nil
}
