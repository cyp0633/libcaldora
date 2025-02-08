package davclient

import (
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

// NewDAVClient creates a new CalDAV client
func NewDAVClient(httpClient httpclient.HttpClientWrapper, calendarURL string) DAVClient {
	return &davClient{
		httpClient:  httpClient,
		calendarURL: calendarURL,
	}
}
