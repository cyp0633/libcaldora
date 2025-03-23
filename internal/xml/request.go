package xml

import (
	"fmt"

	"github.com/beevik/etree"
)

// PropfindRequest represents a PROPFIND request
type PropfindRequest struct {
	Prop      []string
	PropNames bool
	AllProp   bool
	Include   []string
}

// Parse parses a PROPFIND request from an XML document
func (r *PropfindRequest) Parse(doc *etree.Document) error {
	if doc == nil || doc.Root() == nil {
		return fmt.Errorf("empty document")
	}

	root := doc.Root()
	if root.Tag != TagPropfind {
		return fmt.Errorf("invalid root tag: %s", root.Tag)
	}

	// Reset the request fields
	r.Prop = nil
	r.Include = nil
	r.PropNames = false
	r.AllProp = false

	// Check for prop
	if prop := FindElementWithNS(root, TagProp); prop != nil {
		for _, p := range prop.ChildElements() {
			r.Prop = append(r.Prop, p.Tag)
		}
	}

	// Check for propname
	if FindElementWithNS(root, TagPropname) != nil {
		r.PropNames = true
	}

	// Check for allprop
	if FindElementWithNS(root, TagAllprop) != nil {
		r.AllProp = true
	}

	// Check for include
	if include := FindElementWithNS(root, TagInclude); include != nil {
		for _, item := range include.ChildElements() {
			r.Include = append(r.Include, item.Tag)
		}
	}

	return nil
}

// ToXML converts a PropfindRequest to an XML document
func (r *PropfindRequest) ToXML() *etree.Document {
	doc := etree.NewDocument()
	// Create root element without namespace prefix by default
	root := CreateRootElement(doc, TagPropfind, false)
	AddSelectedNamespaces(doc, DAV, CalDAV, CalendarServer)

	if r.PropNames {
		CreateElementWithNS(root, TagPropname)
	} else if r.AllProp {
		CreateElementWithNS(root, TagAllprop)
		if len(r.Include) > 0 {
			include := CreateElementWithNS(root, TagInclude)
			for _, name := range r.Include {
				CreateElementWithNS(include, name)
			}
		}
	} else if len(r.Prop) > 0 {
		prop := CreateElementWithNS(root, TagProp)
		for _, name := range r.Prop {
			CreateElementWithNS(prop, name)
		}
	}

	return doc
}
