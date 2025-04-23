package props

import "github.com/beevik/etree"

// Apple CalendarServer Extensions

type GetCTag struct {
	Value string
}

func (p GetCTag) Encode() *etree.Element {
	elem := createElement("getctag")
	elem.SetText(p.Value)
	return elem
}

func (p *GetCTag) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type CalendarChanges struct {
	Href string
}

func (p CalendarChanges) Encode() *etree.Element {
	elem := createElement("calendar-changes")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

func (p *CalendarChanges) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Href = href.Text()
	}
	return nil
}

type SharedURL struct {
	Value string
}

func (p SharedURL) Encode() *etree.Element {
	elem := createElement("shared-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Value)
	return elem
}

func (p *SharedURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Value = href.Text()
	}
	return nil
}

type Invite struct {
	Value string
}

func (p Invite) Encode() *etree.Element {
	elem := createElement("invite")
	elem.SetText(p.Value)
	return elem
}

func (p *Invite) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type NotificationURL struct {
	Value string
}

func (p NotificationURL) Encode() *etree.Element {
	elem := createElement("notification-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Value)
	return elem
}

func (p *NotificationURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Value = href.Text()
	}
	return nil
}

type AutoSchedule struct {
	Value bool
}

func (p AutoSchedule) Encode() *etree.Element {
	elem := createElement("auto-schedule")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

func (p *AutoSchedule) Decode(elem *etree.Element) error {
	text := elem.Text()
	p.Value = text == "true" || text == "1"
	return nil
}

type CalendarProxyReadFor struct {
	Hrefs []string
}

func (p CalendarProxyReadFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-read-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

func (p *CalendarProxyReadFor) Decode(elem *etree.Element) error {
	p.Hrefs = []string{}
	hrefs := elem.FindElements("href")
	for _, href := range hrefs {
		p.Hrefs = append(p.Hrefs, href.Text())
	}
	return nil
}

type CalendarProxyWriteFor struct {
	Hrefs []string
}

func (p CalendarProxyWriteFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-write-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

func (p *CalendarProxyWriteFor) Decode(elem *etree.Element) error {
	p.Hrefs = []string{}
	hrefs := elem.FindElements("href")
	for _, href := range hrefs {
		p.Hrefs = append(p.Hrefs, href.Text())
	}
	return nil
}

type CalendarColor struct {
	Value string
}

func (p CalendarColor) Encode() *etree.Element {
	elem := createElement("calendar-color")
	elem.SetText(p.Value)
	return elem
}

func (p *CalendarColor) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

// Google CalDAV Extensions

type Color struct {
	Value string
}

func (p Color) Encode() *etree.Element {
	elem := createElement("color")
	elem.SetText(p.Value)
	return elem
}

func (p *Color) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type Timezone struct {
	Value string
}

func (p Timezone) Encode() *etree.Element {
	elem := createElement("timezone")
	elem.SetText(p.Value)
	return elem
}

func (p *Timezone) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type Hidden struct {
	Value bool
}

func (p Hidden) Encode() *etree.Element {
	elem := createElement("hidden")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

func (p *Hidden) Decode(elem *etree.Element) error {
	text := elem.Text()
	p.Value = text == "true" || text == "1"
	return nil
}

type Selected struct {
	Value bool
}

func (p Selected) Encode() *etree.Element {
	elem := createElement("selected")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

func (p *Selected) Decode(elem *etree.Element) error {
	text := elem.Text()
	p.Value = text == "true" || text == "1"
	return nil
}
