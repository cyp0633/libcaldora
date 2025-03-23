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
				"xmlns:D":    DAV,
				"xmlns:C":    CalDAV,
				"xmlns:CS":   CalendarServer,
				"xmlns:A":    AppleICal,
				"xmlns:CARD": CardDAV,
				"xmlns:ICAL": ICal,
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
				"xmlns:A":      AppleICal,
				"xmlns:CARD":   CardDAV,
				"xmlns:ICAL":   ICal,
				"xmlns:custom": "http://example.com/ns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.setup()
			AddNamespaces(doc)

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
				gotAttr[attrName] = attr.Value
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

func TestAddSelectedNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		wantAttr   map[string]string
	}{
		{
			name:       "add single namespace",
			namespaces: []string{DAV},
			wantAttr:   map[string]string{"xmlns:D": DAV},
		},
		{
			name:       "add multiple namespaces",
			namespaces: []string{DAV, CalDAV},
			wantAttr: map[string]string{
				"xmlns:D": DAV,
				"xmlns:C": CalDAV,
			},
		},
		{
			name:       "add no namespaces",
			namespaces: []string{},
			wantAttr: map[string]string{
				"xmlns:D":    DAV,
				"xmlns:C":    CalDAV,
				"xmlns:CS":   CalendarServer,
				"xmlns:A":    AppleICal,
				"xmlns:CARD": CardDAV,
				"xmlns:ICAL": ICal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			doc.CreateElement("test")
			AddSelectedNamespaces(doc, tt.namespaces...)

			root := doc.Root()
			gotAttr := make(map[string]string)
			for _, attr := range root.Attr {
				gotAttr["xmlns:"+attr.Key] = attr.Value
			}

			for attrName, wantValue := range tt.wantAttr {
				if gotValue, ok := gotAttr[attrName]; !ok {
					t.Errorf("missing attribute %s", attrName)
				} else if gotValue != wantValue {
					t.Errorf("attribute %s = %s, want %s", attrName, gotValue, wantValue)
				}
			}
		})
	}
}

func TestNamespaceRegistry(t *testing.T) {
	tests := []struct {
		name       string
		elemName   string
		wantNS     string
		wantPrefix string
	}{
		{"DAV element", "response", DAV, "D"},
		{"CalDAV element", "calendar", CalDAV, "C"},
		{"Apple element", "calendar-color", AppleICal, "A"},
		{"CalendarServer element", "getctag", CalendarServer, "CS"},
		{"Unknown element", "unknown", DAV, "D"}, // Default to DAV
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotNS := GetElementNamespace(tt.elemName); gotNS != tt.wantNS {
				t.Errorf("GetElementNamespace(%q) = %q, want %q", tt.elemName, gotNS, tt.wantNS)
			}
			if gotPrefix := GetNamespacePrefix(tt.wantNS); gotPrefix != tt.wantPrefix {
				t.Errorf("GetNamespacePrefix(%q) = %q, want %q", tt.wantNS, gotPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestRegisterNamespace(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		prefix    string
		wantError bool
	}{
		{"Valid registration", "http://example.com/ns", "EX", false},
		{"Empty URI", "", "TEST", true},
		{"Empty prefix", "http://example.com/ns2", "", true},
		{"Duplicate URI", DAV, "TEST", true},
		{"Duplicate prefix", "http://example.com/ns3", "D", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterNamespace(tt.uri, tt.prefix)
			if tt.wantError {
				if err == nil {
					t.Error("RegisterNamespace() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("RegisterNamespace() error = %v, want nil", err)
				}
				if got := GetNamespacePrefix(tt.uri); got != tt.prefix {
					t.Errorf("GetNamespacePrefix(%q) = %q, want %q", tt.uri, got, tt.prefix)
				}
			}
		})
	}
}

func TestRegisterElement(t *testing.T) {
	tests := []struct {
		name      string
		elemName  string
		namespace string
		wantError bool
	}{
		{"Valid registration", "custom-element", DAV, false},
		{"Empty element name", "", DAV, true},
		{"Empty namespace", "element", "", true},
		{"Unregistered namespace", "element", "http://unknown.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterElement(tt.elemName, tt.namespace)
			if tt.wantError {
				if err == nil {
					t.Error("RegisterElement() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("RegisterElement() error = %v, want nil", err)
				}
				if got := GetElementNamespace(tt.elemName); got != tt.namespace {
					t.Errorf("GetElementNamespace(%q) = %q, want %q", tt.elemName, got, tt.namespace)
				}
			}
		})
	}
}
