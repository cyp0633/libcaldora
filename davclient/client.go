package davclient

import (
	"github.com/cyp0633/libcaldora/internal/httpclient"
)

// DAVClient interface defines the CalDAV client operations
type DAVClient interface {
	GetAllEvents() ObjectFilter
}

type davClient struct {
	httpClient  httpclient.HttpClientWrapper
	calendarURL string
}

// NewDAVClient creates a new CalDAV client
func NewDAVClient(httpClient httpclient.HttpClientWrapper, calendarURL string) DAVClient {
	return &davClient{
		httpClient:  httpClient,
		calendarURL: calendarURL,
	}
}
