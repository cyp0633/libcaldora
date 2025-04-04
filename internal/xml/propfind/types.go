package propfind

import (
	"errors"
	"strconv"
	"strings"

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
	"calendar-color":           new(calendarColor),

	// Google CalDAV Extensions
	"color":    new(color),
	"timezone": new(timezone),
	"hidden":   new(hidden),
	"selected": new(selected),
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

type displayName struct {
	Value string
}

func (p *displayName) Encode() *etree.Element {
	elem := createElement("displayname")
	elem.SetText(p.Value)
	return elem
}

type resourcetype struct {
	Types []string
}

func (p *resourcetype) Encode() *etree.Element {
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

type getEtag struct {
	Value string
}

func (p *getEtag) Encode() *etree.Element {
	elem := createElement("getetag")
	elem.SetText(p.Value)
	return elem
}

type getLastModified struct {
	Value string
}

func (p *getLastModified) Encode() *etree.Element {
	elem := createElement("getlastmodified")
	elem.SetText(p.Value)
	return elem
}

type getContentType struct {
	Value string
}

func (p *getContentType) Encode() *etree.Element {
	elem := createElement("getcontenttype")
	elem.SetText(p.Value)
	return elem
}

type owner struct {
	Value string
}

func (p *owner) Encode() *etree.Element {
	elem := createElement("owner")
	elem.SetText(p.Value)
	return elem
}

type currentUserPrincipal struct {
	Value string
}

func (p *currentUserPrincipal) Encode() *etree.Element {
	elem := createElement("current-user-principal")
	elem.SetText(p.Value)
	return elem
}

type principalURL struct {
	Value string
}

func (p *principalURL) Encode() *etree.Element {
	elem := createElement("principal-url")
	elem.SetText(p.Value)
	return elem
}

type supportedReportSet struct {
	Reports []string
}

func (p *supportedReportSet) Encode() *etree.Element {
	elem := createElement("supported-report-set")
	for _, report := range p.Reports {
		reportElem := createElement("report")
		reportElem.SetText(report)
		elem.AddChild(reportElem)
	}
	return elem
}

type acl struct {
	Aces []ace
}

func (p *acl) Encode() *etree.Element {
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

type ace struct {
	Principal string
	Grant     []string
	Deny      []string
}

type currentUserPrivilegeSet struct {
	Privileges []string
}

func (p *currentUserPrivilegeSet) Encode() *etree.Element {
	elem := createElement("current-user-privilege-set")

	for _, privilege := range p.Privileges {
		privElem := createElement("privilege")
		elem.AddChild(privElem)

		privTypeElem := createElement(privilege)
		privElem.AddChild(privTypeElem)
	}

	return elem
}

type quotaAvailableBytes struct {
	Value int64
}

func (p *quotaAvailableBytes) Encode() *etree.Element {
	elem := createElement("quota-available-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

type quotaUsedBytes struct {
	Value int64
}

func (p *quotaUsedBytes) Encode() *etree.Element {
	elem := createElement("quota-used-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

// CalDAV properties

type calendarDescription struct {
	Value string
}

func (p *calendarDescription) Encode() *etree.Element {
	elem := createElement("calendar-description")
	elem.SetText(p.Value)
	return elem
}

type calendarTimezone struct {
	Value string
}

func (p *calendarTimezone) Encode() *etree.Element {
	elem := createElement("calendar-timezone")
	elem.SetText(p.Value)
	return elem
}

type supportedCalendarComponentSet struct {
	Components []string
}

func (p *supportedCalendarComponentSet) Encode() *etree.Element {
	elem := createElement("supported-calendar-component-set")

	for _, component := range p.Components {
		compElem := createElement("comp")
		compElem.CreateAttr("name", component)
		elem.AddChild(compElem)
	}

	return elem
}

type supportedCalendarData struct {
	ContentType string
	Version     string
}

func (p *supportedCalendarData) Encode() *etree.Element {
	elem := createElement("supported-calendar-data")
	elem.SetText(p.ContentType)
	if p.Version != "" {
		elem.CreateAttr("version", p.Version)
	}
	return elem
}

type maxResourceSize struct {
	Value int64
}

func (p *maxResourceSize) Encode() *etree.Element {
	elem := createElement("max-resource-size")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

type minDateTime struct {
	Value string
}

func (p *minDateTime) Encode() *etree.Element {
	elem := createElement("min-date-time")
	elem.SetText(p.Value)
	return elem
}

type maxDateTime struct {
	Value string
}

func (p *maxDateTime) Encode() *etree.Element {
	elem := createElement("max-date-time")
	elem.SetText(p.Value)
	return elem
}

type maxInstances struct {
	Value int
}

func (p *maxInstances) Encode() *etree.Element {
	elem := createElement("max-instances")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type maxAttendeesPerInstance struct {
	Value int
}

func (p *maxAttendeesPerInstance) Encode() *etree.Element {
	elem := createElement("max-attendees-per-instance")
	elem.SetText(strconv.Itoa(p.Value))
	return elem
}

type calendarHomeSet struct {
	Href string
}

func (p *calendarHomeSet) Encode() *etree.Element {
	elem := createElement("calendar-home-set")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type scheduleInboxURL struct {
	Href string
}

func (p *scheduleInboxURL) Encode() *etree.Element {
	elem := createElement("schedule-inbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type scheduleOutboxURL struct {
	Href string
}

func (p *scheduleOutboxURL) Encode() *etree.Element {
	elem := createElement("schedule-outbox-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type scheduleDefaultCalendarURL struct {
	Href string
}

func (p *scheduleDefaultCalendarURL) Encode() *etree.Element {
	elem := createElement("schedule-default-calendar-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type calendarUserAddressSet struct {
	Addresses []string
}

func (p *calendarUserAddressSet) Encode() *etree.Element {
	elem := createElement("calendar-user-address-set")

	for _, address := range p.Addresses {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(address)
	}

	return elem
}

type calendarUserType struct {
	Value string
}

func (p *calendarUserType) Encode() *etree.Element {
	elem := createElement("calendar-user-type")
	elem.SetText(p.Value)
	return elem
}

// Apple CalendarServer Extensions

type getCTag struct {
	Value string
}

func (p *getCTag) Encode() *etree.Element {
	elem := createElement("getctag")
	elem.SetText(p.Value)
	return elem
}

type calendarChanges struct {
	Href string
}

func (p *calendarChanges) Encode() *etree.Element {
	elem := createElement("calendar-changes")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Href)
	return elem
}

type sharedURL struct {
	Value string
}

func (p *sharedURL) Encode() *etree.Element {
	elem := createElement("shared-url")
	elem.SetText(p.Value)
	return elem
}

type invite struct {
	Value string
}

func (p *invite) Encode() *etree.Element {
	elem := createElement("invite")
	elem.SetText(p.Value)
	return elem
}

type notificationURL struct {
	Value string
}

func (p *notificationURL) Encode() *etree.Element {
	elem := createElement("notification-url")
	elem.SetText(p.Value)
	return elem
}

type autoSchedule struct {
	Value bool
}

func (p *autoSchedule) Encode() *etree.Element {
	elem := createElement("auto-schedule")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

type calendarProxyReadFor struct {
	Hrefs []string
}

func (p *calendarProxyReadFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-read-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

type calendarProxyWriteFor struct {
	Hrefs []string
}

func (p *calendarProxyWriteFor) Encode() *etree.Element {
	elem := createElement("calendar-proxy-write-for")

	for _, href := range p.Hrefs {
		hrefElem := createElement("href")
		elem.AddChild(hrefElem)
		hrefElem.SetText(href)
	}

	return elem
}

type calendarColor struct {
	Value string
}

func (p *calendarColor) Encode() *etree.Element {
	elem := createElement("calendar-color")
	elem.SetText(p.Value)
	return elem
}

// Google CalDAV Extensions

type color struct {
	Value string
}

func (p *color) Encode() *etree.Element {
	elem := createElement("color")
	elem.SetText(p.Value)
	return elem
}

type timezone struct {
	Value string
}

func (p *timezone) Encode() *etree.Element {
	elem := createElement("timezone")
	elem.SetText(p.Value)
	return elem
}

type hidden struct {
	Value bool
}

func (p *hidden) Encode() *etree.Element {
	elem := createElement("hidden")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}

type selected struct {
	Value bool
}

func (p *selected) Encode() *etree.Element {
	elem := createElement("selected")
	if p.Value {
		elem.SetText("true")
	} else {
		elem.SetText("false")
	}
	return elem
}
