package httpclient

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// BasicAuthTransport implements http.RoundTripper and adds Basic Auth
// authentication to outgoing requests.
type BasicAuthTransport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
	Logger    *slog.Logger
}

// NewBasicAuthTransport creates a new BasicAuthTransport with the given
// credentials and optional underlying transport. If transport is nil,
// http.DefaultTransport will be used.
func NewBasicAuthTransport(username, password string, transport http.RoundTripper, logger *slog.Logger) *BasicAuthTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &BasicAuthTransport{
		Username:  username,
		Password:  password,
		Transport: transport,
		Logger:    logger,
	}
}

// RoundTrip implements the http.RoundTripper interface. It adds Basic Auth
// credentials to the request and delegates to the underlying transport.
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request details
	reqBody := ""
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			reqBody = string(bodyBytes)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset the body
		}
	}

	t.Logger.Debug("outgoing request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
		"body", reqBody)

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
	resp, err := t.Transport.RoundTrip(req)

	if err == nil && resp != nil {
		// Log response details
		respBody := ""
		if resp.Body != nil {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				respBody = string(bodyBytes)
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset the body
			}
		}

		t.Logger.Debug("incoming response",
			"status", resp.Status,
			"headers", resp.Header,
			"body", respBody)
	}

	return resp, err
}
