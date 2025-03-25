package xml

import (
	"fmt"

	"github.com/beevik/etree"
)

// MultistatusResponse represents a multistatus response
type MultistatusResponse struct {
	Responses []Response
	SyncToken string
}

// Response represents a single response within a multistatus
type Response struct {
	Href      string
	PropStats []PropStat
	Error     *Error
	Status    string
}

// PropStat represents property status in a response
type PropStat struct {
	Props  []Property
	Status string
}

// Parse parses a multistatus response from an XML document
func (m *MultistatusResponse) Parse(doc *etree.Document) error {
	if doc == nil || doc.Root() == nil {
		return fmt.Errorf("empty document")
	}

	root := doc.Root()
	if root.Tag != TagMultistatus && root.Space+":"+root.Tag != GetElementPrefix(TagMultistatus)+":"+TagMultistatus {
		return fmt.Errorf("invalid root tag: %s", root.Tag)
	}

	m.Responses = nil // Reset responses

	// Try both prefixed and unprefixed response elements
	responseElements := root.SelectElements(GetElementPrefix(TagResponse) + ":" + TagResponse)
	if len(responseElements) == 0 {
		// If no prefixed elements found, try unprefixed
		responseElements = root.SelectElements(TagResponse)
	}
	for _, respElem := range responseElements {
		resp := Response{}

		// Parse href
		if hrefElem := FindElementWithNS(respElem, TagHref); hrefElem != nil {
			resp.Href = hrefElem.Text()
		}

		// Parse error if present
		if errorElem := FindElementWithNS(respElem, TagError); errorElem != nil {
			if child := errorElem.ChildElements(); len(child) > 0 {
				resp.Error = &Error{
					Tag:       child[0].Tag,
					Namespace: GetElementNamespace(child[0].Tag),
					Message:   child[0].Text(),
				}
			}
		} else {
			// Parse propstat elements
			// Try both prefixed and unprefixed propstat elements
			propstatElements := respElem.SelectElements(GetElementPrefix(TagPropstat) + ":" + TagPropstat)
			if len(propstatElements) == 0 {
				// If no prefixed elements found, try unprefixed
				propstatElements = respElem.SelectElements(TagPropstat)
			}
			for _, propstatElem := range propstatElements {
				propstat := PropStat{}

				// Parse properties
				if propElem := FindElementWithNS(propstatElem, TagProp); propElem != nil {
					for _, prop := range propElem.ChildElements() {
						property := Property{}
						property.FromElement(prop)
						propstat.Props = append(propstat.Props, property)
					}
				}

				// Parse status
				if statusElem := FindElementWithNS(propstatElem, TagStatus); statusElem != nil {
					propstat.Status = statusElem.Text()
				}

				resp.PropStats = append(resp.PropStats, propstat)
			}
		}

		m.Responses = append(m.Responses, resp)
	}

	// Parse sync-token if present
	if token := FindElementWithNS(root, "sync-token"); token != nil {
		m.SyncToken = token.Text()
	}

	return nil
}

// ToXML converts a MultistatusResponse to an XML document
func (m *MultistatusResponse) ToXML() *etree.Document {
	doc := etree.NewDocument()

	// Check if we need to use a default namespace for DAV
	// This is specifically for the caldav home response case
	var useDefaultDavNs bool
	if len(m.Responses) == 1 && len(m.Responses[0].PropStats) == 1 {
		if len(m.Responses[0].PropStats[0].Props) == 1 {
			if m.Responses[0].PropStats[0].Props[0].Name == "calendar-home-set" {
				useDefaultDavNs = true
			}
		}
	}

	// Create root element
	var root *etree.Element
	if useDefaultDavNs {
		// Use default namespace for DAV
		root = doc.CreateElement(TagMultistatus)
		root.CreateAttr("xmlns", DAV)
	} else {
		// Use prefixed namespace as usual
		root = CreateRootElement(doc, TagMultistatus, true)
		AddSelectedNamespaces(doc, DAV, CalDAV, CalendarServer)
	}

	// Determine which additional namespaces are needed
	neededNamespaces := []string{CalDAV}

	// Check if any Apple iCal elements are present
	for _, resp := range m.Responses {
		for _, propstat := range resp.PropStats {
			for _, prop := range propstat.Props {
				if prop.Name == "calendar-color" || prop.Name == "calendar-order" ||
					prop.Namespace == AppleICal {
					neededNamespaces = append(neededNamespaces, AppleICal)
					break
				}
			}
		}
	}

	// Add additional namespaces but not DAV if it's already set as default
	for _, ns := range neededNamespaces {
		if ns != DAV || !useDefaultDavNs {
			if prefix := GetNamespacePrefix(ns); prefix != "" {
				root.CreateAttr("xmlns:"+prefix, ns)
			}
		}
	}

	// Rest of the original implementation
	// ...build the response elements
	for _, resp := range m.Responses {
		// Create response element based on namespace style
		var response *etree.Element
		if useDefaultDavNs {
			response = root.CreateElement(TagResponse)
		} else {
			response = CreateElementWithNS(root, TagResponse)
		}

		// Create href element based on namespace style
		var href *etree.Element
		if useDefaultDavNs {
			href = response.CreateElement(TagHref)
		} else {
			href = CreateElementWithNS(response, TagHref)
		}
		href.SetText(resp.Href)

		if resp.Error != nil {
			response.AddChild(resp.Error.ToElement())
		} else if resp.Status != "" {
			var status *etree.Element
			if useDefaultDavNs {
				status = response.CreateElement(TagStatus)
			} else {
				status = CreateElementWithNS(response, TagStatus)
			}
			status.SetText(resp.Status)
		} else {
			for _, propstat := range resp.PropStats {
				var ps *etree.Element
				if useDefaultDavNs {
					ps = response.CreateElement(TagPropstat)
				} else {
					ps = CreateElementWithNS(response, TagPropstat)
				}

				var prop *etree.Element
				if useDefaultDavNs {
					prop = ps.CreateElement(TagProp)
				} else {
					prop = CreateElementWithNS(ps, TagProp)
				}

				for _, p := range propstat.Props {
					prop.AddChild(p.ToElement())
				}

				var status *etree.Element
				if useDefaultDavNs {
					status = ps.CreateElement(TagStatus)
				} else {
					status = CreateElementWithNS(ps, TagStatus)
				}
				status.SetText(propstat.Status)
			}
		}
	}

	return doc
}
