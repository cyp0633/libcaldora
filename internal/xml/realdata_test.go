package xml

import (
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
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1)
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
	resp := &MultistatusResponse{}
	err = resp.Parse(doc)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Len(t, resp.Responses, 1)
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

// TestAllRealData is a general test that verifies all XML files in testdata can be parsed
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
				req := &PropfindRequest{}
				if err := req.Parse(doc); err != nil {
					t.Errorf("Failed to parse request %s: %v", path, err)
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
