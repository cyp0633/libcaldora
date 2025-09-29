package httpclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type PropfindResponse struct {
	IsCalendar           bool
	DisplayName          string
	Color                string
	CurrentUserPrincipal string
	CalendarHomeSet      string
	CanWrite             bool
	Resources            map[string]ResourceProps
}

type ResourceProps struct {
	IsCalendar  bool
	DisplayName string
	Color       string
	CanWrite    bool
	Etag        string
}

type propfindXML struct {
	XMLName xml.Name `xml:"DAV: propfind"`
	Prop    propXML  `xml:"prop"`
}

type propXML struct {
	ResourceType         *xml.Name `xml:"DAV: resourcetype"`
	DisplayName          *xml.Name `xml:"DAV: displayname"`
	CalendarColor        *xml.Name `xml:"http://apple.com/ns/ical/ calendar-color"`
	CurrentUserPrivSet   *xml.Name `xml:"DAV: current-user-privilege-set"`
	CurrentUserPrincipal *xml.Name `xml:"DAV: current-user-principal"`
	CalendarHomeSet      *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set"`
	CalendarTimezone     *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-timezone"`
	SupportedComponents  *xml.Name `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set"`
	GetCTag              *xml.Name `xml:"http://calendarserver.org/ns/ getctag"`
	SyncToken            *xml.Name `xml:"DAV: sync-token"`
	ScheduleInbox        *xml.Name `xml:"urn:ietf:params:xml:ns:caldav schedule-inbox-URL"`
	ScheduleOutbox       *xml.Name `xml:"urn:ietf:params:xml:ns:caldav schedule-outbox-URL"`
	Getetag              *xml.Name `xml:"DAV: getetag"`
}

type responseXML struct {
	XMLName  xml.Name    `xml:"DAV: response"`
	Href     string      `xml:"href"`
	Propstat propstatXML `xml:"propstat"`
}

type propstatXML struct {
	Prop   propertyXML `xml:"prop"`
	Status string      `xml:"status"`
}

type propertyXML struct {
	ResourceType         resourceTypeXML `xml:"resourcetype"`
	DisplayName          string          `xml:"displayname"`
	CalendarColor        string          `xml:"calendar-color"`
	CurrentUserPrivSet   privSetXML      `xml:"current-user-privilege-set"`
	CurrentUserPrincipal string          `xml:"current-user-principal>href"`
	CalendarHomeSet      string          `xml:"calendar-home-set>href"`
	CalendarTimezone     string          `xml:"calendar-timezone"`
	SupportedComponents  componentSetXML `xml:"supported-calendar-component-set"`
	GetCTag              string          `xml:"getctag"`
	SyncToken            string          `xml:"sync-token"`
	ScheduleInbox        string          `xml:"schedule-inbox-URL>href"`
	ScheduleOutbox       string          `xml:"schedule-outbox-URL>href"`
	Getetag              string          `xml:"getetag"`
}

type resourceTypeXML struct {
	Calendar *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar"`
}

type privSetXML struct {
	Privilege []privilegeXML `xml:"privilege"`
}

type privilegeXML struct {
	names []xml.Name
}

func (p *privilegeXML) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	p.names = nil
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			p.names = append(p.names, t.Name)
			if err := dec.Skip(); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

var writePrivileges = map[string]struct{}{
	"all":                              {},
	"write":                            {},
	"write-content":                    {},
	"write-properties":                 {},
	"write-data":                       {},
	"write-collection":                 {},
	"write-acl":                        {},
	"write-current-user-privilege-set": {},
	"bind":                             {},
	"unbind":                           {},
}

func allowsWrite(priv privilegeXML) bool {
	for _, name := range priv.names {
		local := strings.ToLower(name.Local)
		if _, ok := writePrivileges[local]; ok {
			return true
		}
		if strings.Contains(local, "write") {
			return true
		}
	}
	return false
}

type componentSetXML struct {
	Comp []struct {
		Name string `xml:"name,attr"`
	} `xml:"comp"`
}

// DoPROPFIND performs a PROPFIND request
func (w *httpClientWrapper) DoPROPFIND(urlStr string, depth int, props ...string) (*PropfindResponse, error) {
	w.logger.Debug("starting PROPFIND request",
		"url", urlStr,
		"depth", depth,
		"properties", props)

	// Build PROPFIND request body
	body := buildPropfindXML(props...)

	resolvedURL, err := w.resolveURL(urlStr)
	if err != nil {
		w.logger.Debug("failed to resolve URL", "url", urlStr, "error", err)
		return nil, fmt.Errorf("failed to resolve URL %q: %w", urlStr, err)
	}
	w.logger.Debug("resolved URL", "url", resolvedURL.String())

	req, err := http.NewRequest("PROPFIND", resolvedURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Depth", fmt.Sprintf("%d", depth))
	req.Header.Set("Content-Type", "application/xml")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		w.logger.Debug("unexpected response status",
			"status_code", resp.StatusCode,
			"status", resp.Status)
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	w.logger.Debug("received multistatus response")

	// Parse response
	var result PropfindResponse
	result.Resources = make(map[string]ResourceProps)

	// Parse the multistatus response
	var multiStatus struct {
		XMLName  xml.Name      `xml:"DAV: multistatus"`
		Response []responseXML `xml:"response"`
	}

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&multiStatus); err != nil {
		w.logger.Debug("failed to parse XML response", "error", err)
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	w.logger.Debug("parsed XML response",
		"response_count", len(multiStatus.Response))

	// Process each response
	for _, resp := range multiStatus.Response {
		// Skip if not OK status
		if !strings.Contains(resp.Propstat.Status, "200") {
			continue
		}

		props := resp.Propstat.Prop

		// Set current-user-principal if found
		if props.CurrentUserPrincipal != "" {
			result.CurrentUserPrincipal = props.CurrentUserPrincipal
		}

		// Set calendar-home-set if found
		if props.CalendarHomeSet != "" {
			result.CalendarHomeSet = props.CalendarHomeSet
		}

		resource := ResourceProps{
			IsCalendar:  props.ResourceType.Calendar != nil,
			DisplayName: props.DisplayName,
			Color:       props.CalendarColor,
			CanWrite:    false,
			Etag:        props.Getetag,
		}

		// Check write permission
		for _, priv := range props.CurrentUserPrivSet.Privilege {
			if allowsWrite(priv) {
				resource.CanWrite = true
				break
			}
		}

		// Store in results map using href as key
		result.Resources[resp.Href] = resource
	}

	w.logger.Debug("PROPFIND request complete",
		"found_calendars", len(result.Resources),
		"principal_url", result.CurrentUserPrincipal != "",
		"home_set", result.CalendarHomeSet != "")
	return &result, nil
}

func buildPropfindXML(props ...string) []byte {
	propfind := propfindXML{
		Prop: propXML{},
	}

	// Add requested properties
	for _, prop := range props {
		switch prop {
		case "resourcetype":
			propfind.Prop.ResourceType = &xml.Name{Space: "DAV:", Local: "resourcetype"}
		case "displayname":
			propfind.Prop.DisplayName = &xml.Name{Space: "DAV:", Local: "displayname"}
		case "calendar-color":
			propfind.Prop.CalendarColor = &xml.Name{Space: "http://apple.com/ns/ical/", Local: "calendar-color"}
		case "current-user-privilege-set":
			propfind.Prop.CurrentUserPrivSet = &xml.Name{Space: "DAV:", Local: "current-user-privilege-set"}
		case "current-user-principal":
			propfind.Prop.CurrentUserPrincipal = &xml.Name{Space: "DAV:", Local: "current-user-principal"}
		case "calendar-home-set":
			propfind.Prop.CalendarHomeSet = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-home-set"}
		case "calendar-timezone":
			propfind.Prop.CalendarTimezone = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-timezone"}
		case "supported-calendar-component-set":
			propfind.Prop.SupportedComponents = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "supported-calendar-component-set"}
		case "getctag":
			propfind.Prop.GetCTag = &xml.Name{Space: "http://calendarserver.org/ns/", Local: "getctag"}
		case "sync-token":
			propfind.Prop.SyncToken = &xml.Name{Space: "DAV:", Local: "sync-token"}
		case "schedule-inbox-URL":
			propfind.Prop.ScheduleInbox = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "schedule-inbox-URL"}
		case "schedule-outbox-URL":
			propfind.Prop.ScheduleOutbox = &xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "schedule-outbox-URL"}
		case "getetag":
			propfind.Prop.Getetag = &xml.Name{Space: "DAV:", Local: "getetag"}
		}
	}

	// Marshal to XML
	body, err := xml.Marshal(propfind)
	if err != nil {
		return []byte{}
	}

	return body
}
