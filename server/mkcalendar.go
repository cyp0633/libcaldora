package server

import (
	"io"
	"log"
	"net/http"

	"github.com/cyp0633/libcaldora/internal/xml/mkcalendar"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
)

func (h *CaldavHandler) handleMkCalendar(w http.ResponseWriter, r *http.Request, ctx *RequestContext) {
	log.Printf("MKCALENDAR/MKCOL received for %s (User: %s, Calendar: %s, Object: %s)",
		ctx.Resource.ResourceType, ctx.Resource.UserID, ctx.Resource.CalendarID, ctx.Resource.ObjectID)
	if ctx.Resource.ResourceType != storage.ResourceCollection {
		http.Error(w, "Method Not Allowed: MKCALENDAR can only be used to create a calendar collection", http.StatusMethodNotAllowed)
		return
	}

	// parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	properties, err := mkcalendar.ParseRequest(string(bodyBytes))
	if err != nil {
		log.Printf("Failed to parse MKCALENDAR request: %v", err)
		http.Error(w, "Failed to parse MKCALENDAR request", http.StatusBadRequest)
		return
	}

	cal := &storage.Calendar{
		SupportedComponents: []string{}, // Initialize to avoid nil
	}

	// Default to a basic VCALENDAR structure
	cal.CalendarData = ical.NewCalendar()
	cal.CalendarData.Props.SetText(ical.PropProductID, "-//libcaldora//CalDAV Server//EN")
	cal.CalendarData.Props.SetText(ical.PropVersion, "2.0")

	// Process provided properties
	for key, prop := range properties {
		switch key {
		case "displayname":
			if dn, ok := prop.(*props.DisplayName); ok && dn.Value != "" {
				cal.CalendarData.Props.SetText(ical.PropName, dn.Value)
				log.Printf("Setting calendar name: %s", dn.Value)
			}
		case "calendar-description":
			if desc, ok := prop.(*props.CalendarDescription); ok && desc.Value != "" {
				cal.CalendarData.Props.SetText(ical.PropDescription, desc.Value)
				log.Printf("Setting calendar description: %s", desc.Value)
			}
		case "calendar-timezone":
			if tz, ok := prop.(*props.CalendarTimezone); ok && tz.Value != "" {
				// Store the timezone string in calendar data
				// This is a simplification - proper timezone parsing would be better
				vtimezone := &ical.Component{
					Name:  ical.CompTimezone,
					Props: make(ical.Props),
				}
				vtimezone.Props.SetText(ical.PropTimezoneID, tz.Value)
				cal.CalendarData.Children = append(cal.CalendarData.Children, vtimezone)
				log.Printf("Setting calendar timezone: %s", tz.Value)
			}
		case "supported-calendar-component-set":
			if compSet, ok := prop.(*props.SupportedCalendarComponentSet); ok && len(compSet.Components) > 0 {
				cal.SupportedComponents = compSet.Components
				log.Printf("Setting supported components: %v", compSet.Components)
			}
		case "calendar-color", "color":
			// Handle both Apple and Google color properties
			var colorValue string
			if csColor, ok := prop.(*props.CalendarColor); ok && csColor.Value != "" {
				colorValue = csColor.Value
			} else if gColor, ok := prop.(*props.Color); ok && gColor.Value != "" {
				colorValue = gColor.Value
			}

			if colorValue != "" {
				cal.CalendarData.Props.SetText(ical.PropColor, colorValue)
				log.Printf("Setting calendar color: %s", colorValue)
			}
		case "timezone":
			// Google specific timezone
			if tz, ok := prop.(*props.Timezone); ok && tz.Value != "" {
				// Store in a custom property or handle as needed
				cal.CalendarData.Props.SetText("X-TIMEZONE", tz.Value)
				log.Printf("Setting Google timezone: %s", tz.Value)
			}
		default:
			// Ignore unknown or unsupported properties
			log.Printf("Ignoring unsupported property: %s", key)
		}
	}

	// Ensure we have required properties
	if len(cal.SupportedComponents) == 0 {
		// Default to supporting VEVENT if not specified
		cal.SupportedComponents = []string{"VEVENT"}
		log.Printf("No component set specified, defaulting to VEVENT")
	}

	err = h.Storage.CreateCalendar(ctx.Resource.UserID, cal)
	if err != nil {
		log.Printf("Failed to create calendar: %v", err)
		http.Error(w, "Failed to create calendar", http.StatusInternalServerError)
		return
	}
	if cal.ETag == "" || cal.Path == "" {
		log.Printf("Failed to create calendar: ETag or Path is empty")
		http.Error(w, "Failed to create calendar", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", cal.Path)
	w.Header().Set("ETag", cal.ETag)
	w.WriteHeader(http.StatusCreated)
}
