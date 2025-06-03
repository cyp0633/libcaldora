package server

import (
	"io"
	"net/http"
	"time"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
	"github.com/samber/mo"
)

// Update the handlePropfind function to use MergeResponses
func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	// fetch all requested resources as Depth header
	initialResource := ctx.Resource
	children, err := h.fetchChildren(ctx.Depth, initialResource)
	if err != nil {
		h.Logger.Error("failed to fetch children for resource",
			"resource", initialResource,
			"error", err)
		http.Error(w, "Failed to fetch children", http.StatusInternalServerError)
		return
	}
	resources := append([]Resource{initialResource}, children...)

	// parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read request body",
			"error", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	req, _ := propfind.ParseRequest(string(bodyBytes))
	// TODO: PropName handling

	var docs []*etree.Document
	for _, resource := range resources {
		ctx1 := *ctx             // Create a copy of the context
		ctx1.Resource = resource // Update the context for the individual resource

		var doc *etree.Document
		var err error

		switch resource.ResourceType {
		case storage.ResourcePrincipal:
			doc, err = h.handlePropfindPrincipal(req, ctx1.Resource)
		case storage.ResourceHomeSet:
			doc, err = h.handlePropfindHomeSet(req, ctx1.Resource)
		case storage.ResourceCollection:
			doc, err = h.handlePropfindCollection(req, ctx1.Resource)
		case storage.ResourceObject:
			doc, err = h.handlePropfindObject(req, ctx1.Resource)
		case storage.ResourceServiceRoot:
			ctx1.Resource.UserID = ctx.AuthUser // Just a workaround
			doc, err = h.handlePropfindServiceRoot(req, ctx1.Resource)
		default:
			h.Logger.Error("unknown resource type",
				"type", resource.ResourceType,
				"resource", resource)
			http.Error(w, "Unknown resource type", http.StatusNotFound)
			return
		}

		if err != nil {
			h.Logger.Error("error handling PROPFIND",
				"resource_type", resource.ResourceType,
				"resource", resource,
				"error", err)
			continue // Skip this resource but continue with others
		}

		if doc != nil {
			docs = append(docs, doc)
		}
	}

	// Merge all responses
	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		h.Logger.Error("failed to merge PROPFIND responses",
			"error", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status

	// Serialize and write the XML document
	xmlOutput, err := mergedDoc.WriteToString()
	if err != nil {
		h.Logger.Error("failed to serialize XML response",
			"error", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xmlOutput))
}

// handles individual home set request
func (h *CaldavHandler) handlePropfindHomeSet(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	var user *storage.User
	getUser := func() (*storage.User, error) {
		if user != nil {
			return user, nil
		}
		u, err := h.Storage.GetUser(res.UserID)
		if err != nil {
			h.Logger.Error("failed to get user for resource",
				"resource", res,
				"error", err)
			return nil, err
		}
		user = u
		if user == nil {
			h.Logger.Error("no user found for resource",
				"resource", res)
			return nil, propfind.ErrNotFound // Return not found if no user is associated with the resource
		}
		return user, nil
	}

	for key := range req {
		switch key {
		case "displayname":
			req[key] = mo.Ok[props.Property](&props.DisplayName{Value: "Calendar Home"})
		case "resourcetype":
			req[key] = mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceHomeSet})
		case "owner":
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				resource := Resource{
					UserID:       res.UserID,
					ResourceType: storage.ResourcePrincipal,
				}
				encodedPath, err := h.URLConverter.EncodePath(resource)
				if err != nil {
					h.Logger.Error("failed to encode owner URL for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				req[key] = mo.Ok[props.Property](&props.Owner{Value: encodedPath})
			}
		case "current-user-principal":
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				resource := Resource{
					UserID:       res.UserID,
					ResourceType: storage.ResourcePrincipal,
				}
				encodedPath, err := h.URLConverter.EncodePath(resource)
				if err != nil {
					h.Logger.Error("failed to encode principal URL for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				req[key] = mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: encodedPath})
			}
		case "principal-url":
			res := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(res)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.PrincipalURL{Value: encodedPath})
		case "supported-report-set":
			req[key] = mo.Ok[props.Property](&props.SupportedReportSet{Reports: []props.ReportType{}})
		case "acl":
			res := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			principalPath, err := h.URLConverter.EncodePath(res)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			ace := props.ACE{
				Principal: principalPath,
				Grant:     []string{"read", "write"}, // TODO: complete ACL
				Deny:      []string{},
			}
			acl := props.ACL{Aces: []props.ACE{ace}}
			req[key] = mo.Ok[props.Property](&acl)
		case "current-user-privilege-set":
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}})
		case "supported-calendar-data":
			req[key] = mo.Ok[props.Property](&props.SupportedCalendarData{
				ContentType: "icalendar",
				Version:     "2.0",
			})
		case "max-resource-size":
			req[key] = mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760})
		case "min-date-time":
			req[key] = mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()})
		case "max-date-time":
			req[key] = mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
		case "max-instances":
			req[key] = mo.Ok[props.Property](&props.MaxInstances{Value: 100000})
		case "max-attendees-per-instance":
			req[key] = mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100})
		case "calendar-home-set":
			req[key] = mo.Ok[props.Property](&props.CalendarHomeSet{Href: path})
		case "calendar-user-address-set":
			user, err = getUser()
			if err == nil && user != nil && user.UserAddress != "" {
				req[key] = mo.Ok[props.Property](&props.CalendarUserAddressSet{Addresses: []string{user.UserAddress}})
			} else {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound) // fallback to default if no user found
			}
		case "calendar-user-type":
			req[key] = mo.Ok[props.Property](&props.CalendarUserType{Value: "individual"})
		default:
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		}
	}
	return propfind.EncodeResponse(req, path), nil
}

// handles user principal request
func (h *CaldavHandler) handlePropfindPrincipal(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}
	user, err := h.Storage.GetUser(res.UserID)
	if err != nil {
		h.Logger.Error("failed to get user for resource",
			"resource", res,
			"error", err)
		// Return an internal error if we cannot get the user
		return nil, err
	}
	if user == nil {
		h.Logger.Error("no user found for resource",
			"resource", res)
		// Return a not found error if no user is associated with the resource
		return nil, propfind.ErrNotFound
	}

	for key := range req {
		switch key {
		case "displayname":
			if user.DisplayName != "" {
				req[key] = mo.Ok[props.Property](&props.DisplayName{Value: user.DisplayName})
			} else {
				req[key] = mo.Ok[props.Property](&props.DisplayName{Value: res.UserID}) // fallback to UserID if DisplayName is empty
			}
		case "resourcetype":
			req[key] = mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourcePrincipal})
		case "getcontenttype":
			// No file, on purpose
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "owner":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode owner URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.Owner{Value: encodedPath})
		case "current-user-principal":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: encodedPath})
		case "principal-url":
			req[key] = mo.Ok[props.Property](&props.PrincipalURL{Value: path})
		case "supported-report-set":
			req[key] = mo.Ok[props.Property](&props.SupportedReportSet{Reports: []props.ReportType{}})
		case "acl":
			// For now, return a simple ACL with read/write for the principal itself
			ace := props.ACE{
				Principal: path,
				Grant:     []string{"read", "write"}, // TODO: complete ACL
				Deny:      []string{},
			}
			acl := props.ACL{Aces: []props.ACE{ace}}
			req[key] = mo.Ok[props.Property](&acl)
		case "current-user-privilege-set":
			// Return a simple privilege set for the principal
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}})
		case "calendar-home-set":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourceHomeSet,
			}
			homeSetPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode calendar home set for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrInternal)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CalendarHomeSet{Href: homeSetPath})
		case "schedule-inbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-outbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-default-calendar-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "calendar-user-address-set":
			if user.UserAddress == "" {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarUserAddressSet{Addresses: []string{user.UserAddress}})
			}
		case "calendar-user-type":
			req[key] = mo.Ok[props.Property](&props.CalendarUserType{Value: "individual"})
		case "calendar-color", "color":
			if user.PreferredColor == "" {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarColor{Value: user.PreferredColor})
			}
		case "timezone":
			if user.PreferredTimezone == "" {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.Timezone{Value: user.PreferredTimezone})
			}
		case "hidden":
			// default to false
			req[key] = mo.Ok[props.Property](&props.Hidden{Value: false})
		case "selected":
			// default to true
			req[key] = mo.Ok[props.Property](&props.Selected{Value: true})
		default:
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		}
	}
	return propfind.EncodeResponse(req, path), nil
}

// handlePropfindObject is a wrapper that first fetches the object, then calls the inner function
func (h *CaldavHandler) handlePropfindObject(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	if res.URI == "" {
		path, err := h.URLConverter.EncodePath(res)
		if err != nil {
			h.Logger.Error("failed to encode path for resource",
				"resource", res,
				"error", err)
			return nil, err
		}
		res.URI = path
	}

	object, err := h.Storage.GetObject(res.UserID, res.CalendarID, res.ObjectID)
	if err != nil {
		h.Logger.Error("failed to get object for resource",
			"resource", res,
			"error", err)
		return nil, err
	}
	if object == nil || len(object.Component) == 0 {
		h.Logger.Error("no object found for resource",
			"resource", res)
		return nil, propfind.ErrNotFound
	}

	return h.handlePropfindObjectWithObject(req, res, *object)
}

// handlePropfindObjectWithObject processes a PROPFIND request for a calendar object
// when the calendar object has already been fetched
func (h *CaldavHandler) handlePropfindObjectWithObject(req propfind.ResponseMap, res Resource, object storage.CalendarObject) (*etree.Document, error) {
	var calendar *storage.Calendar
	var user *storage.User
	var err error

	for key := range req {
		switch key {
		case "displayname":
			name, err := object.Component[0].Props.Text(ical.PropName)
			if err != nil {
				h.Logger.Debug("failed to get display name for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.DisplayName{Value: name})
			}
		case "resourcetype":
			req[key] = mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceObject, ObjectType: object.Component[0].Name})
		case "getetag":
			if object.ETag != "" {
				req[key] = mo.Ok[props.Property](&props.GetEtag{Value: object.ETag})
			} else {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			}
		case "getlastmodified":
			lastModified, err := object.Component[0].Props.DateTime(ical.PropLastModified, nil)
			if err != nil {
				h.Logger.Debug("failed to get last modified date for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.GetLastModified{Value: lastModified})
			}
		case "getcontenttype":
			req[key] = mo.Ok[props.Property](&props.GetContentType{Value: "text/calendar"})
		case "owner":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode owner URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.Owner{Value: encodedPath})
		case "current-user-principal":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: encodedPath})
		case "principal-url":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrInternal)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.PrincipalURL{Value: encodedPath})
		case "supported-report-set":
			req[key] = mo.Ok[props.Property](&props.SupportedReportSet{Reports: []props.ReportType{}})
		case "acl":
			// For now, return a simple ACL with read/write for the principal itself
			ace := props.ACE{
				Principal: res.URI,
				Grant:     []string{"read", "write"}, // TODO: complete ACL
				Deny:      []string{},
			}
			acl := props.ACL{Aces: []props.ACE{ace}}
			req[key] = mo.Ok[props.Property](&acl)
		case "current-user-privilege-set":
			// Return a simple privilege set for the principal
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}})
		case "calendar-description":
			if calendar == nil {
				calendar, err = h.Storage.GetCalendar(res.UserID, res.CalendarID)
				if err != nil {
					h.Logger.Error("failed to get calendar for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrInternal)
					continue
				}
				if calendar == nil || calendar.CalendarData == nil {
					h.Logger.Error("no calendar found for resource",
						"resource", res)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				description, err := calendar.CalendarData.Props.Text(ical.PropDescription)
				if err != nil {
					h.Logger.Debug("failed to get calendar description for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				} else {
					req[key] = mo.Ok[props.Property](&props.CalendarDescription{Value: description})
				}
			}
		case "calendar-timezone", "timezone":
			if calendar == nil {
				calendar, err = h.Storage.GetCalendar(res.UserID, res.CalendarID)
				if err != nil {
					h.Logger.Error("failed to get calendar for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrInternal)
					continue
				}
				if calendar == nil || calendar.CalendarData == nil {
					h.Logger.Error("no calendar found for resource",
						"resource", res)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				timezone, err := calendar.CalendarData.Component.Props.Text(ical.PropTimezoneID)
				if err != nil {
					h.Logger.Debug("failed to get timezone from calendar data for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				} else {
					req[key] = mo.Ok[props.Property](&props.CalendarTimezone{Value: timezone})
				}
			}
		case "calendar-data":
			ics, err := storage.ICalCompToICS(object.Component, false)
			if err != nil {
				h.Logger.Error("failed to convert calendar component to ICS for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarData{ICal: ics})
			}
		case "supported-calendar-data":
			req[key] = mo.Ok[props.Property](&props.SupportedCalendarData{
				ContentType: "text/calendar",
				Version:     "2.0",
			})
		case "max-resource-size":
			req[key] = mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760})
		case "min-date-time":
			req[key] = mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()})
		case "max-date-time":
			req[key] = mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
		case "max-instances":
			req[key] = mo.Ok[props.Property](&props.MaxInstances{Value: 100000})
		case "max-attendees-per-instance":
			req[key] = mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100})
		case "calendar-home-set":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourceHomeSet,
			}
			homeSetPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode calendar home set for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrInternal)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CalendarHomeSet{Href: homeSetPath})
		case "schedule-inbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-outbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-default-calendar-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "calendar-user-address-set":
			if user == nil {
				user, err = h.Storage.GetUser(res.UserID)
				if err != nil {
					h.Logger.Error("failed to get user for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrInternal)
					continue
				}
				if user == nil || user.UserAddress == "" {
					h.Logger.Error("no user found for resource",
						"resource", res)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				if user.UserAddress == "" {
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				} else {
					req[key] = mo.Ok[props.Property](&props.CalendarUserAddressSet{Addresses: []string{user.UserAddress}})
				}
			}
		case "calendar-user-type":
			req[key] = mo.Ok[props.Property](&props.CalendarUserType{Value: "individual"})
		case "calendar-color", "color":
			if user == nil {
				user, err = h.Storage.GetUser(res.UserID)
				if err != nil {
					h.Logger.Error("failed to get user for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrInternal)
					continue
				}
				if user == nil || user.PreferredColor == "" {
					h.Logger.Error("no user found for resource",
						"resource", res)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				if user.PreferredColor == "" {
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				} else {
					req[key] = mo.Ok[props.Property](&props.CalendarColor{Value: user.PreferredColor})
				}
			}
		case "hidden":
			// default to false
			req[key] = mo.Ok[props.Property](&props.Hidden{Value: false})
		case "selected":
			// default to true
			req[key] = mo.Ok[props.Property](&props.Selected{Value: true})
		default:
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		}
	}
	return propfind.EncodeResponse(req, res.URI), nil
}

func (h *CaldavHandler) handlePropfindCollection(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	h.Logger.Debug("handling PROPFIND for collection",
		"path", path,
		"user_id", res.UserID,
		"calendar_id", res.CalendarID,
		"resource_type", res.ResourceType)

	calendar, err := h.Storage.GetCalendar(res.UserID, res.CalendarID)
	if err != nil {
		h.Logger.Error("failed to get calendar for resource",
			"resource", res,
			"error", err)
		return nil, err
	}

	if calendar == nil {
		h.Logger.Error("calendar not found",
			"user_id", res.UserID,
			"calendar_id", res.CalendarID)
		return nil, propfind.ErrNotFound
	}

	if calendar.CalendarData == nil {
		h.Logger.Error("calendar data is nil",
			"user_id", res.UserID,
			"calendar_id", res.CalendarID)
		return nil, propfind.ErrNotFound
	}

	h.Logger.Debug("found calendar",
		"user_id", res.UserID,
		"calendar_id", res.CalendarID,
		"has_calendar_data", calendar.CalendarData != nil)

	var user *storage.User

	for key := range req {
		switch key {
		case "displayname":
			name, err := calendar.CalendarData.Props.Text(ical.PropName)
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.DisplayName{Value: name})
			}
		case "resourcetype":
			req[key] = mo.Ok[props.Property](&props.Resourcetype{Type: props.ResourceCollection})
		case "getetag":
			if calendar.ETag != "" {
				req[key] = mo.Ok[props.Property](&props.GetEtag{Value: calendar.ETag})
			} else {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			}
		case "getlastmodified":
			lastModified, err := calendar.CalendarData.Props.DateTime(ical.PropLastModified, nil)
			if err != nil {
				h.Logger.Debug("failed to get last modified date for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.GetLastModified{Value: lastModified})
			}
		case "getcontenttype":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "owner":
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				resource := Resource{
					UserID:       res.UserID,
					ResourceType: storage.ResourcePrincipal,
				}
				encodedPath, err := h.URLConverter.EncodePath(resource)
				if err != nil {
					h.Logger.Error("failed to encode owner URL for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				req[key] = mo.Ok[props.Property](&props.Owner{Value: encodedPath})
			}
		case "current-user-principal":
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				resource := Resource{
					UserID:       res.UserID,
					ResourceType: storage.ResourcePrincipal,
				}
				encodedPath, err := h.URLConverter.EncodePath(resource)
				if err != nil {
					h.Logger.Error("failed to encode principal URL for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrNotFound)
					continue
				}
				req[key] = mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: encodedPath})
			}
		case "principal-url":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrInternal)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.PrincipalURL{Value: encodedPath})
		case "supported-report-set":
			req[key] = mo.Ok[props.Property](&props.SupportedReportSet{Reports: []props.ReportType{}})
		case "acl":
			// For now, return a simple ACL with read/write for the principal itself
			ace := props.ACE{
				Principal: path,
				Grant:     []string{"read", "write"}, // TODO: complete ACL
				Deny:      []string{},
			}
			acl := props.ACL{Aces: []props.ACE{ace}}
			req[key] = mo.Ok[props.Property](&acl)
		case "current-user-privilege-set":
			// Return a simple privilege set for the principal
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}})
		case "calendar-description":
			description, err := calendar.CalendarData.Props.Text(ical.PropDescription)
			if err != nil {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarDescription{Value: description})
			}
		case "calendar-timezone", "timezone":
			timezone, err := calendar.CalendarData.Component.Props.Text(ical.PropTimezoneID)
			if err != nil {
				h.Logger.Debug("failed to get timezone from calendar data for resource",
					"path", path,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarTimezone{Value: timezone})
			}
		case "supported-calendar-component-set":
			if len(calendar.SupportedComponents) == 0 {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.SupportedCalendarComponentSet{Components: calendar.SupportedComponents})
			}
		case "supported-calendar-data":
			req[key] = mo.Ok[props.Property](&props.SupportedCalendarData{
				ContentType: "icalendar",
				Version:     "2.0",
			})
		case "max-resource-size":
			req[key] = mo.Ok[props.Property](&props.MaxResourceSize{Value: 10485760}) // 10MB limit for calendar objects
		case "min-date-time":
			req[key] = mo.Ok[props.Property](&props.MinDateTime{Value: time.Unix(0, 0).UTC()}) // Minimum date-time for calendar objects
		case "max-date-time":
			req[key] = mo.Ok[props.Property](&props.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)}) // Maximum date-time for calendar objects
		case "max-instances":
			req[key] = mo.Ok[props.Property](&props.MaxInstances{Value: 100000}) // Allow up to 100000 instances in recurrence
		case "max-attendees-per-instance":
			req[key] = mo.Ok[props.Property](&props.MaxAttendeesPerInstance{Value: 100}) // Allow up to 100 attendees per instance
		case "calendar-home-set":
			resource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourceHomeSet,
			}
			homeSetPath, err := h.URLConverter.EncodePath(resource)
			if err != nil {
				h.Logger.Error("failed to encode calendar home set for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrInternal)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CalendarHomeSet{Href: homeSetPath})
		case "schedule-inbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-outbox-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "schedule-default-calendar-url":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "calendar-user-address-set":
			if user == nil {
				user, err = h.Storage.GetUser(res.UserID)
				if err != nil {
					h.Logger.Error("failed to get user for resource",
						"resource", res,
						"error", err)
					req[key] = mo.Err[props.Property](propfind.ErrInternal)
					continue
				}
			}
			if user.UserAddress == "" {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarUserAddressSet{Addresses: []string{user.UserAddress}})
			}
		case "calendar-user-type":
			req[key] = mo.Ok[props.Property](&props.CalendarUserType{Value: "individual"}) // Default to individual for calendar-user-type
		case "calendar-color", "color":
			color, err := calendar.CalendarData.Props.Text(ical.PropColor)
			if err != nil || color == "" {
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
			} else {
				req[key] = mo.Ok[props.Property](&props.CalendarColor{Value: color})
			}
		case "calendar-proxy-read-for", "calendar-proxy-write-for":
			// TODO
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		case "hidden":
			req[key] = mo.Ok[props.Property](&props.Hidden{Value: false}) // Default to false for hidden property
		case "selected":
			req[key] = mo.Ok[props.Property](&props.Selected{Value: true}) // Default to true for selected property
		default:
			req[key] = mo.Err[props.Property](propfind.ErrNotFound) // Default case for unsupported properties
		}
	}

	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) handlePropfindServiceRoot(req propfind.ResponseMap, res Resource) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(res)
	if err != nil {
		h.Logger.Error("failed to encode path for resource",
			"resource", res,
			"error", err)
		return nil, err
	}
	for key := range req {
		switch key {
		case "displayname":
			req[key] = mo.Ok[props.Property](&props.DisplayName{Value: "CalDAV Service Root"})
		case "current-user-principal":
			principalResource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(principalResource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrincipal{Value: encodedPath})
		case "current-user-privilege-set":
			req[key] = mo.Ok[props.Property](&props.CurrentUserPrivilegeSet{Privileges: []string{"read", "read-acl", "read-current-user-privilege-set"}})
		case "principal-url":
			principalResource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(principalResource)
			if err != nil {
				h.Logger.Error("failed to encode principal URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.PrincipalURL{Value: encodedPath})
		case "calendar-home-set":
			homeSetResource := Resource{
				UserID:       res.UserID,
				ResourceType: storage.ResourceHomeSet,
			}
			encodedPath, err := h.URLConverter.EncodePath(homeSetResource)
			if err != nil {
				h.Logger.Error("failed to encode calendar home set URL for resource",
					"resource", res,
					"error", err)
				req[key] = mo.Err[props.Property](propfind.ErrNotFound)
				continue
			}
			req[key] = mo.Ok[props.Property](&props.CalendarHomeSet{Href: encodedPath})
		default:
			req[key] = mo.Err[props.Property](propfind.ErrNotFound)
		}
	}
	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) fetchChildren(depth int, parent Resource) (resources []Resource, err error) {
	if depth <= 0 {
		return
	}

	h.Logger.Debug("fetching children",
		"depth", depth,
		"parent_type", parent.ResourceType,
		"user_id", parent.UserID,
		"calendar_id", parent.CalendarID)

	switch parent.ResourceType {
	case storage.ResourceObject, storage.ResourcePrincipal:
		// These types don't have children, return empty slice
		h.Logger.Debug("resource type has no children",
			"resource_type", parent.ResourceType)
		return []Resource{}, nil

	case storage.ResourceCollection:
		// find object (event) paths in the collection
		paths, err := h.Storage.GetObjectPathsInCollection(parent.CalendarID)
		if err != nil {
			h.Logger.Error("failed to fetch event paths in collection",
				"calendar_id", parent.CalendarID,
				"error", err)
			return nil, err
		}

		h.Logger.Debug("found event paths in collection",
			"calendar_id", parent.CalendarID,
			"path_count", len(paths))

		for _, path := range paths {
			h.Logger.Debug("parsing event path",
				"path", path,
				"calendar_id", parent.CalendarID)

			resource, err := h.URLConverter.ParsePath(path)
			if err != nil {
				h.Logger.Error("failed to parse path",
					"path", path,
					"error", err)
				return nil, err
			}

			h.Logger.Debug("parsed event resource",
				"path", path,
				"resource_type", resource.ResourceType,
				"object_id", resource.ObjectID)

			resources = append(resources, resource)
			children, err := h.fetchChildren(depth-1, resource) // Recursively fetch children for the object
			if err != nil {
				h.Logger.Error("failed to fetch children for resource",
					"resource", resource,
					"error", err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	case storage.ResourceHomeSet:
		// find collections in the home set
		calendars, err := h.Storage.GetUserCalendars(parent.UserID)
		if err != nil {
			h.Logger.Error("failed to fetch calendars for user",
				"user_id", parent.UserID,
				"error", err)
			return nil, err
		}
		for _, cal := range calendars {
			resource, err := h.URLConverter.ParsePath(cal.Path)
			if err != nil {
				h.Logger.Error("failed to parse calendar path",
					"path", cal.Path,
					"error", err)
				return nil, err
			}
			resources = append(resources, resource)
			// Recursively fetch children for the collection
			children, err := h.fetchChildren(depth-1, resource)
			if err != nil {
				h.Logger.Error("failed to fetch children for resource",
					"resource", resource,
					"error", err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	}
	return
}
