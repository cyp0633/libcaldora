package props

import (
	"fmt"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/stretchr/testify/assert"
)

// Helper function to create an XML element for testing
func createTestElement(prefix, name, content string, attrs map[string]string) *etree.Element {
	elem := etree.NewElement(name)
	if prefix != "" {
		elem.Space = prefix
	}

	if content != "" {
		elem.SetText(content)
	}

	for key, value := range attrs {
		elem.CreateAttr(key, value)
	}

	return elem
}

// Helper function to create an element with a child href element
func createElementWithHrefChild(prefix, name, hrefContent string) *etree.Element {
	elem := etree.NewElement(name)
	if prefix != "" {
		elem.Space = prefix
	}

	href := etree.NewElement("href")
	href.Space = "d"
	href.SetText(hrefContent)
	elem.AddChild(href)

	return elem
}

func TestCalendarPropsDecodeFunctions(t *testing.T) {
	// Test cases for calendar properties
	tests := []struct {
		name     string
		element  *etree.Element
		property Property
		expected interface{}
	}{
		{
			name:     "CalendarDescription",
			element:  createTestElement("cal", "calendar-description", "My Work Calendar", nil),
			property: &CalendarDescription{},
			expected: "My Work Calendar",
		},
		{
			name:     "CalendarTimezone",
			element:  createTestElement("cal", "calendar-timezone", "America/New_York", nil),
			property: &CalendarTimezone{},
			expected: "America/New_York",
		},
		{
			name: "CalendarData",
			element: createTestElement("cal", "calendar-data",
				"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR", nil),
			property: &CalendarData{},
			expected: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR",
		},
		{
			name: "SupportedCalendarComponentSet",
			element: func() *etree.Element {
				elem := etree.NewElement("supported-calendar-component-set")
				elem.Space = "cal"

				comp1 := etree.NewElement("comp")
				comp1.Space = "cal"
				comp1.CreateAttr("name", "VEVENT")
				elem.AddChild(comp1)

				comp2 := etree.NewElement("comp")
				comp2.Space = "cal"
				comp2.CreateAttr("name", "VTODO")
				elem.AddChild(comp2)

				return elem
			}(),
			property: &SupportedCalendarComponentSet{},
			expected: []string{"VEVENT", "VTODO"},
		},
		{
			name: "SupportedCalendarData",
			element: func() *etree.Element {
				elem := etree.NewElement("supported-calendar-data")
				elem.Space = "cal"
				elem.SetText("text/calendar")
				elem.CreateAttr("version", "2.0")
				return elem
			}(),
			property: &SupportedCalendarData{},
			expected: map[string]string{
				"ContentType": "text/calendar",
				"Version":     "2.0",
			},
		},
		{
			name:     "MaxResourceSize",
			element:  createTestElement("cal", "max-resource-size", "10485760", nil),
			property: &MaxResourceSize{},
			expected: int64(10485760),
		},
		{
			name:     "MinDateTime",
			element:  createTestElement("cal", "min-date-time", "2023-01-01T00:00:00Z", nil),
			property: &MinDateTime{},
			expected: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "MaxDateTime",
			element:  createTestElement("cal", "max-date-time", "2025-12-31T23:59:59Z", nil),
			property: &MaxDateTime{},
			expected: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:     "MaxInstances",
			element:  createTestElement("cal", "max-instances", "1000", nil),
			property: &MaxInstances{},
			expected: 1000,
		},
		{
			name:     "MaxAttendeesPerInstance",
			element:  createTestElement("cal", "max-attendees-per-instance", "50", nil),
			property: &MaxAttendeesPerInstance{},
			expected: 50,
		},
		{
			name:     "CalendarHomeSet",
			element:  createElementWithHrefChild("cal", "calendar-home-set", "/calendars/users/alice/"),
			property: &CalendarHomeSet{},
			expected: "/calendars/users/alice/",
		},
		{
			name:     "ScheduleInboxURL",
			element:  createElementWithHrefChild("cal", "schedule-inbox-url", "/calendars/users/alice/inbox/"),
			property: &ScheduleInboxURL{},
			expected: "/calendars/users/alice/inbox/",
		},
		{
			name:     "ScheduleOutboxURL",
			element:  createElementWithHrefChild("cal", "schedule-outbox-url", "/calendars/users/alice/outbox/"),
			property: &ScheduleOutboxURL{},
			expected: "/calendars/users/alice/outbox/",
		},
		{
			name:     "ScheduleDefaultCalendarURL",
			element:  createElementWithHrefChild("cal", "schedule-default-calendar-url", "/calendars/users/alice/calendar/"),
			property: &ScheduleDefaultCalendarURL{},
			expected: "/calendars/users/alice/calendar/",
		},
		{
			name: "CalendarUserAddressSet",
			element: func() *etree.Element {
				elem := etree.NewElement("calendar-user-address-set")
				elem.Space = "cal"

				href1 := etree.NewElement("href")
				href1.Space = "d"
				href1.SetText("mailto:alice@example.com")
				elem.AddChild(href1)

				href2 := etree.NewElement("href")
				href2.Space = "d"
				href2.SetText("https://example.com/alice")
				elem.AddChild(href2)

				return elem
			}(),
			property: &CalendarUserAddressSet{},
			expected: []string{"mailto:alice@example.com", "https://example.com/alice"},
		},
		{
			name:     "CalendarUserType",
			element:  createTestElement("cal", "calendar-user-type", "individual", nil),
			property: &CalendarUserType{},
			expected: "individual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode the element into the property
			err := tt.property.Decode(tt.element)
			assert.NoError(t, err)

			// Check the decoded value matches what we expect
			switch prop := tt.property.(type) {
			case *CalendarDescription:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *CalendarTimezone:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *CalendarData:
				assert.Equal(t, tt.expected.(string), prop.ICal)
			case *SupportedCalendarComponentSet:
				assert.ElementsMatch(t, tt.expected.([]string), prop.Components)
			case *SupportedCalendarData:
				expectedMap := tt.expected.(map[string]string)
				assert.Equal(t, expectedMap["ContentType"], prop.ContentType)
				assert.Equal(t, expectedMap["Version"], prop.Version)
			case *MaxResourceSize:
				assert.Equal(t, tt.expected.(int64), prop.Value)
			case *MinDateTime:
				assert.Equal(t, tt.expected.(time.Time), prop.Value)
			case *MaxDateTime:
				assert.Equal(t, tt.expected.(time.Time), prop.Value)
			case *MaxInstances:
				assert.Equal(t, tt.expected.(int), prop.Value)
			case *MaxAttendeesPerInstance:
				assert.Equal(t, tt.expected.(int), prop.Value)
			case *CalendarHomeSet:
				assert.Equal(t, tt.expected.(string), prop.Href)
			case *ScheduleInboxURL:
				assert.Equal(t, tt.expected.(string), prop.Href)
			case *ScheduleOutboxURL:
				assert.Equal(t, tt.expected.(string), prop.Href)
			case *ScheduleDefaultCalendarURL:
				assert.Equal(t, tt.expected.(string), prop.Href)
			case *CalendarUserAddressSet:
				assert.ElementsMatch(t, tt.expected.([]string), prop.Addresses)
			case *CalendarUserType:
				assert.Equal(t, tt.expected.(string), prop.Value)
			default:
				t.Fatalf("Unexpected property type: %T", prop)
			}
		})
	}
}

// Test error handling in decoders
func TestCalendarPropsDecodeErrors(t *testing.T) {
	// Test cases with invalid input that should result in errors
	tests := []struct {
		name     string
		element  *etree.Element
		property Property
	}{
		{
			name:     "MaxResourceSize_InvalidNumber",
			element:  createTestElement("cal", "max-resource-size", "not-a-number", nil),
			property: &MaxResourceSize{},
		},
		{
			name:     "MinDateTime_InvalidDate",
			element:  createTestElement("cal", "min-date-time", "not-a-date", nil),
			property: &MinDateTime{},
		},
		{
			name:     "MaxDateTime_InvalidDate",
			element:  createTestElement("cal", "max-date-time", "not-a-date", nil),
			property: &MaxDateTime{},
		},
		{
			name:     "MaxInstances_InvalidNumber",
			element:  createTestElement("cal", "max-instances", "not-a-number", nil),
			property: &MaxInstances{},
		},
		{
			name:     "MaxAttendeesPerInstance_InvalidNumber",
			element:  createTestElement("cal", "max-attendees-per-instance", "not-a-number", nil),
			property: &MaxAttendeesPerInstance{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Attempt to decode and expect an error
			err := tt.property.Decode(tt.element)
			assert.Error(t, err, fmt.Sprintf("%s should return an error with invalid input", tt.name))
		})
	}
}

// Test decode-encode round trip
func TestCalendarDecodeEncodeCycle(t *testing.T) {
	// Initialize with test data
	originalProperties := []Property{
		&CalendarDescription{Value: "My Work Calendar"},
		&CalendarTimezone{Value: "America/New_York"},
		&CalendarData{ICal: "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR"},
		&SupportedCalendarComponentSet{Components: []string{"VEVENT", "VTODO"}},
		&SupportedCalendarData{ContentType: "text/calendar", Version: "2.0"},
		&MaxResourceSize{Value: 10485760},
		&MinDateTime{Value: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		&MaxDateTime{Value: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)},
		&MaxInstances{Value: 1000},
		&MaxAttendeesPerInstance{Value: 50},
		&CalendarHomeSet{Href: "/calendars/users/alice/"},
		&ScheduleInboxURL{Href: "/calendars/users/alice/inbox/"},
		&ScheduleOutboxURL{Href: "/calendars/users/alice/outbox/"},
		&ScheduleDefaultCalendarURL{Href: "/calendars/users/alice/calendar/"},
		&CalendarUserAddressSet{Addresses: []string{"mailto:alice@example.com", "https://example.com/alice"}},
		&CalendarUserType{Value: "individual"},
	}

	for _, original := range originalProperties {
		t.Run(fmt.Sprintf("%T", original), func(t *testing.T) {
			// Encode the property
			encoded := original.Encode()

			// Create a new instance of the same property type
			var decoded Property
			switch original.(type) {
			case *CalendarDescription:
				decoded = &CalendarDescription{}
			case *CalendarTimezone:
				decoded = &CalendarTimezone{}
			case *CalendarData:
				decoded = &CalendarData{}
			case *SupportedCalendarComponentSet:
				decoded = &SupportedCalendarComponentSet{}
			case *SupportedCalendarData:
				decoded = &SupportedCalendarData{}
			case *MaxResourceSize:
				decoded = &MaxResourceSize{}
			case *MinDateTime:
				decoded = &MinDateTime{}
			case *MaxDateTime:
				decoded = &MaxDateTime{}
			case *MaxInstances:
				decoded = &MaxInstances{}
			case *MaxAttendeesPerInstance:
				decoded = &MaxAttendeesPerInstance{}
			case *CalendarHomeSet:
				decoded = &CalendarHomeSet{}
			case *ScheduleInboxURL:
				decoded = &ScheduleInboxURL{}
			case *ScheduleOutboxURL:
				decoded = &ScheduleOutboxURL{}
			case *ScheduleDefaultCalendarURL:
				decoded = &ScheduleDefaultCalendarURL{}
			case *CalendarUserAddressSet:
				decoded = &CalendarUserAddressSet{}
			case *CalendarUserType:
				decoded = &CalendarUserType{}
			default:
				t.Fatalf("Unexpected property type: %T", original)
				return
			}

			// Decode the encoded element
			err := decoded.Decode(encoded)
			assert.NoError(t, err)

			// Re-encode the decoded property
			reEncoded := decoded.Encode()

			// Convert both to strings for comparison
			originalXml := elementToString(encoded)
			reEncodedXml := elementToString(reEncoded)

			// Clean strings for comparison
			originalClean := cleanXMLString(originalXml)
			reEncodedClean := cleanXMLString(reEncodedXml)

			// Should match
			assert.Equal(t, originalClean, reEncodedClean, "Round trip encode-decode-encode should produce the same XML")
		})
	}
}

func TestWebDAVPropsDecodeFunctions(t *testing.T) {
	// Test cases for WebDAV properties
	tests := []struct {
		name     string
		element  *etree.Element
		property Property
		expected interface{}
	}{
		// Simple text properties
		{
			name:     "DisplayName",
			element:  createTestElement("d", "displayname", "My Calendar", nil),
			property: &DisplayName{},
			expected: "My Calendar",
		},
		{
			name:     "GetEtag",
			element:  createTestElement("d", "getetag", "\"abc123\"", nil),
			property: &GetEtag{},
			expected: "\"abc123\"",
		},
		{
			name:     "GetContentType",
			element:  createTestElement("d", "getcontenttype", "text/calendar; charset=utf-8", nil),
			property: &GetContentType{},
			expected: "text/calendar; charset=utf-8",
		},

		// Time properties
		{
			name:     "GetLastModified_RFC1123",
			element:  createTestElement("d", "getlastmodified", "Fri, 28 Mar 2025 14:30:45 GMT", nil),
			property: &GetLastModified{},
			expected: time.Date(2025, 3, 28, 14, 30, 45, 0, time.UTC),
		},
		{
			name:     "GetLastModified_RFC3339",
			element:  createTestElement("d", "getlastmodified", "2025-03-28T14:30:45Z", nil),
			property: &GetLastModified{},
			expected: time.Date(2025, 3, 28, 14, 30, 45, 0, time.UTC),
		},

		// Properties with href children
		{
			name:     "Owner",
			element:  createElementWithHrefChild("d", "owner", "/principals/users/alice/"),
			property: &Owner{},
			expected: "/principals/users/alice/",
		},
		{
			name:     "CurrentUserPrincipal",
			element:  createElementWithHrefChild("d", "current-user-principal", "/principals/users/alice/"),
			property: &CurrentUserPrincipal{},
			expected: "/principals/users/alice/",
		},
		{
			name:     "PrincipalURL",
			element:  createElementWithHrefChild("d", "principal-url", "/principals/users/alice/"),
			property: &PrincipalURL{},
			expected: "/principals/users/alice/",
		},

		// Numeric properties
		{
			name:     "QuotaAvailableBytes",
			element:  createTestElement("d", "quota-available-bytes", "1073741824", nil),
			property: &QuotaAvailableBytes{},
			expected: int64(1073741824),
		},
		{
			name:     "QuotaUsedBytes",
			element:  createTestElement("d", "quota-used-bytes", "536870912", nil),
			property: &QuotaUsedBytes{},
			expected: int64(536870912),
		},

		// Complex properties
		{
			name: "ResourcetypePrincipal",
			element: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = "d"
				principal := etree.NewElement("principal")
				principal.Space = "d"
				elem.AddChild(principal)
				return elem
			}(),
			property: &Resourcetype{},
			expected: ResourcePrincipal,
		},
		{
			name: "ResourcetypeHomeSet",
			element: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = "d"

				collection := etree.NewElement("collection")
				collection.Space = "d"
				elem.AddChild(collection)

				homeSet := etree.NewElement("calendar-home-set")
				homeSet.Space = "cal"
				elem.AddChild(homeSet)

				return elem
			}(),
			property: &Resourcetype{},
			expected: ResourceHomeSet,
		},
		{
			name: "ResourcetypeCollection",
			element: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = "d"

				collection := etree.NewElement("collection")
				collection.Space = "d"
				elem.AddChild(collection)

				calendar := etree.NewElement("calendar")
				calendar.Space = "cal"
				elem.AddChild(calendar)

				return elem
			}(),
			property: &Resourcetype{},
			expected: ResourceCollection,
		},
		{
			name: "ResourcetypeObject",
			element: func() *etree.Element {
				elem := etree.NewElement("resourcetype")
				elem.Space = "d"

				vevent := etree.NewElement("vevent")
				vevent.Space = "d"
				elem.AddChild(vevent)

				return elem
			}(),
			property: &Resourcetype{},
			expected: map[string]interface{}{
				"Type":       ResourceObject,
				"ObjectType": "vevent",
			},
		},
		{
			name: "SupportedReportSet",
			element: func() *etree.Element {
				elem := etree.NewElement("supported-report-set")
				elem.Space = "d"

				// Add propfind report
				sr1 := etree.NewElement("supported-report")
				sr1.Space = "d"
				elem.AddChild(sr1)

				report1 := etree.NewElement("report")
				report1.Space = "d"
				sr1.AddChild(report1)

				propfind := etree.NewElement("propfind")
				propfind.Space = "d"
				report1.AddChild(propfind)

				// Add calendar-query report
				sr2 := etree.NewElement("supported-report")
				sr2.Space = "d"
				elem.AddChild(sr2)

				report2 := etree.NewElement("report")
				report2.Space = "d"
				sr2.AddChild(report2)

				calQuery := etree.NewElement("calendar-query")
				calQuery.Space = "cal"
				report2.AddChild(calQuery)

				return elem
			}(),
			property: &SupportedReportSet{},
			expected: []ReportType{ReportTypePropfind, ReportTypeCalendarQuery},
		},
		{
			name: "ACL",
			element: func() *etree.Element {
				elem := etree.NewElement("acl")
				elem.Space = "d"

				ace := etree.NewElement("ace")
				ace.Space = "d"
				elem.AddChild(ace)

				principal := etree.NewElement("principal")
				principal.Space = "d"
				ace.AddChild(principal)

				href := etree.NewElement("href")
				href.Space = "d"
				href.SetText("/principals/users/alice/")
				principal.AddChild(href)

				grant := etree.NewElement("grant")
				grant.Space = "d"
				ace.AddChild(grant)

				priv1 := etree.NewElement("privilege")
				priv1.Space = "d"
				grant.AddChild(priv1)

				read := etree.NewElement("read")
				read.Space = "d"
				priv1.AddChild(read)

				priv2 := etree.NewElement("privilege")
				priv2.Space = "d"
				grant.AddChild(priv2)

				write := etree.NewElement("write")
				write.Space = "d"
				priv2.AddChild(write)

				return elem
			}(),
			property: &ACL{},
			expected: []ACE{
				{
					Principal: "/principals/users/alice/",
					Grant:     []string{"read", "write"},
					Deny:      nil,
				},
			},
		},
		{
			name: "CurrentUserPrivilegeSet",
			element: func() *etree.Element {
				elem := etree.NewElement("current-user-privilege-set")
				elem.Space = "d"

				priv1 := etree.NewElement("privilege")
				priv1.Space = "d"
				elem.AddChild(priv1)

				read := etree.NewElement("read")
				read.Space = "d"
				priv1.AddChild(read)

				priv2 := etree.NewElement("privilege")
				priv2.Space = "d"
				elem.AddChild(priv2)

				write := etree.NewElement("write")
				write.Space = "d"
				priv2.AddChild(write)

				return elem
			}(),
			property: &CurrentUserPrivilegeSet{},
			expected: []string{"read", "write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode the element into the property
			err := tt.property.Decode(tt.element)
			assert.NoError(t, err)

			// Check the decoded value matches what we expect
			switch prop := tt.property.(type) {
			case *DisplayName:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *GetEtag:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *GetContentType:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *GetLastModified:
				assert.Equal(t, tt.expected.(time.Time), prop.Value)
			case *Owner:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *CurrentUserPrincipal:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *PrincipalURL:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *QuotaAvailableBytes:
				assert.Equal(t, tt.expected.(int64), prop.Value)
			case *QuotaUsedBytes:
				assert.Equal(t, tt.expected.(int64), prop.Value)
			case *Resourcetype:
				if rt, ok := tt.expected.(ResourceType); ok {
					assert.Equal(t, rt, prop.Type)
				} else {
					expectedMap := tt.expected.(map[string]interface{})
					assert.Equal(t, expectedMap["Type"].(ResourceType), prop.Type)
					assert.Equal(t, expectedMap["ObjectType"].(string), prop.ObjectType)
				}
			case *SupportedReportSet:
				assert.ElementsMatch(t, tt.expected.([]ReportType), prop.Reports)
			case *ACL:
				expectedAces := tt.expected.([]ACE)
				assert.Len(t, prop.Aces, len(expectedAces))
				for i, expectedAce := range expectedAces {
					assert.Equal(t, expectedAce.Principal, prop.Aces[i].Principal)
					assert.ElementsMatch(t, expectedAce.Grant, prop.Aces[i].Grant)
					if expectedAce.Deny == nil {
						assert.Empty(t, prop.Aces[i].Deny)
					} else {
						assert.ElementsMatch(t, expectedAce.Deny, prop.Aces[i].Deny)
					}
				}
			case *CurrentUserPrivilegeSet:
				assert.ElementsMatch(t, tt.expected.([]string), prop.Privileges)
			default:
				t.Fatalf("Unexpected property type: %T", prop)
			}
		})
	}
}

// Test error handling in WebDAV property decoders
func TestWebDAVPropsDecodeErrors(t *testing.T) {
	// Test cases with invalid input that should result in errors
	tests := []struct {
		name     string
		element  *etree.Element
		property Property
	}{
		{
			name:     "GetLastModified_InvalidDate",
			element:  createTestElement("d", "getlastmodified", "not-a-valid-date", nil),
			property: &GetLastModified{},
		},
		{
			name:     "QuotaAvailableBytes_InvalidNumber",
			element:  createTestElement("d", "quota-available-bytes", "not-a-number", nil),
			property: &QuotaAvailableBytes{},
		},
		{
			name:     "QuotaUsedBytes_InvalidNumber",
			element:  createTestElement("d", "quota-used-bytes", "not-a-number", nil),
			property: &QuotaUsedBytes{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Attempt to decode and expect an error
			err := tt.property.Decode(tt.element)
			assert.Error(t, err, fmt.Sprintf("%s should return an error with invalid input", tt.name))
		})
	}
}

// Test decode-encode round trip for WebDAV properties
func TestWebDAVDecodeEncodeCycle(t *testing.T) {
	// Initialize with test data
	originalProperties := []Property{
		&DisplayName{Value: "My Calendar"},
		&Resourcetype{Type: ResourcePrincipal},
		&Resourcetype{Type: ResourceHomeSet},
		&Resourcetype{Type: ResourceCollection},
		&Resourcetype{Type: ResourceObject, ObjectType: "vevent"},
		&GetEtag{Value: "\"abc123\""},
		&GetLastModified{Value: time.Date(2025, 3, 28, 14, 30, 45, 0, time.UTC)},
		&GetContentType{Value: "text/calendar; charset=utf-8"},
		&Owner{Value: "/principals/users/alice/"},
		&CurrentUserPrincipal{Value: "/principals/users/alice/"},
		&PrincipalURL{Value: "/principals/users/alice/"},
		&SupportedReportSet{Reports: []ReportType{ReportTypePropfind, ReportTypeCalendarQuery}},
		&ACL{Aces: []ACE{{Principal: "/principals/users/alice/", Grant: []string{"read", "write"}}}},
		&CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}},
		&QuotaAvailableBytes{Value: 1073741824},
		&QuotaUsedBytes{Value: 536870912},
	}

	for _, original := range originalProperties {
		t.Run(fmt.Sprintf("%T", original), func(t *testing.T) {
			// Encode the property
			encoded := original.Encode()

			// Create a new instance of the same property type
			var decoded Property
			switch original.(type) {
			case *DisplayName:
				decoded = &DisplayName{}
			case *Resourcetype:
				decoded = &Resourcetype{}
			case *GetEtag:
				decoded = &GetEtag{}
			case *GetLastModified:
				decoded = &GetLastModified{}
			case *GetContentType:
				decoded = &GetContentType{}
			case *Owner:
				decoded = &Owner{}
			case *CurrentUserPrincipal:
				decoded = &CurrentUserPrincipal{}
			case *PrincipalURL:
				decoded = &PrincipalURL{}
			case *SupportedReportSet:
				decoded = &SupportedReportSet{}
			case *ACL:
				decoded = &ACL{}
			case *CurrentUserPrivilegeSet:
				decoded = &CurrentUserPrivilegeSet{}
			case *QuotaAvailableBytes:
				decoded = &QuotaAvailableBytes{}
			case *QuotaUsedBytes:
				decoded = &QuotaUsedBytes{}
			default:
				t.Fatalf("Unexpected property type: %T", original)
				return
			}

			// Decode the encoded element
			err := decoded.Decode(encoded)
			assert.NoError(t, err)

			// Re-encode the decoded property
			reEncoded := decoded.Encode()

			// Convert both to strings for comparison
			originalXml := elementToString(encoded)
			reEncodedXml := elementToString(reEncoded)

			// Clean strings for comparison
			originalClean := cleanXMLString(originalXml)
			reEncodedClean := cleanXMLString(reEncodedXml)

			// Should match
			assert.Equal(t, originalClean, reEncodedClean, "Round trip encode-decode-encode should produce the same XML")
		})
	}
}

func TestExtensionPropsDecodeFunctions(t *testing.T) {
	// Test cases for extension properties
	tests := []struct {
		name     string
		element  *etree.Element
		property Property
		expected interface{}
	}{
		// Apple CalendarServer Extensions
		{
			name:     "GetCTag",
			element:  createTestElement("cs", "getctag", "abc123xyz789", nil),
			property: &GetCTag{},
			expected: "abc123xyz789",
		},
		{
			name:     "CalendarChanges",
			element:  createElementWithHrefChild("cs", "calendar-changes", "/calendars/users/alice/changes/"),
			property: &CalendarChanges{},
			expected: "/calendars/users/alice/changes/",
		},
		{
			name:     "SharedURL",
			element:  createElementWithHrefChild("cs", "shared-url", "https://example.com/shared/calendar/"),
			property: &SharedURL{},
			expected: "https://example.com/shared/calendar/",
		},
		{
			name:     "Invite",
			element:  createTestElement("cs", "invite", "invitetoken123", nil),
			property: &Invite{},
			expected: "invitetoken123",
		},
		{
			name:     "NotificationURL",
			element:  createElementWithHrefChild("cs", "notification-url", "/calendars/users/alice/notifications/"),
			property: &NotificationURL{},
			expected: "/calendars/users/alice/notifications/",
		},
		{
			name:     "AutoSchedule_True",
			element:  createTestElement("cs", "auto-schedule", "true", nil),
			property: &AutoSchedule{},
			expected: true,
		},
		{
			name:     "AutoSchedule_False",
			element:  createTestElement("cs", "auto-schedule", "false", nil),
			property: &AutoSchedule{},
			expected: false,
		},
		{
			name:     "AutoSchedule_1",
			element:  createTestElement("cs", "auto-schedule", "1", nil),
			property: &AutoSchedule{},
			expected: true,
		},
		{
			name:     "AutoSchedule_0",
			element:  createTestElement("cs", "auto-schedule", "0", nil),
			property: &AutoSchedule{},
			expected: false,
		},
		{
			name: "CalendarProxyReadFor",
			element: func() *etree.Element {
				elem := etree.NewElement("calendar-proxy-read-for")
				elem.Space = "cs"

				href1 := etree.NewElement("href")
				href1.Space = "d"
				href1.SetText("/principals/users/manager/")
				elem.AddChild(href1)

				href2 := etree.NewElement("href")
				href2.Space = "d"
				href2.SetText("/principals/users/admin/")
				elem.AddChild(href2)

				return elem
			}(),
			property: &CalendarProxyReadFor{},
			expected: []string{"/principals/users/manager/", "/principals/users/admin/"},
		},
		{
			name: "CalendarProxyWriteFor",
			element: func() *etree.Element {
				elem := etree.NewElement("calendar-proxy-write-for")
				elem.Space = "cs"

				href1 := etree.NewElement("href")
				href1.Space = "d"
				href1.SetText("/principals/users/assistant/")
				elem.AddChild(href1)

				return elem
			}(),
			property: &CalendarProxyWriteFor{},
			expected: []string{"/principals/users/assistant/"},
		},
		{
			name:     "CalendarColor",
			element:  createTestElement("cs", "calendar-color", "#FF5733", nil),
			property: &CalendarColor{},
			expected: "#FF5733",
		},

		// Google CalDAV Extensions
		{
			name:     "Color",
			element:  createTestElement("g", "color", "#33FF57", nil),
			property: &Color{},
			expected: "#33FF57",
		},
		{
			name:     "Timezone",
			element:  createTestElement("g", "timezone", "Europe/London", nil),
			property: &Timezone{},
			expected: "Europe/London",
		},
		{
			name:     "Hidden_True",
			element:  createTestElement("g", "hidden", "true", nil),
			property: &Hidden{},
			expected: true,
		},
		{
			name:     "Hidden_False",
			element:  createTestElement("g", "hidden", "false", nil),
			property: &Hidden{},
			expected: false,
		},
		{
			name:     "Selected_True",
			element:  createTestElement("g", "selected", "true", nil),
			property: &Selected{},
			expected: true,
		},
		{
			name:     "Selected_False",
			element:  createTestElement("g", "selected", "false", nil),
			property: &Selected{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode the element into the property
			err := tt.property.Decode(tt.element)
			assert.NoError(t, err)

			// Check the decoded value matches what we expect
			switch prop := tt.property.(type) {
			case *GetCTag:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *CalendarChanges:
				assert.Equal(t, tt.expected.(string), prop.Href)
			case *SharedURL:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *Invite:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *NotificationURL:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *AutoSchedule:
				assert.Equal(t, tt.expected.(bool), prop.Value)
			case *CalendarProxyReadFor:
				assert.ElementsMatch(t, tt.expected.([]string), prop.Hrefs)
			case *CalendarProxyWriteFor:
				assert.ElementsMatch(t, tt.expected.([]string), prop.Hrefs)
			case *CalendarColor:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *Color:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *Timezone:
				assert.Equal(t, tt.expected.(string), prop.Value)
			case *Hidden:
				assert.Equal(t, tt.expected.(bool), prop.Value)
			case *Selected:
				assert.Equal(t, tt.expected.(bool), prop.Value)
			default:
				t.Fatalf("Unexpected property type: %T", prop)
			}
		})
	}
}

// Test decode-encode round trip for extension properties
func TestExtensionDecodeEncodeCycle(t *testing.T) {
	// Initialize with test data
	originalProperties := []Property{
		&GetCTag{Value: "abcdef123456"},
		&CalendarChanges{Href: "/calendars/users/alice/changes/"},
		&SharedURL{Value: "https://example.com/shared/calendar/"},
		&Invite{Value: "invitetoken123"},
		&NotificationURL{Value: "/calendars/users/alice/notifications/"},
		&AutoSchedule{Value: true},
		&CalendarProxyReadFor{Hrefs: []string{"/principals/users/manager/", "/principals/users/admin/"}},
		&CalendarProxyWriteFor{Hrefs: []string{"/principals/users/assistant/"}},
		&CalendarColor{Value: "#FF5733"},
		&Color{Value: "#33FF57"},
		&Timezone{Value: "Europe/London"},
		&Hidden{Value: true},
		&Selected{Value: false},
	}

	for _, original := range originalProperties {
		t.Run(fmt.Sprintf("%T", original), func(t *testing.T) {
			// Encode the property
			encoded := original.Encode()

			// Create a new instance of the same property type
			var decoded Property
			switch original.(type) {
			case *GetCTag:
				decoded = &GetCTag{}
			case *CalendarChanges:
				decoded = &CalendarChanges{}
			case *SharedURL:
				decoded = &SharedURL{}
			case *Invite:
				decoded = &Invite{}
			case *NotificationURL:
				decoded = &NotificationURL{}
			case *AutoSchedule:
				decoded = &AutoSchedule{}
			case *CalendarProxyReadFor:
				decoded = &CalendarProxyReadFor{}
			case *CalendarProxyWriteFor:
				decoded = &CalendarProxyWriteFor{}
			case *CalendarColor:
				decoded = &CalendarColor{}
			case *Color:
				decoded = &Color{}
			case *Timezone:
				decoded = &Timezone{}
			case *Hidden:
				decoded = &Hidden{}
			case *Selected:
				decoded = &Selected{}
			default:
				t.Fatalf("Unexpected property type: %T", original)
				return
			}

			// Decode the encoded element
			err := decoded.Decode(encoded)
			assert.NoError(t, err)

			// Re-encode the decoded property
			reEncoded := decoded.Encode()

			// Convert both to strings for comparison
			originalXml := elementToString(encoded)
			reEncodedXml := elementToString(reEncoded)

			// Clean strings for comparison
			originalClean := cleanXMLString(originalXml)
			reEncodedClean := cleanXMLString(reEncodedXml)

			// Should match
			assert.Equal(t, originalClean, reEncodedClean, "Round trip encode-decode-encode should produce the same XML")
		})
	}
}
