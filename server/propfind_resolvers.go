package server

import (
	"time"

	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/samber/mo"
)

// Resolver resolves a single property for the given environment.
type Resolver func(env *propEnv) mo.Result[props.Property]

// propEnv provides lazy accessors for frequently used resources.
type propEnv struct {
	h       *CaldavHandler
	res     Resource
	preload *storage.CalendarObject

	user     *storage.User
	calendar *storage.Calendar
	object   *storage.CalendarObject
}

func newPropEnv(h *CaldavHandler, res Resource, preload *storage.CalendarObject) *propEnv {
	return &propEnv{h: h, res: res, preload: preload}
}

func (e *propEnv) ResourceHref() (string, error) {
	if e.res.URI != "" {
		return e.res.URI, nil
	}
	return e.h.URLConverter.EncodePath(e.res)
}

func (e *propEnv) PrincipalHref() (string, error) {
	r := Resource{UserID: e.res.UserID, ResourceType: storage.ResourcePrincipal}
	return e.h.URLConverter.EncodePath(r)
}

func (e *propEnv) HomeSetHref() (string, error) {
	r := Resource{UserID: e.res.UserID, ResourceType: storage.ResourceHomeSet}
	return e.h.URLConverter.EncodePath(r)
}

func (e *propEnv) GetUser() (*storage.User, error) {
	if e.user != nil {
		return e.user, nil
	}
	u, err := e.h.Storage.GetUser(e.res.UserID)
	if err != nil {
		return nil, err
	}
	e.user = u
	return e.user, nil
}

func (e *propEnv) GetCalendar() (*storage.Calendar, error) {
	if e.calendar != nil {
		return e.calendar, nil
	}
	c, err := e.h.Storage.GetCalendar(e.res.UserID, e.res.CalendarID)
	if err != nil {
		return nil, err
	}
	e.calendar = c
	return e.calendar, nil
}

func (e *propEnv) GetObject() (*storage.CalendarObject, error) {
	if e.object != nil {
		return e.object, nil
	}
	if e.preload != nil {
		e.object = e.preload
		return e.object, nil
	}
	o, err := e.h.Storage.GetObject(e.res.UserID, e.res.CalendarID, e.res.ObjectID)
	if err != nil {
		return nil, err
	}
	e.object = o
	return e.object, nil
}

func (e *propEnv) privilegeSet() ([]string, error) {
	switch e.res.ResourceType {
	case storage.ResourceCollection, storage.ResourceObject:
		cal, err := e.GetCalendar()
		if err != nil {
			return nil, err
		}
		if cal != nil && cal.ReadOnly {
			return []string{"read"}, nil
		}
		if cal == nil {
			return []string{"read"}, nil
		}
		return []string{"read", "write"}, nil
	default:
		return []string{"read", "write"}, nil
	}
}

func buildACLProperty(env *propEnv, principal string) mo.Result[props.Property] {
	privs, err := env.privilegeSet()
	if err != nil {
		env.h.Logger.Error("failed to determine privileges for acl",
			"resource", env.res,
			"error", err,
		)
		return mo.Err[props.Property](propfind.ErrInternal)
	}
	ace := props.ACE{Principal: principal, Grant: privs, Deny: []string{}}
	return mo.Ok[props.Property](&props.ACL{Aces: []props.ACE{ace}})
}

// resolveWith dispatches properties using the provided resolver table.
func resolveWith(env *propEnv, resolvers map[string]Resolver, req propfind.ResponseMap) propfind.ResponseMap {
	for key := range req {
		if r, ok := resolvers[key]; ok {
			req[key] = r(env)
		} else {
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		}
	}
	return req
}

// Common resolvers shared across resource types.
var commonResolvers = map[string]Resolver{
	"owner": func(env *propEnv) mo.Result[props.Property] {
		href, err := env.PrincipalHref()
		if err != nil {
			env.h.Logger.Error("failed to encode owner URL", "resource", env.res, "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.Owner{Value: href})
	},
	"current-user-principal": func(env *propEnv) mo.Result[props.Property] {
		href, err := env.PrincipalHref()
		if err != nil {
			env.h.Logger.Error("failed to encode principal URL", "resource", env.res, "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: href})
	},
	"principal-url": func(env *propEnv) mo.Result[props.Property] {
		href, err := env.PrincipalHref()
		if err != nil {
			env.h.Logger.Error("failed to encode principal URL", "resource", env.res, "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.PrincipalURL{Value: href})
	},
	"supported-report-set": func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.SupportedReportSet{Reports: []props.ReportType{}})
	},
	"current-user-privilege-set": func(env *propEnv) mo.Result[props.Property] {
		privs, err := env.privilegeSet()
		if err != nil {
			env.h.Logger.Error("failed to determine privilege set",
				"resource", env.res,
				"error", err,
			)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		return mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: privs})
	},
	"calendar-home-set": func(env *propEnv) mo.Result[props.Property] {
		href, err := env.HomeSetHref()
		if err != nil {
			env.h.Logger.Error("failed to encode calendar home set", "resource", env.res, "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		return mo.Ok[props.Property](&props.CalendarHomeSet{Href: href})
	},
	"calendar-user-address-set": func(env *propEnv) mo.Result[props.Property] {
		user, err := env.GetUser()
		if err != nil {
			env.h.Logger.Error("failed to get user", "resource", env.res, "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if user == nil || user.UserAddress == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarUserAddressSet{Addresses: []string{user.UserAddress}})
	},
	"calendar-user-type": func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.CalendarUserType{Value: "individual"})
	},
	"hidden": func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.Hidden{Value: false})
	},
	"selected": func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.Selected{Value: true})
	},
}

// Principal specific resolvers.
var principalResolvers = func() map[string]Resolver {
	m := map[string]Resolver{}
	// inherit common
	for k, v := range commonResolvers {
		m[k] = v
	}
	m["displayname"] = func(env *propEnv) mo.Result[props.Property] {
		user, err := env.GetUser()
		if err != nil {
			env.h.Logger.Error("failed to get user for displayname", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		name := env.res.UserID
		if user != nil && user.DisplayName != "" {
			name = user.DisplayName
		}
		return mo.Ok[props.Property](&props.DisplayName{Value: name})
	}
	m["resourcetype"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourcePrincipal})
	}
	m["getcontenttype"] = func(_ *propEnv) mo.Result[props.Property] { return mo.Err[props.Property](propfind.ErrNotFound) }
	m["calendar-user-address-set"] = commonResolvers["calendar-user-address-set"]
	m["calendar-color"] = func(env *propEnv) mo.Result[props.Property] {
		user, err := env.GetUser()
		if err != nil {
			env.h.Logger.Error("failed to get user for color", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if user == nil || user.PreferredColor == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarColor{Value: user.PreferredColor})
	}
	m["color"] = m["calendar-color"]
	m["timezone"] = func(env *propEnv) mo.Result[props.Property] {
		user, err := env.GetUser()
		if err != nil {
			env.h.Logger.Error("failed to get user for timezone", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if user == nil || user.PreferredTimezone == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.Timezone{Value: user.PreferredTimezone})
	}
	// ACL principal uses its own href as principal
	m["acl"] = func(env *propEnv) mo.Result[props.Property] {
		href, err := env.ResourceHref()
		if err != nil {
			env.h.Logger.Error("failed to encode resource href for acl", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return buildACLProperty(env, href)
	}
	return m
}()

// HomeSet specific resolvers.
var homeSetResolvers = func() map[string]Resolver {
	m := map[string]Resolver{}
	for k, v := range commonResolvers {
		m[k] = v
	}
	m["displayname"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.DisplayName{Value: "Calendar Home"})
	}
	m["resourcetype"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceHomeSet})
	}
	// ACL for homeset uses principal as ACE principal
	m["acl"] = func(env *propEnv) mo.Result[props.Property] {
		principal, err := env.PrincipalHref()
		if err != nil {
			env.h.Logger.Error("failed to encode principal href for acl", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return buildACLProperty(env, principal)
	}
	// supported-calendar-data on homeset used "icalendar" in existing handlers
	m["supported-calendar-data"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.SupportedCalendarData{ContentType: "icalendar", Version: "2.0"})
	}
	// size/limits
	m["max-resource-size"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760})
	}
	m["min-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()})
	}
	m["max-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
	}
	m["max-instances"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxInstances{Value: 100000})
	}
	m["max-attendees-per-instance"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100})
	}
	// scheduling URLs not implemented
	m["schedule-inbox-url"] = func(_ *propEnv) mo.Result[props.Property] { return mo.Err[props.Property](propfind.ErrNotFound) }
	m["schedule-outbox-url"] = m["schedule-inbox-url"]
	m["schedule-default-calendar-url"] = m["schedule-inbox-url"]
	return m
}()

// Collection specific resolvers.
var collectionResolvers = func() map[string]Resolver {
	m := map[string]Resolver{}
	for k, v := range commonResolvers {
		m[k] = v
	}
	m["displayname"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for displayname", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		name, err := cal.CalendarData.Props.Text(ical.PropName)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.DisplayName{Value: name})
	}
	m["resourcetype"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceCollection})
	}
	m["getetag"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for etag", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		if cal == nil || cal.ETag == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.GetEtag{Value: cal.ETag})
	}
	m["getlastmodified"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Debug("failed to get calendar for lastmodified", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		t, err := cal.CalendarData.Props.DateTime(ical.PropLastModified, nil)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.GetLastModified{Value: t})
	}
	m["getcontenttype"] = func(_ *propEnv) mo.Result[props.Property] { return mo.Err[props.Property](propfind.ErrNotFound) }
	m["calendar-description"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for description", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		desc, err := cal.CalendarData.Props.Text(ical.PropDescription)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarDescription{Value: desc})
	}
	m["calendar-timezone"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for timezone", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		tz, err := cal.CalendarData.Component.Props.Text(ical.PropTimezoneID)
		if err != nil {
			env.h.Logger.Debug("failed to get timezone from calendar data", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarTimezone{Value: tz})
	}
	m["timezone"] = m["calendar-timezone"]
	m["supported-calendar-component-set"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || len(cal.SupportedComponents) == 0 {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.SupportedCalendarComponentSet{Components: cal.SupportedComponents})
	}
	m["supported-calendar-data"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.SupportedCalendarData{ContentType: "icalendar", Version: "2.0"})
	}
	m["max-resource-size"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760})
	}
	m["min-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()})
	}
	m["max-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
	}
	m["max-instances"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxInstances{Value: 100000})
	}
	m["max-attendees-per-instance"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100})
	}
	m["calendar-color"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		color, err := cal.CalendarData.Props.Text(ical.PropColor)
		if err != nil || color == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarColor{Value: color})
	}
	m["color"] = m["calendar-color"]
	// ACL for collection uses its own href as principal
	m["acl"] = func(env *propEnv) mo.Result[props.Property] {
		href, err := env.ResourceHref()
		if err != nil {
			env.h.Logger.Error("failed to encode resource href for acl", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return buildACLProperty(env, href)
	}
	// schedule props not implemented
	m["schedule-inbox-url"] = func(_ *propEnv) mo.Result[props.Property] { return mo.Err[props.Property](propfind.ErrNotFound) }
	m["schedule-outbox-url"] = m["schedule-inbox-url"]
	m["schedule-default-calendar-url"] = m["schedule-inbox-url"]
	return m
}()

// Object specific resolvers.
var objectResolvers = func() map[string]Resolver {
	m := map[string]Resolver{}
	for k, v := range commonResolvers {
		m[k] = v
	}
	m["displayname"] = func(env *propEnv) mo.Result[props.Property] {
		obj, err := env.GetObject()
		if err != nil || obj == nil || len(obj.Component) == 0 {
			env.h.Logger.Debug("failed to get object for displayname", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		name, err := obj.Component[0].Props.Text(ical.PropName)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.DisplayName{Value: name})
	}
	m["resourcetype"] = func(env *propEnv) mo.Result[props.Property] {
		obj, err := env.GetObject()
		if err != nil || obj == nil || len(obj.Component) == 0 {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceObject, ObjectType: obj.Component[0].Name})
	}
	m["getetag"] = func(env *propEnv) mo.Result[props.Property] {
		obj, err := env.GetObject()
		if err != nil || obj == nil || obj.ETag == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.GetEtag{Value: obj.ETag})
	}
	m["getlastmodified"] = func(env *propEnv) mo.Result[props.Property] {
		obj, err := env.GetObject()
		if err != nil || obj == nil || len(obj.Component) == 0 {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		t, err := obj.Component[0].Props.DateTime(ical.PropLastModified, nil)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.GetLastModified{Value: t})
	}
	m["getcontenttype"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.GetContentType{Value: "text/calendar"})
	}
	m["calendar-description"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for description", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		desc, err := cal.CalendarData.Props.Text(ical.PropDescription)
		if err != nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarDescription{Value: desc})
	}
	m["calendar-timezone"] = func(env *propEnv) mo.Result[props.Property] {
		cal, err := env.GetCalendar()
		if err != nil {
			env.h.Logger.Error("failed to get calendar for timezone", "error", err)
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if cal == nil || cal.CalendarData == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		tz, err := cal.CalendarData.Component.Props.Text(ical.PropTimezoneID)
		if err != nil {
			env.h.Logger.Debug("failed to get timezone from calendar data", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarTimezone{Value: tz})
	}
	m["timezone"] = m["calendar-timezone"]
	m["calendar-data"] = func(env *propEnv) mo.Result[props.Property] {
		obj, err := env.GetObject()
		if err != nil || obj == nil {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		ics, err := storage.ICalCompToICS(obj.Component, false)
		if err != nil {
			env.h.Logger.Error("failed to convert component to ics", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarData{ICal: ics})
	}
	m["supported-calendar-data"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.SupportedCalendarData{ContentType: "text/calendar", Version: "2.0"})
	}
	m["max-resource-size"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760})
	}
	m["min-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()})
	}
	m["max-date-time"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
	}
	m["max-instances"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxInstances{Value: 100000})
	}
	m["max-attendees-per-instance"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100})
	}
	// For object ACL, existing code used the resource's own URI as principal
	m["acl"] = func(env *propEnv) mo.Result[props.Property] {
		href, err := env.ResourceHref()
		if err != nil {
			env.h.Logger.Error("failed to encode resource href for acl", "error", err)
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return buildACLProperty(env, href)
	}
	// Color on object uses user's preferred color, per existing behavior
	m["calendar-color"] = func(env *propEnv) mo.Result[props.Property] {
		user, err := env.GetUser()
		if err != nil {
			return mo.Err[props.Property](propfind.ErrInternal)
		}
		if user == nil || user.PreferredColor == "" {
			return mo.Err[props.Property](propfind.ErrNotFound)
		}
		return mo.Ok[props.Property](&props.CalendarColor{Value: user.PreferredColor})
	}
	m["color"] = m["calendar-color"]
	// scheduling props not implemented
	m["schedule-inbox-url"] = func(_ *propEnv) mo.Result[props.Property] { return mo.Err[props.Property](propfind.ErrNotFound) }
	m["schedule-outbox-url"] = m["schedule-inbox-url"]
	m["schedule-default-calendar-url"] = m["schedule-inbox-url"]
	return m
}()

// Service root specific resolvers.
var serviceRootResolvers = func() map[string]Resolver {
	m := map[string]Resolver{}
	// Display name
	m["displayname"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.DisplayName{Value: "CalDAV Service Root"})
	}
	// current-user-principal, principal-url
	m["current-user-principal"] = commonResolvers["current-user-principal"]
	m["principal-url"] = commonResolvers["principal-url"]
	m["calendar-home-set"] = commonResolvers["calendar-home-set"]
	// privileges different on service root
	m["current-user-privilege-set"] = func(_ *propEnv) mo.Result[props.Property] {
		return mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "read-acl", "read-current-user-privilege-set"}})
	}
	return m
}()

// resolvePropfind fills the ResponseMap for the given resource type.
func (h *CaldavHandler) resolvePropfind(req propfind.ResponseMap, res Resource, preload *storage.CalendarObject) propfind.ResponseMap {
	env := newPropEnv(h, res, preload)
	var table map[string]Resolver
	switch res.ResourceType {
	case storage.ResourcePrincipal:
		table = principalResolvers
	case storage.ResourceHomeSet:
		table = homeSetResolvers
	case storage.ResourceCollection:
		table = collectionResolvers
	case storage.ResourceObject:
		table = objectResolvers
	case storage.ResourceServiceRoot:
		table = serviceRootResolvers
	default:
		table = map[string]Resolver{}
	}
	return resolveWith(env, table, req)
}
