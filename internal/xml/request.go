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
			for _, include := range root.FindElements(".//" + TagInclude) {
				r.Include = append(r.Include, include.Text())
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
		root.CreateElement(TagPropname)
	} else if r.AllProp {
		root.CreateElement(TagAllprop)
		if len(r.Include) > 0 {
			include := root.CreateElement(TagInclude)
			for _, prop := range r.Include {
				elem := include.CreateElement(prop)
				if prop == "calendar-data" {
					elem.Space = "C"
				}
			}
		}
	} else if len(r.Prop) > 0 {
		prop := root.CreateElement(TagProp)
		for _, name := range r.Prop {
			elem := prop.CreateElement(name)
			if name == "calendar-data" {
				elem.Space = "C"
			}
		}
	}

	return doc
}
