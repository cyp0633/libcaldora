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
	if root.Tag != TagMultistatus {
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
	// Create root element with prefixed namespace for multistatus responses
	root := CreateRootElement(doc, TagMultistatus, true)
	AddSelectedNamespaces(doc, DAV, CalDAV, CalendarServer, AppleICal)

	for _, resp := range m.Responses {
		response := CreateElementWithNS(root, TagResponse)
		href := CreateElementWithNS(response, TagHref)
		href.SetText(resp.Href)

		if resp.Error != nil {
			response.AddChild(resp.Error.ToElement())
		} else if resp.Status != "" {
			status := CreateElementWithNS(response, TagStatus)
			status.SetText(resp.Status)
		} else {
			for _, propstat := range resp.PropStats {
				ps := CreateElementWithNS(response, TagPropstat)
				prop := CreateElementWithNS(ps, TagProp)

				for _, p := range propstat.Props {
					prop.AddChild(p.ToElement())
				}

				status := CreateElementWithNS(ps, TagStatus)
				status.SetText(propstat.Status)
			}
		}
	}

	return doc
}
