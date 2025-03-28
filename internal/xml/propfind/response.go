package propfind

import "github.com/samber/mo"

type PropfindResponse struct {
	// WebDAV properties
	DisplayName             mo.Option[string]
	Resourcetype            mo.Option[string]
	GetEtag                 mo.Option[string]
	GetLastModified         mo.Option[string]
	GetContentType          mo.Option[string]
	Owner                   mo.Option[string]
	CurrentUserPrincipal    mo.Option[string]
	PrincipalURL            mo.Option[string]
	SupportedReportSet      mo.Option[[]string] // List of supported reports
	ACL                     mo.Option[[]ACLEntry]
	CurrentUserPrivilegeSet mo.Option[[]string]
	QuotaAvailableBytes     mo.Option[int64]
	QuotaUsedBytes          mo.Option[int64]

	// CalDAV properties
	CalendarDescription           mo.Option[string]
	CalendarTimezone              mo.Option[string]
	SupportedCalendarComponentSet mo.Option[[]string]
	SupportedCalendarData         mo.Option[[]string]
	MaxResourceSize               mo.Option[int64]
	MinDateTime                   mo.Option[string]
	MaxDateTime                   mo.Option[string]
	MaxInstances                  mo.Option[int]
	MaxAttendeesPerInstance       mo.Option[int]
	CalendarHomeSet               mo.Option[string]
	ScheduleInboxURL              mo.Option[string]
	ScheduleOutboxURL             mo.Option[string]
	ScheduleDefaultCalendarURL    mo.Option[string]
	CalendarUserAddressSet        mo.Option[[]string]
	CalendarUserType              mo.Option[string]

	// Apple CalendarServer Extensions
	GetCTag               mo.Option[string]
	CalendarChanges       mo.Option[string]
	SharedURL             mo.Option[string]
	Invite                mo.Option[string] // Might need a more complex type
	NotificationURL       mo.Option[string]
	AutoSchedule          mo.Option[bool]
	CalendarProxyReadFor  mo.Option[[]string]
	CalendarProxyWriteFor mo.Option[[]string]

	// Google CalDAV Extensions
	Color    mo.Option[string]
	Timezone mo.Option[string]
	Hidden   mo.Option[bool]
	Selected mo.Option[bool]
}

// ACLEntry represents a single entry in the ACL property
type ACLEntry struct {
	Principal string
	Grant     []string
	Deny      []string
}

var namespaceMap = map[string]string{
	"D":  "DAV:",
	"C":  "urn:ietf:params:xml:ns:caldav",
	"CS": "http://calendarserver.org/ns/",
	"g":  "http://schemas.google.com/gCal/2005",
}

func (r *PropfindResponse) Parse() {
	// Implementation will parse XML response into this structure
}

func (r *PropfindResponse) Encode() {
	// Implementation will encode this structure into XML
}
