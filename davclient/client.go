package davclient

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/cyp0633/libcaldora/internal/httpclient"
	"github.com/emersion/go-ical"
)

// Options configures the DAV client
type Options struct {
	// Client is the http.Client to use for requests. If nil, http.DefaultClient is used.
	Client *http.Client
	// CalendarURL is the URL of the CalDAV calendar
	CalendarURL string
	// Username and Password are used for basic auth if provided
	Username string
	Password string
	// Logger is the slog.Logger to use for logging. If nil, logging is disabled
	Logger *slog.Logger
}

// DAVClient interface defines the CalDAV client operations
type DAVClient interface {
	GetAllEvents() ObjectFilter
	GetObjectETags() ObjectFilter
	GetObjectsByURLs(urls []string) ([]CalendarObject, error)
	GetCalendarEtag() (string, error)
	CreateCalendarObject(collectionURL string, event *ical.Event) (objectURL string, etag string, err error)
	UpdateCalendarObject(objectURL string, event *ical.Event) (etag string, err error)
	DeleteCalendarObject(objectURL string, etag string) error
}

type davClient struct {
	httpClient  httpclient.HttpClientWrapper
	calendarURL string
	logger      *slog.Logger
}

// NewDAVClient creates a new CalDAV client with options
func NewDAVClient(opts Options) (DAVClient, error) {
	baseURL, err := url.Parse(opts.CalendarURL)
	if err != nil {
		return nil, err
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives:   true, // Disable keep-alives to prevent connection drops
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 1,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
			},
		}
	}

	if opts.Username != "" && opts.Password != "" {
		transport := client.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		client = &http.Client{
			Transport: httpclient.NewBasicAuthTransport(opts.Username, opts.Password, transport, logger),
		}
	}

	wrapper, err := httpclient.NewHttpClientWrapper(client, *baseURL, logger)
	if err != nil {
		return nil, err
	}

	logger.Debug("creating new DAV client", "calendar_url", opts.CalendarURL)
	return &davClient{
		httpClient:  wrapper,
		calendarURL: opts.CalendarURL,
		logger:      logger,
	}, nil
}
