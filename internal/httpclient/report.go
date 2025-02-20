package httpclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
)

// DoREPORT executes a CalDAV REPORT request
func (c *httpClientWrapper) DoREPORT(urlStr string, depth int, query interface{}) (*ReportResponse, error) {
	// Marshal query to XML
	queryXML, err := xml.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal REPORT query: %w", err)
	}

	// Resolve URL
	resolvedURL, err := c.resolveURL(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}

	// Create request
	req, err := http.NewRequest("REPORT", resolvedURL.String(), bytes.NewReader(queryXML))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", fmt.Sprintf("%d", depth))

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var multiStatus ReportResponse
	if err := xml.NewDecoder(resp.Body).Decode(&multiStatus); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &multiStatus, nil
}

// ReportResponse represents a CalDAV REPORT response
type ReportResponse struct {
	XMLName   xml.Name `xml:"DAV: multistatus"`
	Responses []struct {
		Href     string `xml:"DAV: href"`
		PropStat struct {
			Prop struct {
				CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
				ETag         string `xml:"DAV: getetag"`
			} `xml:"DAV: prop"`
			Status string `xml:"DAV: status"`
		} `xml:"DAV: propstat"`
	} `xml:"DAV: response"`
}
