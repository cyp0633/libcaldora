package props

import (
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/stretchr/testify/assert"
)

// Helper function to convert element to string
func elementToString(elem *etree.Element) string {
	doc := etree.NewDocument()
	doc.AddChild(elem)
	str, _ := doc.WriteToString()
	return str
}

// Helper function to clean XML string for comparison
func cleanXMLString(s string) string {
	// Remove XML declaration if present
	if strings.HasPrefix(s, "<?xml") {
		endIndex := strings.Index(s, "?>")
		if endIndex != -1 {
			s = s[endIndex+2:]
		}
	}

	// Remove whitespace between tags
	s = strings.ReplaceAll(s, "> <", "><")

	// Remove all spaces and newlines
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")

	return s
}

func TestEncodeFunctions(t *testing.T) {
	// Test cases for each property type
	tests := []struct {
		name            string
		property        PropertyEncoder
		expectedPrefix  string // Expected namespace prefix
		expectedTag     string // Expected element tag (local name only)
		expectedContent string // Expected content or specific element structure
		hasHrefChild    bool   // Whether the property has a nested href element
	}{
		// WebDAV properties
		{
			name:            "displayName",
			property:        &DisplayName{Value: "Test Calendar"},
			expectedPrefix:  "d",
			expectedTag:     "displayname",
			expectedContent: "Test Calendar",
		},
		// ResourceType test cases for different resource types
		{
			name: "resourcetype-principal",
			property: &Resourcetype{
				Type: ResourcePrincipal,
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:principal/>",
		},
		{
			name: "resourcetype-homeset",
			property: &Resourcetype{
				Type: ResourceHomeSet,
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:collection/><cal:calendar-home-set/>",
		},
		{
			name: "resourcetype-collection",
			property: &Resourcetype{
				Type: ResourceCollection,
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:collection/><cal:calendar/>",
		},
		{
			name: "resourcetype-object-vevent",
			property: &Resourcetype{
				Type:       ResourceObject,
				ObjectType: "vevent",
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:vevent/>",
		},
		{
			name: "resourcetype-object-freebusy",
			property: &Resourcetype{
				Type:       ResourceObject,
				ObjectType: "freebusy",
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:freebusy/>",
		},
		{
			name: "resourcetype-object-schedule",
			property: &Resourcetype{
				Type:       ResourceObject,
				ObjectType: "schedule-interaction",
			},
			expectedPrefix:  "d",
			expectedTag:     "resourcetype",
			expectedContent: "<d:schedule-interaction/>",
		},
		{
			name:            "getEtag",
			property:        &GetEtag{Value: "\"2a6b327d6f32a599eb457bedb8c25c1c\""},
			expectedPrefix:  "d",
			expectedTag:     "getetag",
			expectedContent: "\"2a6b327d6f32a599eb457bedb8c25c1c\"",
		},
		{
			name:            "getLastModified",
			property:        &GetLastModified{Value: time.Date(2025, 3, 28, 14, 30, 45, 0, time.UTC)},
			expectedPrefix:  "d",
			expectedTag:     "getlastmodified",
			expectedContent: "Fri, 28 Mar 2025 14:30:45 UTC",
		},
		{
			name:            "getContentType",
			property:        &GetContentType{Value: "text/calendar"},
			expectedPrefix:  "d",
			expectedTag:     "getcontenttype",
			expectedContent: "text/calendar",
		},
		{
			name:            "owner",
			property:        &Owner{Value: "mailto:alice@example.com"},
			expectedPrefix:  "d",
			expectedTag:     "owner",
			expectedContent: "mailto:alice@example.com",
			hasHrefChild:    true,
		},
		{
			name:            "currentUserPrincipal",
			property:        &CurrentUserPrincipal{Value: "mailto:alice@example.com"},
			expectedPrefix:  "d",
			expectedTag:     "current-user-principal",
			expectedContent: "mailto:alice@example.com",
			hasHrefChild:    true,
		},
		{
			name:            "principalURL",
			property:        &PrincipalURL{Value: "/principals/users/alice/"},
			expectedPrefix:  "d",
			expectedTag:     "principal-url",
			expectedContent: "/principals/users/alice/",
			hasHrefChild:    true,
		},
		{
			name: "supportedReportSet",
			property: &SupportedReportSet{
				Reports: []ReportType{
					ReportTypePropfind,
					ReportTypeCalendarQuery,
					ReportTypeCalendarMultiget,
					ReportTypeFreebusyQuery,
					ReportTypeSearch,
				},
			},
			expectedPrefix: "d",
			expectedTag:    "supported-report-set",
			expectedContent: "<d:supported-report><d:report><d:propfind/></d:report></d:supported-report>" +
				"<d:supported-report><d:report><cal:calendar-query/></d:report></d:supported-report>" +
				"<d:supported-report><d:report><cal:calendar-multiget/></d:report></d:supported-report>" +
				"<d:supported-report><d:report><cal:free-busy-query/></d:report></d:supported-report>" +
				"<d:supported-report><d:report><d:search/></d:report></d:supported-report>",
		},
		{
			name: "acl",
			property: &ACL{
				Aces: []ACE{
					{
						Principal: "/principals/users/alice/",
						Grant:     []string{"read", "write"},
					},
				},
			},
			expectedPrefix:  "d",
			expectedTag:     "acl",
			expectedContent: "<d:ace><d:principal><d:href>/principals/users/alice/</d:href></d:principal><d:grant><d:privilege><d:read/></d:privilege><d:privilege><d:write/></d:privilege></d:grant></d:ace>",
		},
		{
			name: "currentUserPrivilegeSet",
			property: &CurrentUserPrivilegeSet{
				Privileges: []string{"read", "write"},
			},
			expectedPrefix:  "d",
			expectedTag:     "current-user-privilege-set",
			expectedContent: "<d:privilege><d:read/></d:privilege><d:privilege><d:write/></d:privilege>",
		},
		{
			name:            "quotaAvailableBytes",
			property:        &QuotaAvailableBytes{Value: 1073741824},
			expectedPrefix:  "d",
			expectedTag:     "quota-available-bytes",
			expectedContent: "1073741824",
		},
		{
			name:            "quotaUsedBytes",
			property:        &QuotaUsedBytes{Value: 214748364},
			expectedPrefix:  "d",
			expectedTag:     "quota-used-bytes",
			expectedContent: "214748364",
		},

		// Test cases for calendar-data property
		{
			name:            "calendarData-basic",
			property:        &CalendarData{ICal: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nSUMMARY:Test Event\r\nEND:VEVENT\r\nEND:VCALENDAR"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-data",
			expectedContent: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nSUMMARY:Test Event\r\nEND:VEVENT\r\nEND:VCALENDAR",
		},
		{
			name:            "calendarData-empty",
			property:        &CalendarData{ICal: ""},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-data",
			expectedContent: "",
		},
		{
			name:            "calendarData-with-special-chars",
			property:        &CalendarData{ICal: "BEGIN:VCALENDAR\r\nDESCRIPTION:Test & Demo < > \" '\r\nEND:VCALENDAR"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-data",
			expectedContent: "BEGIN:VCALENDAR\r\nDESCRIPTION:Test &amp; Demo &lt; &gt; &#34; &#39;\r\nEND:VCALENDAR",
		},

		// CalDAV properties
		{
			name:            "calendarDescription",
			property:        &CalendarDescription{Value: "My personal work calendar"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-description",
			expectedContent: "My personal work calendar",
		},
		{
			name:            "calendarTimezone",
			property:        &CalendarTimezone{Value: "UTC"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-timezone",
			expectedContent: "UTC",
		},
		{
			name: "supportedCalendarComponentSet",
			property: &SupportedCalendarComponentSet{
				Components: []string{"VEVENT", "VTODO"},
			},
			expectedPrefix:  "cal",
			expectedTag:     "supported-calendar-component-set",
			expectedContent: "<cal:compname=\"VEVENT\"/><cal:compname=\"VTODO\"/>",
		},
		{
			name: "supportedCalendarData",
			property: &SupportedCalendarData{
				ContentType: "text/calendar",
				Version:     "2.0",
			},
			expectedPrefix:  "cal",
			expectedTag:     "supported-calendar-data",
			expectedContent: "text/calendar",
		},
		{
			name:            "maxResourceSize",
			property:        &MaxResourceSize{Value: 10485760},
			expectedPrefix:  "cal",
			expectedTag:     "max-resource-size",
			expectedContent: "10485760",
		},
		{
			name:            "minDateTime",
			property:        &MinDateTime{Value: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			expectedPrefix:  "cal",
			expectedTag:     "min-date-time",
			expectedContent: "2025-01-01T00:00:00Z",
		},
		{
			name:            "maxDateTime",
			property:        &MaxDateTime{Value: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)},
			expectedPrefix:  "cal",
			expectedTag:     "max-date-time",
			expectedContent: "2025-12-31T23:59:59Z",
		},
		{
			name:            "maxInstances",
			property:        &MaxInstances{Value: 100},
			expectedPrefix:  "cal",
			expectedTag:     "max-instances",
			expectedContent: "100",
		},
		{
			name:            "maxAttendeesPerInstance",
			property:        &MaxAttendeesPerInstance{Value: 50},
			expectedPrefix:  "cal",
			expectedTag:     "max-attendees-per-instance",
			expectedContent: "50",
		},
		{
			name:            "calendarHomeSet",
			property:        &CalendarHomeSet{Href: "/calendars/users/alice/"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-home-set",
			expectedContent: "<d:href>/calendars/users/alice/</d:href>",
		},
		{
			name:            "scheduleInboxURL",
			property:        &ScheduleInboxURL{Href: "/schedules/inbox/alice/"},
			expectedPrefix:  "cal",
			expectedTag:     "schedule-inbox-url",
			expectedContent: "<d:href>/schedules/inbox/alice/</d:href>",
		},
		{
			name:            "scheduleOutboxURL",
			property:        &ScheduleOutboxURL{Href: "/schedules/outbox/alice/"},
			expectedPrefix:  "cal",
			expectedTag:     "schedule-outbox-url",
			expectedContent: "<d:href>/schedules/outbox/alice/</d:href>",
		},
		{
			name:            "scheduleDefaultCalendarURL",
			property:        &ScheduleDefaultCalendarURL{Href: "/calendars/users/alice/work-calendar/"},
			expectedPrefix:  "cal",
			expectedTag:     "schedule-default-calendar-url",
			expectedContent: "<d:href>/calendars/users/alice/work-calendar/</d:href>",
		},
		{
			name: "calendarUserAddressSet",
			property: &CalendarUserAddressSet{
				Addresses: []string{"mailto:alice@example.com"},
			},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-user-address-set",
			expectedContent: "<d:href>mailto:alice@example.com</d:href>",
		},
		{
			name:            "calendarUserType",
			property:        &CalendarUserType{Value: "individual"},
			expectedPrefix:  "cal",
			expectedTag:     "calendar-user-type",
			expectedContent: "individual",
		},

		// Apple CalendarServer Extensions
		{
			name:            "getCTag",
			property:        &GetCTag{Value: "f7c7abdf2cb5f8c6d2f8d6bd6c71f8d3"},
			expectedPrefix:  "cs",
			expectedTag:     "getctag",
			expectedContent: "f7c7abdf2cb5f8c6d2f8d6bd6c71f8d3",
		},
		{
			name:            "calendarChanges",
			property:        &CalendarChanges{Href: "/calendars/users/alice/work-calendar/"},
			expectedPrefix:  "cs",
			expectedTag:     "calendar-changes",
			expectedContent: "<d:href>/calendars/users/alice/work-calendar/</d:href>",
		},
		{
			name:            "sharedURL",
			property:        &SharedURL{Value: "https://example.com/shared/xyz123"},
			expectedPrefix:  "cs",
			expectedTag:     "shared-url",
			expectedContent: "https://example.com/shared/xyz123",
			hasHrefChild:    true,
		},
		{
			name:            "invite",
			property:        &Invite{Value: "https://example.com/invite/abc456"},
			expectedPrefix:  "cs",
			expectedTag:     "invite",
			expectedContent: "https://example.com/invite/abc456",
		},
		{
			name:            "notificationURL",
			property:        &NotificationURL{Value: "https://example.com/notify/alice"},
			expectedPrefix:  "cs",
			expectedTag:     "notification-url",
			expectedContent: "https://example.com/notify/alice",
			hasHrefChild:    true,
		},
		{
			name:            "autoSchedule",
			property:        &AutoSchedule{Value: false},
			expectedPrefix:  "cs",
			expectedTag:     "auto-schedule",
			expectedContent: "false",
		},
		{
			name: "calendarProxyReadFor",
			property: &CalendarProxyReadFor{
				Hrefs: []string{"mailto:manager@example.com"},
			},
			expectedPrefix:  "cs",
			expectedTag:     "calendar-proxy-read-for",
			expectedContent: "<d:href>mailto:manager@example.com</d:href>",
		},
		{
			name: "calendarProxyWriteFor",
			property: &CalendarProxyWriteFor{
				Hrefs: []string{"mailto:assistant@example.com"},
			},
			expectedPrefix:  "cs",
			expectedTag:     "calendar-proxy-write-for",
			expectedContent: "<d:href>mailto:assistant@example.com</d:href>",
		},
		{
			name:            "calendarColor",
			property:        &CalendarColor{Value: "#FF5733"},
			expectedPrefix:  "cs",
			expectedTag:     "calendar-color",
			expectedContent: "#FF5733",
		},

		// Google CalDAV Extensions
		{
			name:            "color",
			property:        &Color{Value: "#FF5733"},
			expectedPrefix:  "g",
			expectedTag:     "color",
			expectedContent: "#FF5733",
		},
		{
			name:            "timezone",
			property:        &Timezone{Value: "America/New_York"},
			expectedPrefix:  "g",
			expectedTag:     "timezone",
			expectedContent: "America/New_York",
		},
		{
			name:            "hidden",
			property:        &Hidden{Value: false},
			expectedPrefix:  "g",
			expectedTag:     "hidden",
			expectedContent: "false",
		},
		{
			name:            "selected",
			property:        &Selected{Value: true},
			expectedPrefix:  "g",
			expectedTag:     "selected",
			expectedContent: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get encoded element
			elem := tt.property.Encode()

			// Check element namespace prefix and local tag name separately
			assert.Equal(t, tt.expectedPrefix, elem.Space, "Element prefix should be %s, got %s", tt.expectedPrefix, elem.Space)
			assert.Equal(t, tt.expectedTag, elem.Tag, "Element tag should be %s, got %s", tt.expectedTag, elem.Tag)

			// For simple text content
			if !strings.Contains(tt.expectedContent, "<") {
				if tt.hasHrefChild {
					// Check text in href child element for URL properties
					hrefElem := elem.FindElement("./d:href")
					assert.NotNil(t, hrefElem, "Element should have href child element")
					assert.Equal(t, tt.expectedContent, hrefElem.Text(), "href element content should be %s, got %s", tt.expectedContent, hrefElem.Text())
				} else {
					// Check text directly on element
					assert.Equal(t, tt.expectedContent, elem.Text(), "Element content should be %s, got %s", tt.expectedContent, elem.Text())
				}
			} else {
				// For complex element structure, convert element to string and check content
				// We need to clean up whitespace to make comparison reliable
				elementStr := cleanXMLString(elementToString(elem))
				expectedStr := cleanXMLString(tt.expectedContent)
				assert.Contains(t, elementStr, expectedStr, "Element should contain %s in its structure", expectedStr)
			}

			// For specific attribute checks (only for supported-calendar-data that has a version attribute)
			if tt.name == "supportedCalendarData" {
				assert.Equal(t, "2.0", elem.SelectAttrValue("version", ""), "Element should have version attribute set to 2.0")
			}
		})
	}
}

// Test the full encode/decode cycle for a complex property
func TestEncodeDecodeCycle(t *testing.T) {
	// Create a complex property with nested structure
	original := &ACL{
		Aces: []ACE{
			{
				Principal: "/principals/users/alice/",
				Grant:     []string{"read", "write"},
			},
			{
				Principal: "/principals/users/bob/",
				Deny:      []string{"write"},
			},
		},
	}

	// Encode it
	encoded := original.Encode()

	// Convert to XML string for validation
	xmlStr := elementToString(encoded)

	// Validate the structure
	assert.Contains(t, xmlStr, "<d:acl>")
	assert.Contains(t, xmlStr, "<d:ace>")
	assert.Contains(t, xmlStr, "<d:principal>")
	assert.Contains(t, xmlStr, "<d:href>/principals/users/alice/</d:href>")
	assert.Contains(t, xmlStr, "<d:grant>")
	assert.Contains(t, xmlStr, "<d:privilege>")
	assert.Contains(t, xmlStr, "<d:read/>")
	assert.Contains(t, xmlStr, "<d:write/>")
	assert.Contains(t, xmlStr, "<d:deny>")
}

// TestSupportedReportSetNamespaces verifies that all report types are encoded with correct namespaces
func TestSupportedReportSetNamespaces(t *testing.T) {
	// Create a SupportedReportSet with all report types
	srs := SupportedReportSet{
		Reports: []ReportType{
			ReportTypePropfind,
			ReportTypeCalendarQuery,
			ReportTypeCalendarMultiget,
			ReportTypeFreebusyQuery,
			ReportTypeScheduleQuery,
			ReportTypeScheduleMultiget,
			ReportTypeSearch,
		},
	}

	// Encode to XML
	elem := srs.Encode()

	// Basic validation
	assert.Equal(t, "d", elem.Space, "Root element should have DAV namespace")
	assert.Equal(t, "supported-report-set", elem.Tag, "Root element should be supported-report-set")

	// Find all the supported-report elements
	supportedReports := elem.ChildElements()
	assert.Equal(t, 7, len(supportedReports), "Should have 7 supported-report elements")

	// Expected namespace prefixes for each report type
	expectedPrefixes := map[ReportType]string{
		ReportTypePropfind:         "d",
		ReportTypeCalendarQuery:    "cal",
		ReportTypeCalendarMultiget: "cal",
		ReportTypeFreebusyQuery:    "cal",
		ReportTypeScheduleQuery:    "cal",
		ReportTypeScheduleMultiget: "cal",
		ReportTypeSearch:           "d",
	}

	// Expected tag names for each report type
	expectedTags := map[ReportType]string{
		ReportTypePropfind:         "propfind",
		ReportTypeCalendarQuery:    "calendar-query",
		ReportTypeCalendarMultiget: "calendar-multiget",
		ReportTypeFreebusyQuery:    "free-busy-query",
		ReportTypeScheduleQuery:    "schedule-query",
		ReportTypeScheduleMultiget: "schedule-multiget",
		ReportTypeSearch:           "search",
	}

	// Verify each report type has the correct structure and namespace
	reportCount := make(map[string]int)

	for _, supportedReport := range supportedReports {
		// Check structure: supported-report -> report -> specific report type
		assert.Equal(t, "supported-report", supportedReport.Tag)
		assert.Equal(t, "d", supportedReport.Space)

		// Check report element
		report := supportedReport.SelectElement("d:report")
		assert.NotNil(t, report, "Each supported-report should contain a report element")

		// Get the specific report type (e.g. propfind, calendar-query)
		reportTypes := report.ChildElements()
		assert.Equal(t, 1, len(reportTypes), "Each report should have exactly one report type child")

		reportType := reportTypes[0]
		reportCount[reportType.Tag]++

		// Verify the namespace for this report type
		for rt, expectedTag := range expectedTags {
			if reportType.Tag == expectedTag {
				expectedPrefix := expectedPrefixes[rt]
				assert.Equal(t, expectedPrefix, reportType.Space,
					"Report type %s should have namespace prefix %s", reportType.Tag, expectedPrefix)
			}
		}
	}

	// Verify we have one of each report type
	for _, tag := range expectedTags {
		assert.Equal(t, 1, reportCount[tag], "Should have exactly one %s report type", tag)
	}
}

// Create a map of property name to struct for testing
var propNameToStruct = map[string]PropertyEncoder{
	// WebDAV properties
	"displayname":                &DisplayName{Value: "Test Calendar"},
	"resourcetype":               &Resourcetype{Type: ResourceCollection},
	"getetag":                    &GetEtag{Value: "abc123"},
	"getlastmodified":            &GetLastModified{Value: time.Now()},
	"getcontenttype":             &GetContentType{Value: "text/calendar"},
	"owner":                      &Owner{Value: "mailto:user@example.com"},
	"current-user-principal":     &CurrentUserPrincipal{Value: "/principals/users/johndoe/"},
	"principal-url":              &PrincipalURL{Value: "/principals/users/johndoe/"},
	"supported-report-set":       &SupportedReportSet{Reports: []ReportType{ReportTypePropfind}},
	"acl":                        &ACL{Aces: []ACE{{Principal: "/principals/users/johndoe/", Grant: []string{"read"}}}},
	"current-user-privilege-set": &CurrentUserPrivilegeSet{Privileges: []string{"read"}},
	"quota-available-bytes":      &QuotaAvailableBytes{Value: 1000000},
	"quota-used-bytes":           &QuotaUsedBytes{Value: 5000},

	// CalDAV properties
	"calendar-description":             &CalendarDescription{Value: "My calendar"},
	"calendar-timezone":                &CalendarTimezone{Value: "UTC"},
	"calendar-data":                    &CalendarData{ICal: "BEGIN:VCALENDAR\r\nEND:VCALENDAR"},
	"supported-calendar-component-set": &SupportedCalendarComponentSet{Components: []string{"VEVENT"}},
	"supported-calendar-data":          &SupportedCalendarData{ContentType: "text/calendar", Version: "2.0"},
	"max-resource-size":                &MaxResourceSize{Value: 10485760},
	"min-date-time":                    &MinDateTime{Value: time.Now()},
	"max-date-time":                    &MaxDateTime{Value: time.Now().AddDate(1, 0, 0)},
	"max-instances":                    &MaxInstances{Value: 100},
	"max-attendees-per-instance":       &MaxAttendeesPerInstance{Value: 50},
	"calendar-home-set":                &CalendarHomeSet{Href: "/calendars/users/johndoe/"},
	"schedule-inbox-url":               &ScheduleInboxURL{Href: "/calendars/users/johndoe/inbox/"},
	"schedule-outbox-url":              &ScheduleOutboxURL{Href: "/calendars/users/johndoe/outbox/"},
	"schedule-default-calendar-url":    &ScheduleDefaultCalendarURL{Href: "/calendars/users/johndoe/default/"},
	"calendar-user-address-set":        &CalendarUserAddressSet{Addresses: []string{"mailto:johndoe@example.com"}},
	"calendar-user-type":               &CalendarUserType{Value: "INDIVIDUAL"},

	// Apple CalendarServer Extensions
	"getctag":                  &GetCTag{Value: "abc123"},
	"calendar-changes":         &CalendarChanges{Href: "/calendars/users/johndoe/changes/"},
	"shared-url":               &SharedURL{Value: "https://example.com/shared"},
	"invite":                   &Invite{Value: "https://example.com/invite"},
	"notification-url":         &NotificationURL{Value: "https://example.com/notify"},
	"auto-schedule":            &AutoSchedule{Value: true},
	"calendar-proxy-read-for":  &CalendarProxyReadFor{Hrefs: []string{"mailto:manager@example.com"}},
	"calendar-proxy-write-for": &CalendarProxyWriteFor{Hrefs: []string{"mailto:assistant@example.com"}},
	"calendar-color":           &CalendarColor{Value: "#FF5733"},

	// Google CalDAV Extensions
	"color":    &Color{Value: "#3366CC"},
	"timezone": &Timezone{Value: "America/New_York"},
	"hidden":   &Hidden{Value: false},
	"selected": &Selected{Value: true},
}

// Test that all properties in propNameToStruct map have a working Encode method
func TestAllPropertiesEncode(t *testing.T) {
	for propName, propValue := range propNameToStruct {
		t.Run(propName, func(t *testing.T) {
			// Call Encode method
			elem := propValue.Encode()

			// Basic validation
			assert.NotNil(t, elem, "Encoded element for %s should not be nil", propName)

			// Check that tag has correct prefix
			prefix := PropPrefixMap[propName]
			assert.Equal(t, prefix, elem.Space, "Element prefix should be %s, got %s", prefix, elem.Space)
			assert.Equal(t, propName, elem.Tag, "Element tag should be %s, got %s", propName, elem.Tag)

			// Convert to string to ensure it's valid XML
			xmlStr := elementToString(elem)
			assert.NotEmpty(t, xmlStr, "XML string for %s should not be empty", propName)
		})
	}
}
