package xml

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/stretchr/testify/assert"
)

func TestRealDiscoveryRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/01_root_discovery_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &PropfindRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{"getcontenttype", "resourcetype", "displayname", "calendar-color"}, req.Prop)
	assert.False(t, req.PropNames)
	assert.False(t, req.AllProp)
	assert.Empty(t, req.Include)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "propfind", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
}

func TestRealDiscoveryResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/01_root_discovery_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	t.Logf("Parsing calendar-multiget response file")
	doc.WriteTo(os.Stdout) // Debug: print the XML document
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Log debug info
	t.Logf("Root tag: %s", doc.Root().Tag)
	t.Logf("Root namespaces: %v", doc.Root().Attr)
	t.Logf("Response elements found: %d", len(resp.Responses))
	if len(resp.Responses) > 0 {
		t.Logf("First response href: %s", resp.Responses[0].Href)
		if len(resp.Responses[0].PropStats) > 0 {
			t.Logf("First propstat props: %v", resp.Responses[0].PropStats[0].Props)
		}
	}

	// Verify parsed data
	assert.Len(t, resp.Responses, 1, "Expected 1 response element")
	r := resp.Responses[0]
	assert.Equal(t, "/", r.Href)
	assert.Len(t, r.PropStats, 2)

	// First propstat (404)
	assert.Contains(t, r.PropStats[0].Status, "404 Not Found")
	assert.Len(t, r.PropStats[0].Props, 3)
	propNames := []string{r.PropStats[0].Props[0].Name, r.PropStats[0].Props[1].Name, r.PropStats[0].Props[2].Name}
	assert.ElementsMatch(t, []string{"getcontenttype", "displayname", "calendar-color"}, propNames)

	// Second propstat (200)
	assert.Contains(t, r.PropStats[1].Status, "200 OK")
	assert.Len(t, r.PropStats[1].Props, 1)
	assert.Equal(t, "resourcetype", r.PropStats[1].Props[0].Name)
	assert.Len(t, r.PropStats[1].Props[0].Children, 1)
	assert.Equal(t, "collection", r.PropStats[1].Props[0].Children[0].Name)

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
}

func TestCalendarPropsRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/02_calendar_props_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &PropfindRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{
		"resourcetype",
		"owner",
		"current-user-principal",
		"current-user-privilege-set",
		"supported-report-set",
		"supported-calendar-component-set",
		"getctag",
	}, req.Prop)
	assert.False(t, req.PropNames)
	assert.False(t, req.AllProp)
	assert.Empty(t, req.Include)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "propfind", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	caldav := root.SelectAttr("xmlns:C")
	calendarserver := root.SelectAttr("xmlns:CS")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.NotNil(t, calendarserver, "CalendarServer namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
	assert.Equal(t, CalendarServer, calendarserver.Value)
}

func TestCalendarPropsResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/02_calendar_props_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	t.Logf("Parsing calendar-multiget response file")
	doc.WriteTo(os.Stdout) // Debug: print the XML document
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Log debug info
	t.Logf("Root tag: %s", doc.Root().Tag)
	t.Logf("Root namespaces: %v", doc.Root().Attr)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1, "Expected 1 response element")
	r := resp.Responses[0]
	assert.Equal(t, "/calendars/example/main-calendar/", r.Href)
	assert.Len(t, r.PropStats, 1)

	// Check propstat
	propstat := r.PropStats[0]
	assert.Contains(t, propstat.Status, "200 OK")
	t.Logf("Got %d props: %v", len(propstat.Props), propstat.Props)
	assert.Len(t, propstat.Props, 7) // The correct count based on the actual response

	// Verify each property
	for _, prop := range propstat.Props {
		switch prop.Name {
		case "resourcetype":
			assert.Len(t, prop.Children, 2)
			types := []string{prop.Children[0].Name, prop.Children[1].Name}
			assert.ElementsMatch(t, []string{"calendar", "collection"}, types)

		case "owner", "current-user-principal":
			assert.Len(t, prop.Children, 1)
			assert.Equal(t, "href", prop.Children[0].Name)
			assert.Equal(t, "/principals/users/example/", prop.Children[0].TextContent)

		case "current-user-privilege-set":
			assert.Len(t, prop.Children, 5)
			var privileges []string
			for _, priv := range prop.Children {
				assert.Equal(t, "privilege", priv.Name)
				assert.Len(t, priv.Children, 1)
				privileges = append(privileges, priv.Children[0].Name)
			}
			assert.ElementsMatch(t, []string{"read", "all", "write", "write-properties", "write-content"}, privileges)

		case "supported-report-set":
			assert.Len(t, prop.Children, 6)
			var reports []string
			for _, sr := range prop.Children {
				assert.Equal(t, "supported-report", sr.Name)
				assert.Len(t, sr.Children, 1)
				assert.Equal(t, "report", sr.Children[0].Name)
				assert.Len(t, sr.Children[0].Children, 1)
				reports = append(reports, sr.Children[0].Children[0].Name)
			}
			assert.ElementsMatch(t, []string{
				"expand-property",
				"principal-search-property-set",
				"principal-property-search",
				"sync-collection",
				"calendar-multiget",
				"calendar-query",
			}, reports)

		case "supported-calendar-component-set":
			assert.Len(t, prop.Children, 3)
			var components []string
			for _, comp := range prop.Children {
				assert.Equal(t, "comp", comp.Name)
				components = append(components, comp.GetAttr("name"))
			}
			assert.ElementsMatch(t, []string{"VEVENT", "VJOURNAL", "VTODO"}, components)

		case "getctag":
			assert.Equal(t, `"example-ctag-replaced-for-testing"`, prop.TextContent)
		}
	}

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	caldav := root.SelectAttr("xmlns:C")
	calendarserver := root.SelectAttr("xmlns:CS")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.NotNil(t, calendarserver, "CalendarServer namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
	assert.Equal(t, CalendarServer, calendarserver.Value)
}

func TestCalendarMultigetRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/01_calendar_multiget_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &CalendarMultigetRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{"getetag", "calendar-data"}, req.Prop)
	assert.Equal(t, []string{"/calendars/user/calendar-1/event-1.ics"}, req.Hrefs)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "calendar-multiget", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	caldav := root.SelectAttr("xmlns:C")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

func TestCalendarMultigetResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/01_calendar_multiget_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1)
	r := resp.Responses[0]
	assert.Equal(t, "/calendars/user/calendar-1/event-1.ics", r.Href)
	assert.Len(t, r.PropStats, 1)

	// Check propstat
	propstat := r.PropStats[0]
	assert.Contains(t, propstat.Status, "200 OK")
	assert.Len(t, propstat.Props, 2)

	// Check individual properties
	for _, prop := range propstat.Props {
		switch prop.Name {
		case "getetag":
			assert.Equal(t, `"123456789abcdef123456789abcdef123456789a"`, prop.TextContent)
		case "calendar-data":
			// Verify basic iCalendar structure
			assert.Contains(t, prop.TextContent, "BEGIN:VCALENDAR")
			assert.Contains(t, prop.TextContent, "VERSION:2.0")
			assert.Contains(t, prop.TextContent, "BEGIN:VEVENT")
			assert.Contains(t, prop.TextContent, "UID:event-1")
			assert.Contains(t, prop.TextContent, "END:VEVENT")
			assert.Contains(t, prop.TextContent, "END:VCALENDAR")
		}
	}

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	t.Logf("Generated XML: %s", elementToString(root))
	t.Logf("Generated namespaces: %v", root.Attr)

	// Check for either prefixed or default DAV namespace
	dav := root.SelectAttr("xmlns:D")
	if dav == nil {
		dav = root.SelectAttr("xmlns")
	}
	caldav := root.SelectAttr("xmlns:C")

	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

func TestCalendarHomeRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/02_calendar_home_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &PropfindRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{"calendar-home-set"}, req.Prop)
	assert.False(t, req.PropNames)
	assert.False(t, req.AllProp)
	assert.Empty(t, req.Include)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "propfind", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	caldav := root.SelectAttr("xmlns:C")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

func TestCalendarHomeResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/02_calendar_home_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1)
	r := resp.Responses[0]
	assert.Equal(t, "/calendars/user/", r.Href)
	assert.Len(t, r.PropStats, 1)

	// Check propstat
	propstat := r.PropStats[0]
	assert.Contains(t, propstat.Status, "200 OK")
	assert.Len(t, propstat.Props, 1)

	// Check calendar-home-set property
	prop := propstat.Props[0]
	assert.Equal(t, "calendar-home-set", prop.Name)
	assert.Len(t, prop.Children, 1)
	assert.Equal(t, "href", prop.Children[0].Name)
	assert.Equal(t, "/calendars/user/", prop.Children[0].TextContent)

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	t.Logf("Generated XML: %s", elementToString(root))
	t.Logf("Generated namespaces: %v", root.Attr)

	// Check for default DAV namespace
	dav := root.SelectAttr("xmlns")
	if dav == nil {
		dav = root.SelectAttr("xmlns:D")
	}
	caldav := root.SelectAttr("xmlns:C")

	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

// TestAllRealData is a general test that verifies all XML files in testdata can be parsed
func TestCaldavDiscoveryRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/03_caldav_discovery_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &PropfindRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{
		"resourcetype",
		"owner",
		"displayname",
		"current-user-principal",
		"current-user-privilege-set",
		"calendar-color",
		"calendar-home-set",
	}, req.Prop)
	assert.False(t, req.PropNames)
	assert.False(t, req.AllProp)
	assert.Empty(t, req.Include)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "propfind", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	apple := root.SelectAttr("xmlns:A")
	caldav := root.SelectAttr("xmlns:C")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, apple, "Apple namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, AppleICal, apple.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

func TestCaldavDiscoveryResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/discovery/03_caldav_discovery_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1)
	r := resp.Responses[0]
	assert.Equal(t, "/user/", r.Href)
	assert.Len(t, r.PropStats, 1)

	// Check propstat
	propstat := r.PropStats[0]
	assert.Contains(t, propstat.Status, "200 OK")
	assert.Len(t, propstat.Props, 1)

	// Check calendar-home-set property
	prop := propstat.Props[0]
	assert.Equal(t, "calendar-home-set", prop.Name)
	assert.Len(t, prop.Children, 1)
	assert.Equal(t, "href", prop.Children[0].Name)
	assert.Equal(t, "/user/", prop.Children[0].TextContent)

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)
}

func TestCalendarSyncRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/03_calendar_sync_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &SyncCollectionRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Empty(t, req.SyncToken)
	assert.Equal(t, "1", req.SyncLevel)
	assert.ElementsMatch(t, []string{"getcontenttype", "getetag"}, req.Prop)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "sync-collection", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
}

func TestCalendarSyncResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/03_calendar_sync_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Equal(t, "http://radicale.org/ns/sync/test-token", resp.SyncToken)
	assert.Len(t, resp.Responses, 1)

	r := resp.Responses[0]
	assert.Equal(t, "/user/calendar1/event1.ics", r.Href)
	assert.Len(t, r.PropStats, 1)

	propstat := r.PropStats[0]
	assert.Contains(t, propstat.Status, "200 OK")
	assert.Len(t, propstat.Props, 2)

	// Check properties
	for _, prop := range propstat.Props {
		switch prop.Name {
		case "getcontenttype":
			assert.Equal(t, "text/calendar;charset=utf-8;component=VEVENT", prop.TextContent)
		case "getetag":
			assert.Equal(t, `"event1-etag"`, prop.TextContent)
		}
	}

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)
}

func TestCalendarMultigetMultipleRequest(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/04_calendar_multiget_multiple_request.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	req := &CalendarMultigetRequest{}
	err = req.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.ElementsMatch(t, []string{"getetag", "calendar-data"}, req.Prop)
	expectedHrefs := []string{
		"/user/calendar1/event1.ics",
		"/user/calendar1/event2.ics",
		"/user/calendar1/event3.ics",
	}
	assert.Equal(t, expectedHrefs, req.Hrefs)

	// Test generation
	generated := req.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "calendar-multiget", generated.Root().Tag)

	// Verify generated namespaces
	root := generated.Root()
	dav := root.SelectAttr("xmlns:D")
	caldav := root.SelectAttr("xmlns:C")
	assert.NotNil(t, dav, "DAV namespace should be present")
	assert.NotNil(t, caldav, "CalDAV namespace should be present")
	assert.Equal(t, DAV, dav.Value)
	assert.Equal(t, CalDAV, caldav.Value)
}

func TestCalendarMultigetMultipleResponse(t *testing.T) {
	// Read test file
	doc := etree.NewDocument()
	file, err := os.ReadFile("testdata/events/04_calendar_multiget_multiple_response.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReadFromBytes(file); err != nil {
		t.Fatal(err)
	}

	// Test parsing
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 3)

	// Verify each response
	for i, r := range resp.Responses {
		assert.Equal(t, fmt.Sprintf("/user/calendar1/event%d.ics", i+1), r.Href)
		assert.Len(t, r.PropStats, 1)

		propstat := r.PropStats[0]
		assert.Contains(t, propstat.Status, "200 OK")
		assert.Len(t, propstat.Props, 2)

		// Check individual properties
		for _, prop := range propstat.Props {
			switch prop.Name {
			case "getetag":
				assert.Equal(t, fmt.Sprintf(`"event%d-etag"`, i+1), prop.TextContent)
			case "calendar-data":
				assert.Contains(t, prop.TextContent, "BEGIN:VCALENDAR")
				assert.Contains(t, prop.TextContent, "VERSION:2.0")
				assert.Contains(t, prop.TextContent, "BEGIN:VEVENT")
				assert.Contains(t, prop.TextContent, fmt.Sprintf("UID:event%d-uid", i+1))
				assert.Contains(t, prop.TextContent, "END:VEVENT")
				assert.Contains(t, prop.TextContent, "END:VCALENDAR")
			}
		}
	}

	// Test generation
	generated := resp.ToXML()
	assert.NotNil(t, generated)
	assert.Equal(t, "multistatus", generated.Root().Tag)
}

func TestCalendarListRequest(t *testing.T) {
// Read test file
doc := etree.NewDocument()
file, err := os.ReadFile("testdata/discovery/05_calendar_list_request.xml")
if err != nil {
t.Fatal(err)
}
if err := doc.ReadFromBytes(file); err != nil {
t.Fatal(err)
}

// Test parsing
req := &PropfindRequest{}
err = req.Parse(doc)
assert.NoError(t, err)

// Verify parsed data
assert.ElementsMatch(t, []string{
"resourcetype",
"displayname",
"current-user-privilege-set",
"calendar-color",
}, req.Prop)
assert.False(t, req.PropNames)
assert.False(t, req.AllProp)
assert.Empty(t, req.Include)

// Test generation
generated := req.ToXML()
assert.NotNil(t, generated)
assert.Equal(t, "propfind", generated.Root().Tag)

// Verify generated namespaces
root := generated.Root()
dav := root.SelectAttr("xmlns:D")
apple := root.SelectAttr("xmlns:A")
assert.NotNil(t, dav, "DAV namespace should be present")
assert.NotNil(t, apple, "Apple iCal namespace should be present")
assert.Equal(t, DAV, dav.Value)
assert.Equal(t, AppleICal, apple.Value)
}

func TestCalendarListResponse(t *testing.T) {
// Read test file
doc := etree.NewDocument()
file, err := os.ReadFile("testdata/discovery/05_calendar_list_response.xml")
if err != nil {
t.Fatal(err)
}
if err := doc.ReadFromBytes(file); err != nil {
t.Fatal(err)
}

// Test parsing
resp := &MultistatusResponse{}
err = resp.Parse(doc)
assert.NoError(t, err)

// Verify parsed data
assert.Len(t, resp.Responses, 3)

// Verify principal response
r := resp.Responses[0]
assert.Equal(t, "/user/", r.Href)
assert.Len(t, r.PropStats, 2)

// First propstat (200 OK)
assert.Contains(t, r.PropStats[0].Status, "200 OK")
assert.Len(t, r.PropStats[0].Props, 2)
foundResourceType := false
for _, prop := range r.PropStats[0].Props {
if prop.Name == "resourcetype" {
foundResourceType = true
assert.Len(t, prop.Children, 2)
types := []string{prop.Children[0].Name, prop.Children[1].Name}
assert.ElementsMatch(t, []string{"principal", "collection"}, types)
}
}
assert.True(t, foundResourceType, "resourcetype property not found")

// Second propstat (404 Not Found)
assert.Contains(t, r.PropStats[1].Status, "404 Not Found")
assert.Len(t, r.PropStats[1].Props, 2)

// Verify calendar responses
for i, r := range resp.Responses[1:] {
calNum := i + 1
assert.Equal(t, fmt.Sprintf("/user/calendar%d/", calNum), r.Href)
assert.Len(t, r.PropStats, 1)

propstat := r.PropStats[0]
assert.Contains(t, propstat.Status, "200 OK")
assert.Len(t, propstat.Props, 4)

for _, prop := range propstat.Props {
switch prop.Name {
case "resourcetype":
assert.Len(t, prop.Children, 2)
types := []string{prop.Children[0].Name, prop.Children[1].Name}
assert.ElementsMatch(t, []string{"calendar", "collection"}, types)
case "displayname":
assert.Equal(t, fmt.Sprintf("Calendar %d", calNum), prop.TextContent)
case "calendar-color":
if calNum == 1 {
assert.Equal(t, "#ff0000ff", prop.TextContent)
} else {
assert.Equal(t, "#00ff00ff", prop.TextContent)
}
}
}
}

// Test generation
generated := resp.ToXML()
assert.NotNil(t, generated)
assert.Equal(t, "multistatus", generated.Root().Tag)

// Verify generated namespaces
root := generated.Root()
dav := root.SelectAttr("xmlns:D")
caldav := root.SelectAttr("xmlns:C")
apple := root.SelectAttr("xmlns:A")
assert.NotNil(t, dav, "DAV namespace should be present")
assert.NotNil(t, caldav, "CalDAV namespace should be present")
assert.NotNil(t, apple, "Apple iCal namespace should be present")
assert.Equal(t, DAV, dav.Value)
assert.Equal(t, CalDAV, caldav.Value)
assert.Equal(t, AppleICal, apple.Value)
}

func TestAllRealData(t *testing.T) {
	err := filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".xml") {
			doc := etree.NewDocument()
			file, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read %s: %v", path, err)
				return nil
			}
			if err := doc.ReadFromBytes(file); err != nil {
				t.Errorf("Failed to parse %s: %v", path, err)
				return nil
			}

			// Try parsing based on file type
			if strings.Contains(path, "_request") {
				// Determine the type of request based on file content
				root := doc.Root()
				switch root.Tag {
				case "calendar-multiget":
					req := &CalendarMultigetRequest{}
					if err := req.Parse(doc); err != nil {
						t.Errorf("Failed to parse calendar-multiget request %s: %v", path, err)
					}
				case "sync-collection":
					req := &SyncCollectionRequest{}
					if err := req.Parse(doc); err != nil {
						t.Errorf("Failed to parse sync-collection request %s: %v", path, err)
					}
				default:
					req := &PropfindRequest{}
					if err := req.Parse(doc); err != nil {
						t.Errorf("Failed to parse propfind request %s: %v", path, err)
					}
				}
			} else if strings.Contains(path, "_response") {
				resp := &MultistatusResponse{}
				if err := resp.Parse(doc); err != nil {
					t.Errorf("Failed to parse response %s: %v", path, err)
				}
			}
		}
		return nil
	})
	assert.NoError(t, err)
}
