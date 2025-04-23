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

func (p *CalendarDescription) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type CalendarTimezone struct {
	Value string
}

func (p CalendarTimezone) Encode() *etree.Element {
	elem := createElement("calendar-timezone")
	elem.SetText(p.Value)
	return elem
}

func (p *CalendarTimezone) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
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

func (p *CalendarData) Decode(elem *etree.Element) error {
	p.ICal = html.UnescapeString(elem.Text())
	return nil
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

func (p *SupportedCalendarComponentSet) Decode(elem *etree.Element) error {
	p.Components = []string{}

	compElems := elem.FindElements("comp")
	for _, compElem := range compElems {
		if nameAttr := compElem.SelectAttr("name"); nameAttr != nil {
			p.Components = append(p.Components, nameAttr.Value)
		}
	}

	return nil
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

func (p *SupportedCalendarData) Decode(elem *etree.Element) error {
	p.ContentType = elem.Text()
	if versionAttr := elem.SelectAttr("version"); versionAttr != nil {
		p.Version = versionAttr.Value
	}
	return nil
}

type MaxResourceSize struct {
	Value int64
}

func (p MaxResourceSize) Encode() *etree.Element {
	elem := createElement("max-resource-size")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

func (p *MaxResourceSize) Decode(elem *etree.Element) error {
	val, err := strconv.ParseInt(elem.Text(), 10, 64)
	if err != nil {
		return err
	}
	p.Value = val
	return nil
}

type MinDateTime struct {
	Value time.Time
}

func (p MinDateTime) Encode() *etree.Element {
	elem := createElement("min-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

func (p *MinDateTime) Decode(elem *etree.Element) error {
	t, err := time.Parse(time.RFC3339, elem.Text())
	if err != nil {
		return err
	}
	p.Value = t
	return nil
}

type MaxDateTime struct {
	Value time.Time
}

func (p MaxDateTime) Encode() *etree.Element {
	elem := createElement("max-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

func (p *MaxDateTime) Decode(elem *etree.Element) error {
	t, err := time.Parse(time.RFC3339, elem.Text())
	if err != nil {
		return err
	}
	p.Value = t
	return nil
}

type MaxInstances struct {
	Value int
}

func (p MaxInstances) Encode() *etree.Element {
	elem := createElement("max-instances")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

func (p *MaxInstances) Decode(elem *etree.Element) error {
	val, err := strconv.Atoi(elem.Text())
	if err != nil {
		return err
	}
	p.Value = val
	return nil
}

type MaxAttendeesPerInstance struct {
	Value int
}

func (p MaxAttendeesPerInstance) Encode() *etree.Element {
	elem := createElement("max-attendees-per-instance")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

func (p *MaxAttendeesPerInstance) Decode(elem *etree.Element) error {
	val, err := strconv.Atoi(elem.Text())
	if err != nil {
		return err
	}
	p.Value = val
	return nil
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

func (p *CalendarHomeSet) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Href = href.Text()
	}
	return nil
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

func (p *ScheduleInboxURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Href = href.Text()
	}
	return nil
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

func (p *ScheduleOutboxURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Href = href.Text()
	}
	return nil
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

func (p *ScheduleDefaultCalendarURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Href = href.Text()
	}
	return nil
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

func (p *CalendarUserAddressSet) Decode(elem *etree.Element) error {
	p.Addresses = []string{}

	hrefs := elem.FindElements("href")
	for _, href := range hrefs {
		p.Addresses = append(p.Addresses, href.Text())
	}

	return nil
}

type CalendarUserType struct {
	Value string
}

func (p CalendarUserType) Encode() *etree.Element {
	elem := createElement("calendar-user-type")
	elem.SetText(p.Value)
	return elem
}

func (p *CalendarUserType) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}
