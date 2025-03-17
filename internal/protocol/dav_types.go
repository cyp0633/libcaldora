package protocol

import (
	"encoding/xml"
)

// PROPFIND Request Types

type PropfindRequest struct {
	XMLName  xml.Name  `xml:"DAV: propfind"`
	Props    *Props    `xml:"prop,omitempty"`
	AllProp  *struct{} `xml:"allprop,omitempty"`
	PropName *struct{} `xml:"propname,omitempty"`
}

type Props struct {
	ResourceType         *xml.Name `xml:"DAV: resourcetype,omitempty"`
	DisplayName          *xml.Name `xml:"DAV: displayname,omitempty"`
	CalendarColor        *xml.Name `xml:"http://apple.com/ns/ical/ calendar-color,omitempty"`
	CurrentUserPrivSet   *xml.Name `xml:"DAV: current-user-privilege-set,omitempty"`
	SupportedComponents  *xml.Name `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set,omitempty"`
	GetCTag              *xml.Name `xml:"http://calendarserver.org/ns/ getctag,omitempty"`
	GetETag              *xml.Name `xml:"DAV: getetag,omitempty"`
	CalendarData         *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
	CurrentUserPrincipal *xml.Name `xml:"DAV: current-user-principal,omitempty"`
	CalendarHomeSet      *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set,omitempty"`
}

// PROPFIND Response Types

type MultistatusResponse struct {
	XMLName  xml.Name   `xml:"DAV: multistatus"`
	Response []Response `xml:"response"`
}

type Response struct {
	XMLName  xml.Name  `xml:"DAV: response"`
	Href     string    `xml:"href"`
	Propstat *Propstat `xml:"propstat"`
}

type Propstat struct {
	XMLName xml.Name    `xml:"DAV: propstat"`
	Prop    PropertySet `xml:"prop"`
	Status  string      `xml:"status"`
}

type PropertySet struct {
	ResourceType         *ResourceType         `xml:"DAV: resourcetype,omitempty"`
	DisplayName          string                `xml:"DAV: displayname,omitempty"`
	CalendarColor        string                `xml:"http://apple.com/ns/ical/ calendar-color,omitempty"`
	CurrentUserPrivSet   *CurrentUserPrivSet   `xml:"DAV: current-user-privilege-set,omitempty"`
	SupportedComponents  *SupportedComponents  `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set,omitempty"`
	GetCTag              string                `xml:"http://calendarserver.org/ns/ getctag,omitempty"`
	GetETag              string                `xml:"DAV: getetag,omitempty"`
	CalendarData         string                `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
	CurrentUserPrincipal *CurrentUserPrincipal `xml:"DAV: current-user-principal,omitempty"`
	CalendarHomeSet      *CalendarHomeSet      `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set,omitempty"`
}

type ResourceType struct {
	Collection *xml.Name `xml:"DAV: collection,omitempty"`
	Calendar   *xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar,omitempty"`
}

type CurrentUserPrivSet struct {
	Privilege []Privilege `xml:"privilege"`
}

type Privilege struct {
	Read         *xml.Name `xml:"DAV: read,omitempty"`
	Write        *xml.Name `xml:"DAV: write,omitempty"`
	WriteContent *xml.Name `xml:"DAV: write-content,omitempty"`
	Bind         *xml.Name `xml:"DAV: bind,omitempty"`
	Unbind       *xml.Name `xml:"DAV: unbind,omitempty"`
}

type SupportedComponents struct {
	Comp []Component `xml:"comp"`
}

type Component struct {
	Name string `xml:"name,attr"`
}

type CurrentUserPrincipal struct {
	Href string `xml:"href"`
}

type CalendarHomeSet struct {
	Href string `xml:"href"`
}

// REPORT Request Types

type CalendarQueryReport struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Props   Props    `xml:"DAV: prop"`
	Filter  Filter   `xml:"filter"`
}

type Filter struct {
	XMLName    xml.Name    `xml:"urn:ietf:params:xml:ns:caldav filter"`
	CompFilter *CompFilter `xml:"comp-filter"`
}

type CompFilter struct {
	XMLName   xml.Name   `xml:"comp-filter"`
	Name      string     `xml:"name,attr"`
	Test      string     `xml:"test,attr,omitempty"`
	TimeRange *TimeRange `xml:"time-range,omitempty"`
}

type TimeRange struct {
	XMLName xml.Name `xml:"time-range"`
	Start   string   `xml:"start,attr,omitempty"`
	End     string   `xml:"end,attr,omitempty"`
}

type Owner struct {
	XMLName xml.Name `xml:"DAV: owner"`
	Href    string   `xml:"href"`
}

// MultiGet Request Types

type CalendarMultiGet struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-multiget"`
	Props   Props    `xml:"DAV: prop"`
	Href    []string `xml:"DAV: href"`
}

// Utility functions for creating common responses

func NewOKResponse(href string, props PropertySet) Response {
	return Response{
		Href: href,
		Propstat: &Propstat{
			Prop:   props,
			Status: "HTTP/1.1 200 OK",
		},
	}
}

func NewErrorResponse(href string, status string) Response {
	return Response{
		Href: href,
		Propstat: &Propstat{
			Status: status,
		},
	}
}

func NewMultistatusResponse(responses ...Response) *MultistatusResponse {
	return &MultistatusResponse{
		Response: responses,
	}
}
