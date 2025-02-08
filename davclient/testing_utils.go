package davclient

import "github.com/cyp0633/libcaldora/internal/httpclient"

type mockPutResponse struct {
	etag string
	err  error
}

// PropfindFunc is a function type for mocking PROPFIND
type PropfindFunc func(url string, depth int, props ...string) (*httpclient.PropfindResponse, error)

// Mock types for testing
type mockHTTPClient struct {
	propfindResponse *httpclient.PropfindResponse
	reportResponse   *httpclient.ReportResponse
	putResponse      *mockPutResponse
	deleteResponse   error
	doPropfind       PropfindFunc
}

func (m *mockHTTPClient) DoPROPFIND(url string, depth int, props ...string) (*httpclient.PropfindResponse, error) {
	if m.doPropfind != nil {
		return m.doPropfind(url, depth, props...)
	}
	return m.propfindResponse, nil
}

func (m *mockHTTPClient) DoREPORT(url string, depth int, query interface{}) (*httpclient.ReportResponse, error) {
	return m.reportResponse, nil
}

func (m *mockHTTPClient) DoPUT(url string, etag string, data []byte) (string, error) {
	if m.putResponse != nil {
		return m.putResponse.etag, m.putResponse.err
	}
	return "new-etag", nil
}

func (m *mockHTTPClient) DoDELETE(url string, etag string) error {
	return m.deleteResponse
}
