package handlers

import (
	"net/http"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/internal/xml"
	"github.com/cyp0633/libcaldora/server/auth"
	"github.com/cyp0633/libcaldora/server/storage"
)

// handlePropfind handles PROPFIND requests
func (r *Router) handlePropfind(w http.ResponseWriter, req *http.Request) {
	// Parse resource path
	path := StripPrefix(req.URL.Path, r.baseURI)

	// Read request body
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(req.Body); err != nil {
		r.logger.Error("failed to read PROPFIND request body",
			"error", err,
			"path", path)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse PROPFIND request
	var propfind xml.PropfindRequest
	if err := propfind.Parse(doc); err != nil {
		r.logger.Error("failed to parse PROPFIND request",
			"error", err,
			"path", path)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build multistatus response
	response := &xml.MultistatusResponse{
		Responses: []xml.Response{
			{
				Href:      r.baseURI + path,
				PropStats: []xml.PropStat{},
			},
		},
	}

	// Get requested properties
	props := propfind.Prop
	if propfind.AllProp {
		// Standard properties to include for allprop
		props = []string{
			xml.TagResourcetype,
			"getcontenttype",
			"displayname",
			"current-user-principal",
		}
		props = append(props, propfind.Include...)
	}

	// Handle root path specially
	if path == "" || path == "/" {
		// Get authenticated principal
		principal := auth.GetPrincipalFromContext(req.Context())
		if principal == nil {
			r.logger.Error("no authenticated principal in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Split properties into found and not found
		foundProps := []xml.Property{}
		notFoundProps := []xml.Property{}

		for _, prop := range props {
			switch prop {
			case xml.TagResourcetype:
				// Root is a collection
				foundProps = append(foundProps, xml.Property{
					Name:      xml.TagResourcetype,
					Namespace: xml.DAV,
					Children: []xml.Property{
						{Name: xml.TagCollection, Namespace: xml.DAV},
					},
				})
			case "current-user-principal":
				// Return the principal URL for the authenticated user
				foundProps = append(foundProps, xml.Property{
					Name:      "current-user-principal",
					Namespace: xml.DAV,
					Children: []xml.Property{
						{
							Name:        "href",
							Namespace:   xml.DAV,
							TextContent: r.baseURI + "/u/" + principal.ID,
						},
					},
				})
			default:
				// Other properties not found on root
				notFoundProps = append(notFoundProps, xml.Property{
					Name:      prop,
					Namespace: xml.DAV,
				})
			}
		}

		// Add found properties
		if len(foundProps) > 0 {
			response.Responses[0].PropStats = append(response.Responses[0].PropStats, xml.PropStat{
				Props:  foundProps,
				Status: "HTTP/1.1 200 OK",
			})
		}

		// Add not found properties
		if len(notFoundProps) > 0 {
			response.Responses[0].PropStats = append(response.Responses[0].PropStats, xml.PropStat{
				Props:  notFoundProps,
				Status: "HTTP/1.1 404 Not Found",
			})
		}
	} else {
		// Non-root paths
		_, err := storage.ParseResourcePath(path)
		if err != nil {
			r.logger.Error("invalid resource path in PROPFIND request",
				"error", err,
				"path", path)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// TODO: Handle non-root paths
	}

	// Convert response to XML and send
	respDoc := response.ToXML()
	w.Header().Set(HeaderContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	respDoc.WriteTo(w)
}
