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
	TagError        = "error"
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
	Attributes  map[string]string
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
	// Add attributes
	for key, value := range p.Attributes {
		elem.CreateAttr(key, value)
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
	p.Attributes = make(map[string]string)

	// Copy attributes
	for _, attr := range elem.Attr {
		p.Attributes[attr.Key] = attr.Value
	}

	for _, child := range elem.ChildElements() {
		childProp := Property{}
		childProp.FromElement(child)
		p.Children = append(p.Children, childProp)
	}
}

// GetAttr returns the value of an attribute, or empty string if not found
func (p *Property) GetAttr(name string) string {
	if p.Attributes == nil {
		return ""
	}
	return p.Attributes[name]
}

// SetAttr sets an attribute value
func (p *Property) SetAttr(name, value string) {
	if p.Attributes == nil {
		p.Attributes = make(map[string]string)
	}
	p.Attributes[name] = value
}

// Error represents a WebDAV error response
type Error struct {
	Namespace string
	Tag       string
	Message   string
}

// ToElement converts an Error to an etree.Element
func (e *Error) ToElement() *etree.Element {
	err := etree.NewElement(TagError)
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
