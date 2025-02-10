package davclient

import (
	"net/http"
	"net/url"

	"github.com/cyp0633/libcaldora/internal/httpclient"
	"github.com/emersion/go-ical"
)

// DAVClient interface defines the CalDAV client operations
type DAVClient interface {
	GetAllEvents() ObjectFilter
	GetCalendarEtag() (string, error)
	CreateCalendarObject(collectionURL string, event *ical.Event) (objectURL string, etag string, err error)
	UpdateCalendarObject(objectURL string, event *ical.Event) (etag string, err error)
	DeleteCalendarObject(objectURL string, etag string) error
}

type davClient struct {
	httpClient  httpclient.HttpClientWrapper
	calendarURL string
}

// NewDAVClient creates a new CalDAV client with the given http.Client and calendar URL
func NewDAVClient(client *http.Client, calendarURL string) (DAVClient, error) {
	baseURL, err := url.Parse(calendarURL)
	if err != nil {
		return nil, err
	}

	wrapper, err := httpclient.NewHttpClientWrapper(client, *baseURL)
	if err != nil {
		return nil, err
	}

	return &davClient{
		httpClient:  wrapper,
		calendarURL: calendarURL,
	}, nil
}

// NewDAVClientWithBasicAuth creates a new CalDAV client with basic auth credentials
func NewDAVClientWithBasicAuth(username, password, calendarURL string) (DAVClient, error) {
	client := &http.Client{
		Transport: httpclient.NewBasicAuthTransport(username, password, nil),
	}
	return NewDAVClient(client, calendarURL)
}
