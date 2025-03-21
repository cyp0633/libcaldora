package xml

import (
	"testing"

	"github.com/beevik/etree"
)

func TestAddNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *etree.Document
		wantAttr map[string]string
	}{
		{
			name: "add namespaces to empty document with root",
			setup: func() *etree.Document {
				doc := etree.NewDocument()
				doc.CreateElement("test")
				return doc
			},
			wantAttr: map[string]string{
				"xmlns:D":  DAV,
				"xmlns:C":  CalDAV,
				"xmlns:CS": CalendarServer,
			},
		},
		{
			name: "add namespaces to document with existing attributes",
			setup: func() *etree.Document {
				doc := etree.NewDocument()
				root := doc.CreateElement("test")
				root.CreateAttr("xmlns:custom", "http://example.com/ns")
				return doc
			},
			wantAttr: map[string]string{
				"xmlns:D":      DAV,
				"xmlns:C":      CalDAV,
				"xmlns:CS":     CalendarServer,
				"xmlns:custom": "http://example.com/ns",
			},
		},
		{
			name: "add namespaces to document without root",
			setup: func() *etree.Document {
				return etree.NewDocument()
			},
			wantAttr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.setup()
			AddNamespaces(doc)

			if tt.wantAttr == nil {
				if doc.Root() != nil {
					t.Error("expected no root element")
				}
				return
			}

			root := doc.Root()
			if root == nil {
				t.Fatal("expected root element")
			}

			// Create a map of actual attributes
			gotAttr := make(map[string]string)
			for _, attr := range root.Attr {
				attrName := attr.Space
				if attrName != "" {
					attrName += ":"
				}
				attrName += attr.Key
				if attrName == "xmlns:D" || attrName == "xmlns:C" || attrName == "xmlns:CS" || attrName == "xmlns:custom" {
					gotAttr[attrName] = attr.Value
				}
			}

			// Compare expected and actual attributes
			for attrName, wantValue := range tt.wantAttr {
				if gotValue, ok := gotAttr[attrName]; !ok {
					t.Errorf("missing attribute %s", attrName)
				} else if gotValue != wantValue {
					t.Errorf("attribute %s = %s, want %s", attrName, gotValue, wantValue)
				}
			}

			// Check no extra namespace attributes were added
			for attrName := range gotAttr {
				if _, ok := tt.wantAttr[attrName]; !ok {
					t.Errorf("unexpected attribute %s = %s", attrName, gotAttr[attrName])
				}
			}
		})
	}
}

func TestNamespaceConstants(t *testing.T) {
	if DAV != "DAV:" {
		t.Errorf("DAV namespace = %s, want DAV:", DAV)
	}
	if CalDAV != "urn:ietf:params:xml:ns:caldav" {
		t.Errorf("CalDAV namespace = %s, want urn:ietf:params:xml:ns:caldav", CalDAV)
	}
	if CalendarServer != "http://calendarserver.org/ns/" {
		t.Errorf("CalendarServer namespace = %s, want http://calendarserver.org/ns/", CalendarServer)
	}
}
