package caldav

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/samber/mo"
)

// Update the handlePropfind function to use MergeResponses
func (h *CaldavHandler) handlePropfind(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	// fetch all requested resources as Depth header
	initialResource := ctx.Resource
	children, err := h.fetchChildren(ctx.Depth, initialResource)
	if err != nil {
		log.Printf("Failed to fetch children for resource %s: %v", initialResource, err)
		http.Error(w, "Failed to fetch children", http.StatusInternalServerError)
		return
	}
	resources := append([]Resource{initialResource}, children...)

	// parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
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
		case ResourcePrincipal:
			doc, err = h.handlePropfindPrincipal(req, &ctx1)
		case ResourceHomeSet:
			doc, err = h.handlePropfindHomeSet(req, &ctx1)
		case ResourceCollection:
			doc, err = h.handlePropfindCollection(req, &ctx1)
		case ResourceObject:
			doc, err = h.handlePropfindObject(req, &ctx1)
		}

		if err != nil {
			log.Printf("Error handling PROPFIND for resource %v: %v", resource, err)
			continue // Skip this resource but continue with others
		}

		if doc != nil {
			docs = append(docs, doc)
		}
	}

	// Merge all responses
	mergedDoc, err := propfind.MergeResponses(docs)
	if err != nil {
		log.Printf("Failed to merge PROPFIND responses: %v", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus) // 207 Multi-Status

	// Serialize and write the XML document
	xmlOutput, err := mergedDoc.WriteToString()
	if err != nil {
		log.Printf("Failed to serialize XML response: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xmlOutput))
}

// handles individual home set request
func (h *CaldavHandler) handlePropfindHomeSet(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}
	for key := range req {
		switch key {
		case "displayname":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.DisplayName{Value: "Calendar Home"})
		case "resourcetype":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "getetag":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "getlastmodified":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "owner":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "current-user-principal":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "principal-url":
			res := Resource{
				UserID:       ctx.Resource.UserID,
				ResourceType: ResourcePrincipal,
			}
			encodedPath, err := h.URLConverter.EncodePath(res)
			if err != nil {
				log.Printf("Failed to encode principal URL for resource %s: %v", ctx.Resource, err)
				return nil, err
			}
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.PrincipalURL{Value: encodedPath})
		case "supported-report-set":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "acl":
			res := Resource{
				UserID:       ctx.Resource.UserID,
				ResourceType: ResourcePrincipal,
			}
			principalPath, err := h.URLConverter.EncodePath(res)
			if err != nil {
				log.Printf("Failed to encode principal URL for resource %s: %v", ctx.Resource, err)
				return nil, err
			}
			ace := propfind.ACE{
				Principal: principalPath,
				Grant:     []string{"read", "write"}, // TODO: complete ACL
				Deny:      []string{},
			}
			acl := propfind.ACL{Aces: []propfind.ACE{ace}}
			req[key] = mo.Ok[propfind.PropertyEncoder](acl)
		case "current-user-privilege-set":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.CurrentUserPrivilegeSet{Privileges: []string{"read", "write"}})
		case "supported-calendar-component-set":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.SupportedCalendarComponentSet{Components: []string{"VEVENT", "VTODO", "VJOURNAL", "VFREEBUSY"}})
		case "supported-calendar-data":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.SupportedCalendarData{
				ContentType: "icalendar",
				Version:     "2.0",
			})
		case "max-resource-size":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.MaxResourceSize{Value: 10485760})
		case "min-date-time":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.MinDateTime{Value: time.Unix(0, 0).UTC()})
		case "max-date-time":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.MaxDateTime{Value: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
		case "max-instances":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.MaxInstances{Value: 100000})
		case "max-attendees-per-instance":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.MaxAttendeesPerInstance{Value: 100})
		case "calendar-home-set":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.CalendarHomeSet{Href: path})
		case "calendar-user-address-set":
			// TODO
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		case "calendar-user-type":
			req[key] = mo.Ok[propfind.PropertyEncoder](propfind.CalendarUserType{Value: "individual"})
		default:
			req[key] = mo.Err[propfind.PropertyEncoder](propfind.ErrNotFound)
		}
	}
	return propfind.EncodeResponse(req, path), nil
}

// handles individual resource
func (h *CaldavHandler) handlePropfindPrincipal(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

// handles individual resource
func (h *CaldavHandler) handlePropfindObject(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) handlePropfindCollection(req propfind.ResponseMap, ctx *RequestContext) (*etree.Document, error) {
	path, err := h.URLConverter.EncodePath(ctx.Resource)
	if err != nil {
		log.Printf("Failed to encode path for resource %s: %v", ctx.Resource, err)
		return nil, err
	}

	return propfind.EncodeResponse(req, path), nil
}

func (h *CaldavHandler) fetchChildren(depth int, parent Resource) (resources []Resource, err error) {
	if depth <= 0 {
		return
	}

	switch parent.ResourceType {
	case ResourceObject, ResourcePrincipal:
		// These types don't have children, return empty slice
		return []Resource{}, nil

	case ResourceCollection:
		// find object (event) paths in the collection
		paths, err := h.Storage.GetObjectPathsInCollection(parent.CalendarID)
		if err != nil {
			log.Printf("Failed to fetch event paths in collection %s: %v", parent.CalendarID, err)
			return nil, err
		}
		for _, path := range paths {
			resource, err := h.URLConverter.ParsePath(path)
			if err != nil {
				log.Printf("Failed to parse path %s: %v", path, err)
				return nil, err
			}
			resources = append(resources, resource)
			children, err := h.fetchChildren(depth-1, resource) // Recursively fetch children for the object
			if err != nil {
				log.Printf("Failed to fetch children for resource %s: %v", resource, err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	case ResourceHomeSet:
		// find collections in the home set
		calendars, err := h.Storage.GetUserCalendars(parent.UserID)
		if err != nil {
			log.Printf("Failed to fetch calendars for user %s: %v", parent.UserID, err)
			return nil, err
		}
		for _, cal := range calendars {
			resource, err := h.URLConverter.ParsePath(cal.Path)
			if err != nil {
				log.Printf("Failed to parse calendar path %s: %v", cal.Path, err)
				return nil, err
			}
			resources = append(resources, resource)
			// Recursively fetch children for the collection
			children, err := h.fetchChildren(depth-1, resource)
			if err != nil {
				log.Printf("Failed to fetch children for resource %s: %v", resource, err)
				return nil, err
			}
			resources = append(resources, children...)
		}
	}
	return
}
