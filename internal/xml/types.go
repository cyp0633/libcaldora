package xml

import "github.com/beevik/etree"

// Common XML tag names used in CalDAV
const (
	TagPropfind     = "propfind"
	TagProp         = "prop"
	TagPropname     = "propname"
	TagAllprop      = "allprop"
	TagInclude      = "include"
	TagMultistatus  = "multistatus"
	TagResponse     = "response"
	TagHref         = "href"
	TagPropstat     = "propstat"
	TagStatus       = "status"
	TagResourcetype = "resourcetype"
	TagCollection   = "collection"
	TagCalendar     = "calendar"
)

// Property represents a generic XML property
type Property struct {
	Name        string
	Namespace   string
	TextContent string
	Children    []Property
}

// ToElement converts a Property to an etree.Element
func (p *Property) ToElement() *etree.Element {
	elem := etree.NewElement(p.Name)
	if p.Namespace != "" {
		elem.Space = p.Namespace
	}
	if p.TextContent != "" {
		elem.SetText(p.TextContent)
	}
	for _, child := range p.Children {
		elem.AddChild(child.ToElement())
	}
	return elem
}

// FromElement populates a Property from an etree.Element
func (p *Property) FromElement(elem *etree.Element) {
	p.Name = elem.Tag
	p.Namespace = elem.Space
	p.TextContent = elem.Text()
	p.Children = nil
	for _, child := range elem.ChildElements() {
		childProp := Property{}
		childProp.FromElement(child)
		p.Children = append(p.Children, childProp)
	}
}

// Error represents a WebDAV error response
type Error struct {
	Namespace string
	Tag       string
	Message   string
}

// ToElement converts an Error to an etree.Element
func (e *Error) ToElement() *etree.Element {
	err := etree.NewElement("error")
	tag := etree.NewElement(e.Tag)
	if e.Namespace != "" {
		tag.Space = e.Namespace
	}
	if e.Message != "" {
		tag.SetText(e.Message)
	}
	err.AddChild(tag)
	return err
}
