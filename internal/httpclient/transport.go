package httpclient

import (
	"errors"
	"net/http"
)

// BasicAuthTransport implements http.RoundTripper and adds Basic Auth
// authentication to outgoing requests.
type BasicAuthTransport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
}

// NewBasicAuthTransport creates a new BasicAuthTransport with the given
// credentials and optional underlying transport. If transport is nil,
// http.DefaultTransport will be used.
func NewBasicAuthTransport(username, password string, transport http.RoundTripper) *BasicAuthTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &BasicAuthTransport{
		Username:  username,
		Password:  password,
		Transport: transport,
	}
}

// RoundTrip implements the http.RoundTripper interface. It adds Basic Auth
// credentials to the request and delegates to the underlying transport.
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Username == "" {
		return nil, errors.New("basic auth username cannot be empty")
	}
	if t.Password == "" {
		return nil, errors.New("basic auth password cannot be empty")
	}
	if t.Transport == nil {
		return nil, errors.New("transport cannot be nil")
	}
	req.SetBasicAuth(t.Username, t.Password)
	return t.Transport.RoundTrip(req)
}
