package props

import (
	"html"
	"strconv"
	"time"

	"github.com/beevik/etree"
)

type CalendarDescription struct {
	Value string
}

func (p CalendarDescription) Encode() *etree.Element {
	elem := createElement("calendar-description")
	elem.SetText(p.Value)
	return elem
}

type CalendarTimezone struct {
	Value string
}

func (p CalendarTimezone) Encode() *etree.Element {
	elem := createElement("calendar-timezone")
	elem.SetText(p.Value)
	return elem
}

type CalendarData struct {
	// Note: the raw ICS data must contain BEGIN:VCALENDAR. It does not check for this.
	ICal string
}

func (p CalendarData) Encode() *etree.Element {
	elem := createElement("calendar-data")
	elem.SetText(html.EscapeString(p.ICal))
	return elem
}

type SupportedCalendarComponentSet struct {
	Components []string
}

func (p SupportedCalendarComponentSet) Encode() *etree.Element {
	elem := createElement("supported-calendar-component-set")

	for _, component := range p.Components {
		compElem := createElement("comp")
		compElem.CreateAttr("name", component)
		elem.AddChild(compElem)
	}

	return elem
}

type SupportedCalendarData struct {
	ContentType string
	Version     string
}

func (p SupportedCalendarData) Encode() *etree.Element {
	elem := createElement("supported-calendar-data")
	elem.SetText(p.ContentType)
	if p.Version != "" {
		elem.CreateAttr("version", p.Version)
	}
	return elem
}

type MaxResourceSize struct {
	Value int64
}

func (p MaxResourceSize) Encode() *etree.Element {
	elem := createElement("max-resource-size")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

type MinDateTime struct {
	Value time.Time
}

func (p MinDateTime) Encode() *etree.Element {
	elem := createElement("min-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

type MaxDateTime struct {
	Value time.Time
}

func (p MaxDateTime) Encode() *etree.Element {
	elem := createElement("max-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

type MaxInstances struct {
	Value int
}

func (p MaxInstances) Encode() *etree.Element {
	elem := createElement("max-instances")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type MaxAttendeesPerInstance struct {
	Value int
}

func (p MaxAttendeesPerInstance) Encode() *etree.Element {
	elem := createElement("max-attendees-per-instance")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type CalendarHomeSet struct {
	Href string
}

func (p CalendarHomeSet) Encode() *etree.Element {
	elem := createElement("calendar-home-set")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleInboxURL struct {
	Href string
}

func (p ScheduleInboxURL) Encode() *etree.Element {
	elem := createElement("schedule-inbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleOutboxURL struct {
	Href string
}

func (p ScheduleOutboxURL) Encode() *etree.Element {
	elem := createElement("schedule-outbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleDefaultCalendarURL struct {
	Href string
}

func (p ScheduleDefaultCalendarURL) Encode() *etree.Element {
	elem := createElement("schedule-default-calendar-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type CalendarUserAddressSet struct {
	Addresses []string
}

func (p CalendarUserAddressSet) Encode() *etree.Element {
	elem := createElement("calendar-user-address-set")

	for _, address := range p.Addresses {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(address)
	}

	return elem
}

type CalendarUserType struct {
	Value string
}

func (p CalendarUserType) Encode() *etree.Element {
	elem := createElement("calendar-user-type")
	elem.SetText(p.Value)
	return elem
}
