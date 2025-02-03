package httpclient

import "net/http"

type BasicAuthTransport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
}

func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return t.Transport.RoundTrip(req)
}
