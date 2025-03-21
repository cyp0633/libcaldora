package xml

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/beevik/etree"
)

func TestPropfindRequest_Parse(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    *PropfindRequest
		wantErr bool
	}{
		{
			name:    "empty document",
			xml:     "",
			wantErr: true,
		},
		{
			name:    "invalid root tag",
			xml:     `<?xml version="1.0" encoding="utf-8"?><wrong/>`,
			wantErr: true,
		},
		{
			name: "simple propfind with specific props",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:prop>
<D:displayname/>
<D:resourcetype/>
<C:calendar-home-set/>
</D:prop>
</D:propfind>`,
			want: &PropfindRequest{
				Prop: []string{"displayname", "resourcetype", "calendar-home-set"},
			},
		},
		{
			name: "propfind with propname",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
<D:propname/>
</D:propfind>`,
			want: &PropfindRequest{
				PropNames: true,
			},
		},
		{
			name: "propfind with allprop",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
<D:allprop/>
</D:propfind>`,
			want: &PropfindRequest{
				AllProp: true,
			},
		},
		{
			name: "allprop with include",
			xml: `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
<D:allprop/>
<D:include>
<C:calendar-data/>
<D:sync-token/>
</D:include>
</D:propfind>`,
			want: &PropfindRequest{
				AllProp: true,
				Include: []string{"calendar-data", "sync-token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			if tt.xml != "" {
				err := doc.ReadFromString(tt.xml)
				if err != nil {
					t.Fatalf("failed to parse test XML: %v", err)
				}
			}

			var got PropfindRequest
			err := got.Parse(doc)

			if (err != nil) != tt.wantErr {
				t.Errorf("PropfindRequest.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if tt.want == nil {
					t.Error("PropfindRequest.Parse() succeeded but want error")
					return
				}
				if !reflect.DeepEqual(&got, tt.want) {
					t.Errorf("PropfindRequest.Parse() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestPropfindRequest_ToXML(t *testing.T) {
	tests := []struct {
		name    string
		request PropfindRequest
		want    string
	}{
		{
			name: "propfind with specific props",
			request: PropfindRequest{
				Prop: []string{"displayname", "resourcetype", "calendar-data"},
			},
			want: `<propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/"><D:prop><D:displayname/><D:resourcetype/><C:calendar-data/></D:prop></propfind>`,
		},
		{
			name: "propfind with propname",
			request: PropfindRequest{
				PropNames: true,
			},
			want: `<propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/"><D:propname/></propfind>`,
		},
		{
			name: "allprop with include",
			request: PropfindRequest{
				AllProp: true,
				Include: []string{"calendar-data", "sync-token"},
			},
			want: `<propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/"><D:allprop/><D:include><C:calendar-data/><D:sync-token/></D:include></propfind>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.request.ToXML()
			got, err := doc.WriteToString()
			if err != nil {
				t.Fatalf("failed to serialize XML: %v", err)
			}

			// Remove XML declaration and normalize spaces
			gotNorm := normalizeXML(got)
			wantNorm := normalizeXML(tt.want)

			if gotNorm != wantNorm {
				t.Errorf("PropfindRequest.ToXML() =\nGot:  [%s]\nWant: [%s]", gotNorm, wantNorm)
				fmt.Printf("Debug - Got runes:  %v\n", []rune(gotNorm))
				fmt.Printf("Debug - Want runes: %v\n", []rune(wantNorm))
			}
		})
	}
}
