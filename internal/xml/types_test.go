package xml

import (
	"reflect"
	"testing"

	"github.com/beevik/etree"
)

func TestProperty_ToElement(t *testing.T) {
	tests := []struct {
		name     string
		property Property
		want     func() *etree.Element
	}{
		{
			name: "simple property without namespace",
			property: Property{
				Name:        "displayname",
				TextContent: "Calendar",
			},
			want: func() *etree.Element {
				elem := etree.NewElement("displayname")
				elem.Space = "D" // Add DAV namespace prefix
				elem.SetText("Calendar")
				return elem
			},
		},
		{
			name: "property with namespace",
			property: Property{
				Name:        "calendar",
				Namespace:   CalDAV,
				TextContent: "Calendar Resource",
			},
			want: func() *etree.Element {
				elem := etree.NewElement("calendar")
				elem.Space = "C" // CalDAV prefix
				elem.SetText("Calendar Resource")
				return elem
			},
		},
		{
			name: "property with child elements",
			property: Property{
				Name:      "resourcetype",
				Namespace: DAV,
				Children: []Property{
					{
						Name:      "collection",
						Namespace: DAV,
					},
					{
						Name:      "calendar",
						Namespace: CalDAV,
					},
				},
			},
			want: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = "D"
				child1 := etree.NewElement("collection")
				child1.Space = "D"
				child2 := etree.NewElement("calendar")
				child2.Space = "C"
				elem.AddChild(child1)
				elem.AddChild(child2)
				return elem
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.property.ToElement()
			want := tt.want()

			gotStr := elementToString(got)
			wantStr := elementToString(want)
			if gotStr != wantStr {
				t.Errorf("Property.ToElement() =\n%s\nwant:\n%s", gotStr, wantStr)
			}
		})
	}
}

func TestProperty_FromElement(t *testing.T) {
	tests := []struct {
		name  string
		input func() *etree.Element
		want  Property
	}{
		{
			name: "simple property without namespace",
			input: func() *etree.Element {
				elem := etree.NewElement("displayname")
				elem.SetText("Calendar")
				return elem
			},
			want: Property{
				Name:        "displayname",
				TextContent: "Calendar",
				Attributes:  make(map[string]string),
			},
		},
		{
			name: "property with namespace",
			input: func() *etree.Element {
				elem := etree.NewElement("calendar")
				elem.Space = CalDAV
				elem.SetText("Calendar Resource")
				return elem
			},
			want: Property{
				Name:        "calendar",
				Namespace:   CalDAV,
				TextContent: "Calendar Resource",
				Attributes:  make(map[string]string),
			},
		},
		{
			name: "property with child elements",
			input: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = DAV
				child1 := etree.NewElement("collection")
				child1.Space = DAV
				child2 := etree.NewElement("calendar")
				child2.Space = CalDAV
				elem.AddChild(child1)
				elem.AddChild(child2)
				return elem
			},
			want: Property{
				Name:      "resourcetype",
				Namespace: DAV,
				Children: []Property{
					{
						Name:       "collection",
						Namespace:  DAV,
						Attributes: make(map[string]string),
					},
					{
						Name:       "calendar",
						Namespace:  CalDAV,
						Attributes: make(map[string]string),
					},
				},
				Attributes: make(map[string]string),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Property
			got.FromElement(tt.input())

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Property.FromElement() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestError_ToElement(t *testing.T) {
	tests := []struct {
		name  string
		error Error
		want  func() *etree.Element
	}{
		{
			name: "simple error without namespace",
			error: Error{
				Tag:     "cannot-modify-property",
				Message: "Property cannot be modified",
			},
			want: func() *etree.Element {
				elem := etree.NewElement("error")
				elem.Space = "D" // Add DAV namespace prefix
				child := etree.NewElement("cannot-modify-property")
				child.SetText("Property cannot be modified")
				elem.AddChild(child)
				return elem
			},
		},
		{
			name: "error with namespace",
			error: Error{
				Namespace: DAV,
				Tag:       "locked",
				Message:   "Resource is locked",
			},
			want: func() *etree.Element {
				elem := etree.NewElement("error")
				elem.Space = "D" // Add DAV namespace prefix
				child := etree.NewElement("locked")
				child.Space = "D"
				child.SetText("Resource is locked")
				elem.AddChild(child)
				return elem
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.error.ToElement()
			want := tt.want()

			gotStr := elementToString(got)
			wantStr := elementToString(want)
			if gotStr != wantStr {
				t.Errorf("Error.ToElement() =\n%s\nwant:\n%s", gotStr, wantStr)
			}
		})
	}
}
