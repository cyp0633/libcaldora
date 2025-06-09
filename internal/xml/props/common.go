package props

import "github.com/beevik/etree"

// Property interface for all property types (use pointer!)
type Property interface {
	Encode() *etree.Element
	Decode(element *etree.Element) error
}

// Namespace map for declaration (if needed by etree)
var NamespaceMap = map[string]string{
	"d":    "DAV:",
	"cal":  "urn:ietf:params:xml:ns:caldav",
	"cs":   "http://calendarserver.org/ns/",
	"g":    "http://schemas.google.com/gCal/2005",
	"ical": "http://apple.com/ns/ical/",
}

// Prefix map for each property and child element
var PropPrefixMap = map[string]string{
	// WebDAV properties (d: prefix)
	"displayname":                "d",
	"resourcetype":               "d",
	"getetag":                    "d",
	"getlastmodified":            "d",
	"getcontenttype":             "d",
	"owner":                      "d",
	"current-user-principal":     "d",
	"principal-url":              "d",
	"supported-report-set":       "d",
	"acl":                        "d",
	"current-user-privilege-set": "d",
	"quota-available-bytes":      "d",
	"quota-used-bytes":           "d",
	// Additional child elements for WebDAV
	"collection":       "d",
	"principal":        "d",
	"href":             "d",
	"grant":            "d",
	"privilege":        "d",
	"supported-report": "d",
	"search":           "d",
	"report":           "d",
	"ace":              "d",

	// CalDAV properties (cal: prefix)
	"calendar-description":             "cal",
	"calendar-timezone":                "cal",
	"calendar-data":                    "cal",
	"supported-calendar-component-set": "cal",
	"supported-calendar-data":          "cal",
	"max-resource-size":                "cal",
	"min-date-time":                    "cal",
	"max-date-time":                    "cal",
	"max-instances":                    "cal",
	"max-attendees-per-instance":       "cal",
	"calendar-home-set":                "cal",
	"schedule-inbox-url":               "cal",
	"schedule-outbox-url":              "cal",
	"schedule-default-calendar-url":    "cal",
	"calendar-user-address-set":        "cal",
	"calendar-user-type":               "cal",
	"calendar":                         "cal",
	"comp":                             "cal",
	"calendar-query":                   "cal",
	"calendar-multiget":                "cal",
	"free-busy-query":                  "cal",
	"schedule-query":                   "cal",
	"schedule-multiget":                "cal",

	// Apple CalendarServer Extensions (cs: prefix)
	"getctag":                  "cs",
	"calendar-changes":         "cs",
	"shared-url":               "cs",
	"invite":                   "cs",
	"notification-url":         "cs",
	"auto-schedule":            "cs",
	"calendar-proxy-read-for":  "cs",
	"calendar-proxy-write-for": "cs",
	"calendar-color":           "ical",

	// Google CalDAV Extensions (g: prefix)
	"color":    "g",
	"timezone": "g",
	"hidden":   "g",
	"selected": "g",
}

// Reuse the property mapping from propfind
var PropNameToStruct = map[string]Property{
	// WebDAV properties
	"displayname":                new(DisplayName),
	"resourcetype":               new(Resourcetype),
	"getetag":                    new(GetEtag),
	"getlastmodified":            new(GetLastModified),
	"getcontenttype":             new(GetContentType),
	"owner":                      new(Owner),
	"current-user-principal":     new(CurrentUserPrincipal),
	"principal-url":              new(PrincipalURL),
	"supported-report-set":       new(SupportedReportSet),
	"acl":                        new(ACL),
	"current-user-privilege-set": new(CurrentUserPrivilegeSet),
	"quota-available-bytes":      new(QuotaAvailableBytes),
	"quota-used-bytes":           new(QuotaUsedBytes),

	// CalDAV properties
	"calendar-description":             new(CalendarDescription),
	"calendar-timezone":                new(CalendarTimezone),
	"calendar-data":                    new(CalendarData),
	"supported-calendar-component-set": new(SupportedCalendarComponentSet),
	"supported-calendar-data":          new(SupportedCalendarData),
	"max-resource-size":                new(MaxResourceSize),
	"min-date-time":                    new(MinDateTime),
	"max-date-time":                    new(MaxDateTime),
	"max-instances":                    new(MaxInstances),
	"max-attendees-per-instance":       new(MaxAttendeesPerInstance),
	"calendar-home-set":                new(CalendarHomeSet),
	"schedule-inbox-url":               new(ScheduleInboxURL),
	"schedule-outbox-url":              new(ScheduleOutboxURL),
	"schedule-default-calendar-url":    new(ScheduleDefaultCalendarURL),
	"calendar-user-address-set":        new(CalendarUserAddressSet),
	"calendar-user-type":               new(CalendarUserType),

	// Apple CalendarServer Extensions
	"getctag":                  new(GetCTag),
	"calendar-changes":         new(CalendarChanges),
	"shared-url":               new(SharedURL),
	"invite":                   new(Invite),
	"notification-url":         new(NotificationURL),
	"auto-schedule":            new(AutoSchedule),
	"calendar-proxy-read-for":  new(CalendarProxyReadFor),
	"calendar-proxy-write-for": new(CalendarProxyWriteFor),
	"calendar-color":           new(CalendarColor),

	// Google CalDAV Extensions
	"color":    new(Color),
	"timezone": new(Timezone),
	"hidden":   new(Hidden),
	"selected": new(Selected),
}

// createElement creates an element with the namespace prefix taken from the propPrefixMap.
// If the name is not found in the map, it defaults to "d".
func createElement(name string) *etree.Element {
	prefix, exists := PropPrefixMap[name]
	if !exists {
		prefix = "d" // Default to DAV namespace
	}
	elem := etree.NewElement(name)
	elem.Space = prefix
	return elem
}

// createElementWithPrefix creates an element with the provided name and explicitly sets the given prefix.
func createElementWithPrefix(name, prefix string) *etree.Element {
	elem := etree.NewElement(name)
	elem.Space = prefix
	return elem
}
