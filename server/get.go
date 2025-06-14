package server

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
)

func (h *CaldavHandler) handleGet(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	h.Logger.Info("get request received",
		"resource_type", ctx.Resource.ResourceType,
		"user_id", ctx.Resource.UserID,
		"calendar_id", ctx.Resource.CalendarID,
		"object_id", ctx.Resource.ObjectID)

	if ctx.Resource.ResourceType != storage.ResourceObject {
		// Technically GET might be allowed on collections by some servers (listing?), but often not.
		// GET on Principal/HomeSet is unusual in CalDAV.
		h.Logger.Warn("get not supported on this resource type",
			"resource_type", ctx.Resource.ResourceType)
		http.Error(w, "Method Not Allowed on this resource type (or GET not implemented)", http.StatusMethodNotAllowed)
		return
	}

	// get object
	object, err := h.Storage.GetObject(ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if err != nil || object == nil || len(object.Component) == 0 {
		h.Logger.Error("failed to retrieve object",
			"error", err,
			"object_id", ctx.Resource.ObjectID)
		http.Error(w, "Internal Server Error: Unable to retrieve object", http.StatusInternalServerError)
		return
	}

	// judge etag
	if etag := r.Header.Get("If-None-Match"); etag == object.ETag {
		// ETag matches, return 304 Not Modified
		h.Logger.Info("returning not modified due to etag match",
			"etag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// get associated collection
	collection, err := h.Storage.GetCalendar(ctx.Resource.UserID, ctx.Resource.CalendarID)
	if err != nil || collection == nil || collection.CalendarData == nil {
		h.Logger.Error("failed to retrieve calendar collection",
			"error", err,
			"calendar_id", ctx.Resource.CalendarID)
		http.Error(w, "Internal Server Error: Unable to retrieve calendar collection", http.StatusInternalServerError)
		return
	}

	// wrap event into calendar
	for _, component := range object.Component {
		if component != nil {
			collection.CalendarData.Children = append(collection.CalendarData.Children, component)
		}
	}

	// Ensure PRODID and VERSION are set to avoid encoding errors
	if _, err := collection.CalendarData.Props.Text(ical.PropProductID); err != nil {
		collection.CalendarData.Props.SetText(ical.PropProductID, "-//libcaldora//NONSGML v1.0//EN")
	}
	if _, err := collection.CalendarData.Props.Text(ical.PropVersion); err != nil {
		collection.CalendarData.Props.SetText(ical.PropVersion, "2.0")
	}

	// Ensure DTSTAMP is set in all VEVENT components
	for _, child := range collection.CalendarData.Children {
		if child.Name == ical.CompEvent {
			if _, err := child.Props.DateTime(ical.PropDateTimeStamp, nil); err != nil {
				// Missing DTSTAMP, set it to now
				child.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
			}
		}
	}

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(collection.CalendarData); err != nil {
		h.Logger.Error("failed to encode calendar",
			"error", err)
		http.Error(w, "Internal Server Error: Failed to encode calendar", http.StatusInternalServerError)
		return
	}

	// Set the content type and return the ICS data
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(buf.Bytes())))
	w.Header().Set("ETag", object.ETag)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buf.Bytes())
	if err != nil {
		h.Logger.Error("failed to write response",
			"error", err)
	}
}
