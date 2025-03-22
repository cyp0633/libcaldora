package xml

import (
	"fmt"

	"github.com/beevik/etree"
)

// MultistatusResponse represents a multistatus response
type MultistatusResponse struct {
	Responses []Response
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

	for _, respElem := range root.SelectElements("response") {
		resp := Response{}

		// Parse href
		if hrefElem := respElem.SelectElement("href"); hrefElem != nil {
			resp.Href = hrefElem.Text()
		}

		// Parse error if present
		if errorElem := respElem.SelectElement("error"); errorElem != nil {
			if child := errorElem.ChildElements(); len(child) > 0 {
				resp.Error = &Error{
					Tag:       child[0].Tag,
					Namespace: child[0].Space,
					Message:   child[0].Text(),
				}
				if resp.Error.Namespace == "D" {
					resp.Error.Namespace = DAV
				}
			}
		} else {
			// Parse propstat elements
			for _, propstatElem := range respElem.SelectElements("propstat") {
				propstat := PropStat{}

				// Parse properties
				if propElem := propstatElem.SelectElement("prop"); propElem != nil {
					for _, prop := range propElem.ChildElements() {
						property := Property{}
						property.FromElement(prop)
						// Convert "D" namespace to "DAV:"
						if property.Namespace == "D" {
							property.Namespace = DAV
						}
						propstat.Props = append(propstat.Props, property)
					}
				}

				// Parse status
				if statusElem := propstatElem.SelectElement("status"); statusElem != nil {
					propstat.Status = statusElem.Text()
				}

				resp.PropStats = append(resp.PropStats, propstat)
			}
		}

		m.Responses = append(m.Responses, resp)
	}

	return nil
}

// ToXML converts a MultistatusResponse to an XML document
func (m *MultistatusResponse) ToXML() *etree.Document {
	doc := etree.NewDocument()
	root := doc.CreateElement(TagMultistatus)
	AddNamespaces(doc)

	for _, resp := range m.Responses {
		response := root.CreateElement(TagResponse)
		href := response.CreateElement(TagHref)
		href.SetText(resp.Href)

		if resp.Error != nil {
			response.AddChild(resp.Error.ToElement())
		} else if resp.Status != "" {
			status := response.CreateElement(TagStatus)
			status.SetText(resp.Status)
		} else {
			for _, propstat := range resp.PropStats {
				ps := response.CreateElement(TagPropstat)
				prop := ps.CreateElement(TagProp)

				for _, p := range propstat.Props {
					elem := p.ToElement()
					// Add namespace prefix only for resourcetype and collection-related elements
					needsNamespace := p.Name == "resourcetype" || p.Name == "collection" || p.Name == "calendar"
					if needsNamespace && p.Namespace != "" {
						switch p.Namespace {
						case DAV:
							elem.Space = "D"
						case CalDAV:
							elem.Space = "C"
						case CalendarServer:
							elem.Space = "CS"
						}
					}

					// For resourcetype's children, also set the namespace
					for _, child := range elem.ChildElements() {
						if child.Space == DAV {
							child.Space = "D"
						} else if child.Space == CalDAV {
							child.Space = "C"
						} else if child.Space == CalendarServer {
							child.Space = "CS"
						}
					}

					prop.AddChild(elem)
				}

				status := ps.CreateElement(TagStatus)
				status.SetText(propstat.Status)
			}
		}
	}

	return doc
}
