package propfind

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/samber/mo"
)

// PropertyEncoder interface for all property types
type PropertyEncoder interface {
	Encode() *etree.Element
}

// Mapping of WebDAV/CalDAV property names to their struct types
var propNameToStruct = map[string]PropertyEncoder{
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

// Prefix map for each property and child element
var propPrefixMap = map[string]string{
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
	"collection": "d",
	"report":     "d",
	"ace":        "d",
	"principal":  "d",
	"href":       "d",
	"grant":      "d",
	"privilege":  "d",

	// CalDAV properties (cal: prefix)
	"calendar-description":             "cal",
	"calendar-timezone":                "cal",
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

	// Apple CalendarServer Extensions (cs: prefix)
	"getctag":                  "cs",
	"calendar-changes":         "cs",
	"shared-url":               "cs",
	"invite":                   "cs",
	"notification-url":         "cs",
	"auto-schedule":            "cs",
	"calendar-proxy-read-for":  "cs",
	"calendar-proxy-write-for": "cs",
	"calendar-color":           "cs",

	// Google CalDAV Extensions (g: prefix)
	"color":    "g",
	"timezone": "g",
	"hidden":   "g",
	"selected": "g",
}

type ResponseMap map[string]mo.Result[PropertyEncoder]

type RequestType int

const (
	RequestTypeProp     RequestType = iota // Propfind request
	RequestTypePropName                    // Propname request (only return property names)
	RequestTypeAllProp                     // Allprop request (return all properties)
)

var (
	ErrNotFound   = errors.New("HTTP 404: Property not found")
	ErrForbidden  = errors.New("HTTP 403: Forbidden access to the resource")
	ErrInternal   = errors.New("HTTP 500: Internal server error")
	ErrBadRequest = errors.New("HTTP 400: Bad request")
)

// Namespace map for declaration (if needed by etree)
var namespaceMap = map[string]string{
	"d":   "DAV:",
	"cal": "urn:ietf:params:xml:ns:caldav",
	"cs":  "http://calendarserver.org/ns/",
	"g":   "http://schemas.google.com/gCal/2005",
}

// createElement creates an element with the namespace prefix taken from the propPrefixMap.
// If the name is not found in the map, it defaults to "d".
func createElement(name string) *etree.Element {
	prefix, exists := propPrefixMap[name]
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

// WebDAV properties

type DisplayName struct {
	Value string
}

func (p DisplayName) Encode() *etree.Element {
	elem := createElement("displayname")
	elem.SetText(p.Value)
	return elem
}

type Resourcetype struct {
	Types []string
}

func (p Resourcetype) Encode() *etree.Element {
	elem := createElement("resourcetype")

	for _, typeName := range p.Types {
		if typeName == "collection" {
			collElem := createElement("collection")
			elem.AddChild(collElem)
		} else if typeName == "calendar" {
			calElem := createElement("calendar")
			elem.AddChild(calElem)
		} else {
			parts := strings.Split(typeName, ":")
			if len(parts) > 1 {
				// Use the provided prefix and element name
				child := createElementWithPrefix(parts[1], parts[0])
				elem.AddChild(child)
			} else {
				child := createElement(typeName)
				elem.AddChild(child)
			}
		}
	}
	return elem
}

type GetEtag struct {
	Value string
}

func (p GetEtag) Encode() *etree.Element {
	elem := createElement("getetag")
	elem.SetText(p.Value)
	return elem
}

type GetLastModified struct {
	Value string
}

func (p GetLastModified) Encode() *etree.Element {
	elem := createElement("getlastmodified")
	elem.SetText(p.Value)
	return elem
}

type GetContentType struct {
	Value string
}

func (p GetContentType) Encode() *etree.Element {
	elem := createElement("getcontenttype")
	elem.SetText(p.Value)
	return elem
}

type Owner struct {
	Value string
}

func (p Owner) Encode() *etree.Element {
	elem := createElement("owner")
	elem.SetText(p.Value)
	return elem
}

type CurrentUserPrincipal struct {
	Value string
}

func (p CurrentUserPrincipal) Encode() *etree.Element {
	elem := createElement("current-user-principal")
	elem.SetText(p.Value)
	return elem
}

type PrincipalURL struct {
	Value string
}

func (p PrincipalURL) Encode() *etree.Element {
	elem := createElement("principal-url")
	elem.SetText(p.Value)
	return elem
}

type SupportedReportSet struct {
	Reports []string
}

func (p SupportedReportSet) Encode() *etree.Element {
	elem := createElement("supported-report-set")
	for _, report := range p.Reports {
		reportElem := createElement("report")
		reportElem.SetText(report)
		elem.AddChild(reportElem)
	}
	return elem
}

type ACL struct {
	Aces []ACE
}

func (p ACL) Encode() *etree.Element {
	elem := createElement("acl")

	for _, aceEntry := range p.Aces {
		aceElem := createElement("ace")
		elem.AddChild(aceElem)

		// Principal
		principalElem := createElement("principal")
		aceElem.AddChild(principalElem)

		hrefElem := createElement("href")
		principalElem.AddChild(hrefElem)
		hrefElem.SetText(aceEntry.Principal)

		// Grant privileges
		if len(aceEntry.Grant) > 0 {
			grantElem := createElement("grant")
			aceElem.AddChild(grantElem)

			for _, privilege := range aceEntry.Grant {
				privElem := createElement("privilege")
				grantElem.AddChild(privElem)

				privTypeElem := createElement(privilege)
				privElem.AddChild(privTypeElem)
			}
		}

		// Deny privileges
		if len(aceEntry.Deny) > 0 {
			denyElem := createElement("deny")
			aceElem.AddChild(denyElem)

			for _, privilege := range aceEntry.Deny {
				privElem := createElement("privilege")
				denyElem.AddChild(privElem)

				privTypeElem := createElement(privilege)
				privElem.AddChild(privTypeElem)
			}
		}
	}

	return elem
}

type ACE struct {
	Principal string
	Grant     []string
	Deny      []string
}

type CurrentUserPrivilegeSet struct {
	Privileges []string
}

func (p CurrentUserPrivilegeSet) Encode() *etree.Element {
	elem := createElement("current-user-privilege-set")

	for _, privilege := range p.Privileges {
		privElem := createElement("privilege")
		elem.AddChild(privElem)

		privTypeElem := createElement(privilege)
		privElem.AddChild(privTypeElem)
	}

	return elem
}

type QuotaAvailableBytes struct {
	Value int64
}

func (p QuotaAvailableBytes) Encode() *etree.Element {
	elem := createElement("quota-available-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

type QuotaUsedBytes struct {
	Value int64
}

func (p QuotaUsedBytes) Encode() *etree.Element {
	elem := createElement("quota-used-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

// CalDAV properties

type CalendarDescription struct {
	Value string
}

func (p CalendarDescription) Encode() *etree.Element {
	elem := createElement("calendar-description")
	elem.SetText(p.Value)
	return elem
}

type CalendarTimezone struct {
	Value string
}

func (p CalendarTimezone) Encode() *etree.Element {
	elem := createElement("calendar-timezone")
	elem.SetText(p.Value)
	return elem
}

type SupportedCalendarComponentSet struct {
	Components []string
}

func (p SupportedCalendarComponentSet) Encode() *etree.Element {
	elem := createElement("supported-calendar-component-set")

	for _, component := range p.Components {
		compElem := createElement("comp")
		compElem.CreateAttr("name", component)
		elem.AddChild(compElem)
	}

	return elem
}

type SupportedCalendarData struct {
	ContentType string
	Version     string
}

func (p SupportedCalendarData) Encode() *etree.Element {
	elem := createElement("supported-calendar-data")
	elem.SetText(p.ContentType)
	if p.Version != "" {
		elem.CreateAttr("version", p.Version)
	}
	return elem
}

type MaxResourceSize struct {
	Value int64
}

func (p MaxResourceSize) Encode() *etree.Element {
	elem := createElement("max-resource-size")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

type MinDateTime struct {
	Value time.Time
}

func (p MinDateTime) Encode() *etree.Element {
	elem := createElement("min-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

type MaxDateTime struct {
	Value time.Time
}

func (p MaxDateTime) Encode() *etree.Element {
	elem := createElement("max-date-time")
	elem.SetText(p.Value.Format(time.RFC3339))
	return elem
}

type MaxInstances struct {
	Value int
}

func (p MaxInstances) Encode() *etree.Element {
	elem := createElement("max-instances")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type MaxAttendeesPerInstance struct {
	Value int
}

func (p MaxAttendeesPerInstance) Encode() *etree.Element {
	elem := createElement("max-attendees-per-instance")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type CalendarHomeSet struct {
	Href string
}

func (p CalendarHomeSet) Encode() *etree.Element {
	elem := createElement("calendar-home-set")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleInboxURL struct {
	Href string
}

func (p ScheduleInboxURL) Encode() *etree.Element {
	elem := createElement("schedule-inbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleOutboxURL struct {
	Href string
}

func (p ScheduleOutboxURL) Encode() *etree.Element {
	elem := createElement("schedule-outbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type ScheduleDefaultCalendarURL struct {
	Href string
}

func (p ScheduleDefaultCalendarURL) Encode() *etree.Element {
	elem := createElement("schedule-default-calendar-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type CalendarUserAddressSet struct {
	Addresses []string
}

func (p CalendarUserAddressSet) Encode() *etree.Element {
	elem := createElement("calendar-user-address-set")

	for _, address := range p.Addresses {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(address)
	}

	return elem
}

type CalendarUserType struct {
	Value string
}

func (p CalendarUserType) Encode() *etree.Element {
	elem := createElement("calendar-user-type")
	elem.SetText(p.Value)
	return elem
}

// Apple CalendarServer Extensions

type GetCTag struct {
	Value string
}

func (p GetCTag) Encode() *etree.Element {
	elem := createElement("getctag")
	elem.SetText(p.Value)
	return elem
}

type CalendarChanges struct {
	Href string
}

func (p CalendarChanges) Encode() *etree.Element {
	elem := createElement("calendar-changes")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type SharedURL struct {
	Value string
}

func (p SharedURL) Encode() *etree.Element {
	elem := createElement("shared-url")
	elem.SetText(p.Value)
	return elem
}

type Invite struct {
	Value string
}

func (p Invite) Encode() *etree.Element {
	elem := createElement("invite")
	elem.SetText(p.Value)
	return elem
}

type NotificationURL struct {
	Value string
}

func (p NotificationURL) Encode() *etree.Element {
	elem := createElement("notification-url")
	elem.SetText(p.Value)
	return elem
}

type AutoSchedule struct {
	Value bool
}

func (p AutoSchedule) Encode() *etree.Element {
	elem := createElement("auto-schedule")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

type CalendarProxyReadFor struct {
	Hrefs []string
}

func (p CalendarProxyReadFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-read-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

type CalendarProxyWriteFor struct {
	Hrefs []string
}

func (p CalendarProxyWriteFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-write-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

type CalendarColor struct {
	Value string
}

func (p CalendarColor) Encode() *etree.Element {
	elem := createElement("calendar-color")
	elem.SetText(p.Value)
	return elem
}

// Google CalDAV Extensions

type Color struct {
	Value string
}

func (p Color) Encode() *etree.Element {
	elem := createElement("color")
	elem.SetText(p.Value)
	return elem
}

type Timezone struct {
	Value string
}

func (p Timezone) Encode() *etree.Element {
	elem := createElement("timezone")
	elem.SetText(p.Value)
	return elem
}

type Hidden struct {
	Value bool
}

func (p Hidden) Encode() *etree.Element {
	elem := createElement("hidden")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

type Selected struct {
	Value bool
}

func (p Selected) Encode() *etree.Element {
	elem := createElement("selected")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}
