package httpclient

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// LevelTrace is a custom log level for very verbose HTTP wire logging.
// It is lower (more verbose) than slog.LevelDebug.
const LevelTrace = slog.Level(-8)

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
	start := time.Now()

	// Log request details
	isTrace := t.Logger != nil && t.Logger.Enabled(req.Context(), LevelTrace)
	if isTrace {
		reqBody := ""
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				// Cap request body size to avoid huge logs
				if len(bodyBytes) > 8192 {
					reqBody = string(bodyBytes[:8192]) + "...<truncated>"
				} else {
					reqBody = string(bodyBytes)
				}
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset the body
			}
		}
		// Full wire log at TRACE
		t.Logger.Log(req.Context(), LevelTrace, "outgoing request",
			"method", req.Method,
			"url", req.URL.String(),
			"headers", redactHeaders(req.Header),
			"body", reqBody)
	} else {
		// Concise at DEBUG
		t.Logger.Debug("outgoing request",
			"method", req.Method,
			"url", req.URL.String())
	}

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

	duration := time.Since(start)

	if err != nil {
		t.Logger.Error("request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"duration_ms", duration.Milliseconds(),
			"error", err)
		return resp, err
	}

	if resp != nil {
		if isTrace {
			// Log response details at TRACE
			respBody := ""
			if resp.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err == nil {
					if len(bodyBytes) > 8192 {
						respBody = string(bodyBytes[:8192]) + "...<truncated>"
					} else {
						respBody = string(bodyBytes)
					}
					resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset the body
				}
			}

			t.Logger.Log(req.Context(), LevelTrace, "incoming response",
				"status", resp.Status,
				"status_code", resp.StatusCode,
				"headers", redactHeaders(resp.Header),
				"duration_ms", duration.Milliseconds(),
				"body", respBody)
		} else {
			// Concise at DEBUG
			t.Logger.Debug("incoming response",
				"status", resp.Status,
				"status_code", resp.StatusCode,
				"duration_ms", duration.Milliseconds())
		}
	}

	return resp, nil
}

// redactHeaders returns a shallow-copied header map with sensitive headers redacted.
func redactHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	// Shallow copy
	out := make(http.Header, len(h))
	for k, vv := range h {
		// Normalize header name compare
		switch http.CanonicalHeaderKey(k) {
		case "Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie":
			out[k] = []string{"<redacted>"}
		default:
			cp := make([]string, len(vv))
			copy(cp, vv)
			out[k] = cp
		}
	}
	return out
}
