package xml

import "github.com/beevik/etree"

// Namespace definitions for CalDAV and WebDAV
const (
	// DAV is the WebDAV namespace
	DAV = "DAV:"
	// CalDAV is the CalDAV namespace
	CalDAV = "urn:ietf:params:xml:ns:caldav"
	// CalendarServer is the Calendar Server namespace (used by some implementations)
	CalendarServer = "http://calendarserver.org/ns/"
)

// AddNamespaces adds standard CalDAV namespaces to the XML document
func AddNamespaces(doc *etree.Document) {
	root := doc.Root()
	if root == nil {
		return
	}
	root.CreateAttr("xmlns:D", DAV)
	root.CreateAttr("xmlns:C", CalDAV)
	root.CreateAttr("xmlns:CS", CalendarServer)
}
