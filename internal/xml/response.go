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
					prop.AddChild(p.ToElement())
				}

				status := ps.CreateElement(TagStatus)
				status.SetText(propstat.Status)
			}
		}
	}

	return doc
}

// HTTPStatus converts a status code to a WebDAV status string
func HTTPStatus(code int) string {
	switch code {
	case 200:
		return "HTTP/1.1 200 OK"
	case 404:
		return "HTTP/1.1 404 Not Found"
	case 207:
		return "HTTP/1.1 207 Multi-Status"
	case 403:
		return "HTTP/1.1 403 Forbidden"
	default:
		return fmt.Sprintf("HTTP/1.1 %d Status", code)
	}
}
