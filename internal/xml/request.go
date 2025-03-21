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

	for _, child := range root.ChildElements() {
		switch child.Tag {
		case TagProp:
			for _, prop := range child.ChildElements() {
				r.Prop = append(r.Prop, prop.Tag)
			}
		case TagPropname:
			r.PropNames = true
		case TagAllprop:
			r.AllProp = true
		case TagInclude:
			for _, item := range child.ChildElements() {
				r.Include = append(r.Include, item.Tag)
			}
		}
	}

	return nil
}

// ToXML converts a PropfindRequest to an XML document
func (r *PropfindRequest) ToXML() *etree.Document {
	doc := etree.NewDocument()
	root := doc.CreateElement(TagPropfind)
	AddNamespaces(doc)

	if r.PropNames {
		root.CreateElement(TagPropname).Space = "D"
	} else if r.AllProp {
		root.CreateElement(TagAllprop).Space = "D"
		if len(r.Include) > 0 {
			include := root.CreateElement(TagInclude)
			include.Space = "D"
			for _, name := range r.Include {
				elem := include.CreateElement(name)
				if name == "calendar-data" {
					elem.Space = "C"
				} else if name == "sync-token" {
					elem.Space = "D"
				}
			}
		}
	} else if len(r.Prop) > 0 {
		prop := root.CreateElement(TagProp)
		prop.Space = "D"
		for _, name := range r.Prop {
			elem := prop.CreateElement(name)
			if name == "calendar-data" {
				elem.Space = "C"
			} else {
				elem.Space = "D"
			}
		}
	}

	return doc
}
