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

type Invite struct {
	Value string
}

func (p Invite) Encode() *etree.Element {
	elem := createElement("invite")
	elem.SetText(p.Value)
	return elem
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

type CalendarColor struct {
	Value string
}

func (p CalendarColor) Encode() *etree.Element {
	elem := createElement("calendar-color")
	elem.SetText(p.Value)
	return elem
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

type Timezone struct {
	Value string
}

func (p Timezone) Encode() *etree.Element {
	elem := createElement("timezone")
	elem.SetText(p.Value)
	return elem
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
