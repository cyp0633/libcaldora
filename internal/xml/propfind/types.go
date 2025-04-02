package propfind

// Mapping of WebDAV/CalDAV property names to their struct types
var propNameToStruct = map[string]any{
	// WebDAV properties
	"displayname":                new(displayName),
	"resourcetype":               new(resourcetype),
	"getetag":                    new(getEtag),
	"getlastmodified":            new(getLastModified),
	"getcontenttype":             new(getContentType),
	"owner":                      new(owner),
	"current-user-principal":     new(currentUserPrincipal),
	"principal-url":              new(principalURL),
	"supported-report-set":       new(supportedReportSet),
	"acl":                        new(acl),
	"current-user-privilege-set": new(currentUserPrivilegeSet),
	"quota-available-bytes":      new(quotaAvailableBytes),
	"quota-used-bytes":           new(quotaUsedBytes),

	// CalDAV properties
	"calendar-description":             new(calendarDescription),
	"calendar-timezone":                new(calendarTimezone),
	"supported-calendar-component-set": new(supportedCalendarComponentSet),
	"supported-calendar-data":          new(supportedCalendarData),
	"max-resource-size":                new(maxResourceSize),
	"min-date-time":                    new(minDateTime),
	"max-date-time":                    new(maxDateTime),
	"max-instances":                    new(maxInstances),
	"max-attendees-per-instance":       new(maxAttendeesPerInstance),
	"calendar-home-set":                new(calendarHomeSet),
	"schedule-inbox-url":               new(scheduleInboxURL),
	"schedule-outbox-url":              new(scheduleOutboxURL),
	"schedule-default-calendar-url":    new(scheduleDefaultCalendarURL),
	"calendar-user-address-set":        new(calendarUserAddressSet),
	"calendar-user-type":               new(calendarUserType),

	// Apple CalendarServer Extensions
	"getctag":                  new(getCTag),
	"calendar-changes":         new(calendarChanges),
	"shared-url":               new(sharedURL),
	"invite":                   new(invite),
	"notification-url":         new(notificationURL),
	"auto-schedule":            new(autoSchedule),
	"calendar-proxy-read-for":  new(calendarProxyReadFor),
	"calendar-proxy-write-for": new(calendarProxyWriteFor),

	// Google CalDAV Extensions
	"color":    new(color),
	"timezone": new(timezone),
	"hidden":   new(hidden),
	"selected": new(selected),
}

var namespaceMap = map[string]string{
	"D":  "DAV:",
	"C":  "urn:ietf:params:xml:ns:caldav",
	"CS": "http://calendarserver.org/ns/",
	"g":  "http://schemas.google.com/gCal/2005",
}

// WebDAV properties

type displayName struct{}

type resourcetype struct{}

type getEtag struct{}

type getLastModified struct{}

type getContentType struct{}

type owner struct{}

type currentUserPrincipal struct{}

type principalURL struct{}

type supportedReportSet struct{}

type acl struct{}

type currentUserPrivilegeSet struct{}

type quotaAvailableBytes struct{}

type quotaUsedBytes struct{}

// CalDAV properties

type calendarDescription struct{}

type calendarTimezone struct{}

type supportedCalendarComponentSet struct{}

type supportedCalendarData struct{}

type maxResourceSize struct{}

type minDateTime struct{}

type maxDateTime struct{}

type maxInstances struct{}

type maxAttendeesPerInstance struct{}

type calendarHomeSet struct{}

type scheduleInboxURL struct{}

type scheduleOutboxURL struct{}

type scheduleDefaultCalendarURL struct{}

type calendarUserAddressSet struct{}

type calendarUserType struct{}

// Apple CalendarServer Extensions

type getCTag struct{}

type calendarChanges struct{}

type sharedURL struct{}

type invite struct{}

type notificationURL struct{}

type autoSchedule struct{}

type calendarProxyReadFor struct{}

type calendarProxyWriteFor struct{}

// Google CalDAV Extensions

type color struct{}

type timezone struct{}

type hidden struct{}

type selected struct{}
