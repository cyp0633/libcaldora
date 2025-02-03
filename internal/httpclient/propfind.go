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
}

type resourceTypeXML struct {
	Calendar *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar"`
}

type privSetXML struct {
	Privilege []privilegeXML `xml:"privilege"`
}

type privilegeXML struct {
	Write *xml.Name `xml:"write"`
}

type componentSetXML struct {
	Comp []struct {
		Name string `xml:"name,attr"`
	} `xml:"comp"`
}

// doPROPFIND performs a PROPFIND request
func (w *httpClientWrapper) DoPROPFIND(url string, depth int, props ...string) (*PropfindResponse, error) {
	// Build PROPFIND request body
	body := buildPropfindXML(props...)

	req, err := http.NewRequest("PROPFIND", url, bytes.NewReader(body))
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
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse response
	var result PropfindResponse
	result.Resources = make(map[string]ResourceProps)

	decoder := xml.NewDecoder(resp.Body)

	var responses []responseXML
	err = decoder.Decode(&responses)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	// Process each response
	for _, resp := range responses {
		// Skip if not OK status
		if !strings.Contains(resp.Propstat.Status, "200") {
			continue
		}

		props := resp.Propstat.Prop

		resource := ResourceProps{
			IsCalendar:  props.ResourceType.Calendar != nil,
			DisplayName: props.DisplayName,
			Color:       props.CalendarColor,
			CanWrite:    false,
		}

		// Check write permission
		for _, priv := range props.CurrentUserPrivSet.Privilege {
			if priv.Write != nil {
				resource.CanWrite = true
				break
			}
		}

		// Store in results map using href as key
		result.Resources[resp.Href] = resource
	}

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
		}
	}

	// Marshal to XML
	body, err := xml.Marshal(propfind)
	if err != nil {
		return []byte{}
	}

	return body
}
