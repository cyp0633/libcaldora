package xml

import (
	"strings"

	"github.com/beevik/etree"
)

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

	// Use the registry to determine namespace
	if p.Namespace == "" {
		p.Namespace = GetElementNamespace(p.Name)
	}

	// Apply namespace prefix from registry
	if prefix := GetNamespacePrefix(p.Namespace); prefix != "" {
		elem.Space = prefix
	}

	if p.TextContent != "" {
		elem.SetText(p.TextContent)
	}

	// Add attributes
	for key, value := range p.Attributes {
		elem.CreateAttr(key, value)
	}

	// Process children with their own namespaces
	for _, child := range p.Children {
		elem.AddChild(child.ToElement())
	}

	return elem
}

// FromElement populates a Property from an etree.Element
func (p *Property) FromElement(elem *etree.Element) {
	p.Name = elem.Tag
	p.TextContent = elem.Text()
	p.Children = nil
	p.Attributes = make(map[string]string)

	// Handle namespace from prefix using registry
	if elem.Space != "" {
		if ns, ok := PrefixNamespace[elem.Space]; ok {
			p.Namespace = ns
		} else {
			p.Namespace = elem.Space
		}
	}

	// Copy attributes
	for _, attr := range elem.Attr {
		if attr.Key != "xmlns" && !strings.HasPrefix(attr.Key, "xmlns:") {
			p.Attributes[attr.Key] = attr.Value
		}
	}

	// Process child elements
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
	// Create error element with DAV namespace prefix
	err := etree.NewElement(TagError)
	if prefix := GetNamespacePrefix(DAV); prefix != "" {
		err.Space = prefix
	}

	// Create the specific error tag
	tag := etree.NewElement(e.Tag)

	// Apply namespace to the error tag if specified
	if e.Namespace != "" {
		if prefix := GetNamespacePrefix(e.Namespace); prefix != "" {
			tag.Space = prefix
		}
	}

	if e.Message != "" {
		tag.SetText(e.Message)
	}
	err.AddChild(tag)
	return err
}
