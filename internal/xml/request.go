package xml

import (
	"fmt"

	"github.com/beevik/etree"
)

// CalendarMultigetRequest represents a calendar-multiget REPORT request
type CalendarMultigetRequest struct {
	Prop  []string
	Hrefs []string
}

// Parse parses a calendar-multiget request from an XML document
func (r *CalendarMultigetRequest) Parse(doc *etree.Document) error {
	if doc == nil || doc.Root() == nil {
		return fmt.Errorf("empty document")
	}

	root := doc.Root()
	if root.Tag != "calendar-multiget" {
		return fmt.Errorf("invalid root tag: %s", root.Tag)
	}

	// Reset the request fields
	r.Prop = nil
	r.Hrefs = nil

	// Get props
	if prop := FindElementWithNS(root, TagProp); prop != nil {
		for _, p := range prop.ChildElements() {
			r.Prop = append(r.Prop, p.Tag)
		}
	}

	// Get hrefs
	for _, href := range root.SelectElements("href") {
		r.Hrefs = append(r.Hrefs, href.Text())
	}

	return nil
}

// ToXML converts a CalendarMultigetRequest to an XML document
func (r *CalendarMultigetRequest) ToXML() *etree.Document {
	doc := etree.NewDocument()
	root := CreateRootElement(doc, "calendar-multiget", true)
	AddSelectedNamespaces(doc, DAV, CalDAV)

	if len(r.Prop) > 0 {
		prop := CreateElementWithNS(root, TagProp)
		for _, name := range r.Prop {
			CreateElementWithNS(prop, name)
		}
	}

	for _, href := range r.Hrefs {
		h := CreateElementWithNS(root, TagHref)
		h.SetText(href)
	}

	return doc
}

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
// SyncCollectionRequest represents a sync-collection REPORT request
type SyncCollectionRequest struct {
    SyncToken string
    SyncLevel string
    Prop      []string
}

// Parse parses a sync-collection request from an XML document
func (r *SyncCollectionRequest) Parse(doc *etree.Document) error {
    if doc == nil || doc.Root() == nil {
        return fmt.Errorf("empty document")
    }

    root := doc.Root()
    if root.Tag != "sync-collection" {
        return fmt.Errorf("invalid root tag: %s", root.Tag)
    }

    // Reset the request fields
    r.SyncToken = ""
    r.SyncLevel = ""
    r.Prop = nil

    // Get sync-token
    if token := FindElementWithNS(root, "sync-token"); token != nil {
        r.SyncToken = token.Text()
    }

    // Get sync-level
    if level := FindElementWithNS(root, "sync-level"); level != nil {
        r.SyncLevel = level.Text()
    }

    // Get props
    if prop := FindElementWithNS(root, TagProp); prop != nil {
        for _, p := range prop.ChildElements() {
            r.Prop = append(r.Prop, p.Tag)
        }
    }

    return nil
}

// ToXML converts a SyncCollectionRequest to an XML document
func (r *SyncCollectionRequest) ToXML() *etree.Document {
    doc := etree.NewDocument()
    root := CreateRootElement(doc, "sync-collection", false)
    AddSelectedNamespaces(doc, DAV)

    token := CreateElementWithNS(root, "sync-token")
    token.SetText(r.SyncToken)

    level := CreateElementWithNS(root, "sync-level")
    level.SetText(r.SyncLevel)

    if len(r.Prop) > 0 {
        prop := CreateElementWithNS(root, TagProp)
        for _, name := range r.Prop {
            CreateElementWithNS(prop, name)
        }
    }

    return doc
}

func (r *PropfindRequest) ToXML() *etree.Document {
doc := etree.NewDocument()
// Create root element without namespace prefix by default
root := CreateRootElement(doc, TagPropfind, false)
AddSelectedNamespaces(doc, DAV, CalDAV, CalendarServer, AppleICal)

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
