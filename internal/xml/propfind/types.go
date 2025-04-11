package propfind

import (
	"errors"

	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/samber/mo"
)

// Mapping of WebDAV/CalDAV property names to their struct types
var propNameToStruct = map[string]props.PropertyEncoder{
	// WebDAV properties
	"displayname":                new(props.DisplayName),
	"resourcetype":               new(props.Resourcetype),
	"getetag":                    new(props.GetEtag),
	"getlastmodified":            new(props.GetLastModified),
	"getcontenttype":             new(props.GetContentType),
	"owner":                      new(props.Owner),
	"current-user-principal":     new(props.CurrentUserPrincipal),
	"principal-url":              new(props.PrincipalURL),
	"supported-report-set":       new(props.SupportedReportSet),
	"acl":                        new(props.ACL),
	"current-user-privilege-set": new(props.CurrentUserPrivilegeSet),
	"quota-available-bytes":      new(props.QuotaAvailableBytes),
	"quota-used-bytes":           new(props.QuotaUsedBytes),

	// CalDAV properties
	"calendar-description":             new(props.CalendarDescription),
	"calendar-timezone":                new(props.CalendarTimezone),
	"calendar-data":                    new(props.CalendarData),
	"supported-calendar-component-set": new(props.SupportedCalendarComponentSet),
	"supported-calendar-data":          new(props.SupportedCalendarData),
	"max-resource-size":                new(props.MaxResourceSize),
	"min-date-time":                    new(props.MinDateTime),
	"max-date-time":                    new(props.MaxDateTime),
	"max-instances":                    new(props.MaxInstances),
	"max-attendees-per-instance":       new(props.MaxAttendeesPerInstance),
	"calendar-home-set":                new(props.CalendarHomeSet),
	"schedule-inbox-url":               new(props.ScheduleInboxURL),
	"schedule-outbox-url":              new(props.ScheduleOutboxURL),
	"schedule-default-calendar-url":    new(props.ScheduleDefaultCalendarURL),
	"calendar-user-address-set":        new(props.CalendarUserAddressSet),
	"calendar-user-type":               new(props.CalendarUserType),

	// Apple CalendarServer Extensions
	"getctag":                  new(props.GetCTag),
	"calendar-changes":         new(props.CalendarChanges),
	"shared-url":               new(props.SharedURL),
	"invite":                   new(props.Invite),
	"notification-url":         new(props.NotificationURL),
	"auto-schedule":            new(props.AutoSchedule),
	"calendar-proxy-read-for":  new(props.CalendarProxyReadFor),
	"calendar-proxy-write-for": new(props.CalendarProxyWriteFor),
	"calendar-color":           new(props.CalendarColor),

	// Google CalDAV Extensions
	"color":    new(props.Color),
	"timezone": new(props.Timezone),
	"hidden":   new(props.Hidden),
	"selected": new(props.Selected),
}

type ResponseMap map[string]mo.Result[props.PropertyEncoder]

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
