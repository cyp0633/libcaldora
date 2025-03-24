package xml

import (
	"fmt"

	"github.com/beevik/etree"
)

// Namespace definitions for CalDAV and related protocols
const (
	// DAV is the WebDAV namespace
	DAV = "DAV:"
	// CalDAV is the CalDAV namespace
	CalDAV = "urn:ietf:params:xml:ns:caldav"
	// CalendarServer is the Calendar Server namespace (used by some implementations)
	CalendarServer = "http://calendarserver.org/ns/"
	// AppleICal is the Apple iCal namespace
	AppleICal = "http://apple.com/ns/ical/"
	// CardDAV is the CardDAV namespace
	CardDAV = "urn:ietf:params:xml:ns:carddav"
	// ICal is the iCalendar namespace
	ICal = "http://www.w3.org/2002/12/cal/ical#"
)

// NamespacePrefix maps namespace URIs to their standard prefixes
var NamespacePrefix = map[string]string{
	DAV:            "D",
	CalDAV:         "C",
	CalendarServer: "CS",
	AppleICal:      "A",
	CardDAV:        "CARD",
	ICal:           "ICAL",
}

// PrefixNamespace is the inverse of NamespacePrefix
var PrefixNamespace = map[string]string{
	"D":    DAV,
	"C":    CalDAV,
	"CS":   CalendarServer,
	"A":    AppleICal,
	"CARD": CardDAV,
	"ICAL": ICal,
}

// ElementNamespaces maps element names to their default namespaces
var ElementNamespaces = map[string]string{
	// DAV elements
	"multistatus":            DAV,
	"response":               DAV,
	"href":                   DAV,
	"propstat":               DAV,
	"status":                 DAV,
	"prop":                   DAV,
	"resourcetype":           DAV,
	"displayname":            DAV,
	"getetag":                DAV,
	"sync-token":             DAV,
	"collection":             DAV,
	"current-user-principal": DAV,
	"principal-URL":          DAV,

	// CalDAV elements
	"calendar":             CalDAV,
	"calendar-data":        CalDAV,
	"calendar-query":       CalDAV,
	"calendar-multiget":    CalDAV,
	"free-busy-query":      CalDAV,
	"filter":               CalDAV,
	"comp-filter":          CalDAV,
	"time-range":           CalDAV,
	"supported-report-set": CalDAV,

	// Apple iCal elements
	"calendar-color": AppleICal,
	"calendar-order": AppleICal,

	// CalendarServer elements
	"getctag":               CalendarServer,
	"invite":                CalendarServer,
	"allowed-sharing-modes": CalendarServer,
}

// NamespaceError represents a namespace-related error
type NamespaceError struct {
	Message string
	URI     string
}

func (e *NamespaceError) Error() string {
	return fmt.Sprintf("namespace error: %s (URI: %s)", e.Message, e.URI)
}

// RegisterNamespace adds a new namespace with prefix to the registry
func RegisterNamespace(uri, prefix string) error {
	if uri == "" {
		return &NamespaceError{Message: "empty namespace URI", URI: uri}
	}
	if prefix == "" {
		return &NamespaceError{Message: "empty prefix", URI: uri}
	}
	if _, exists := NamespacePrefix[uri]; exists {
		return &NamespaceError{Message: "namespace already registered", URI: uri}
	}
	if _, exists := PrefixNamespace[prefix]; exists {
		return &NamespaceError{Message: "prefix already in use", URI: uri}
	}

	NamespacePrefix[uri] = prefix
	PrefixNamespace[prefix] = uri
	return nil
}

// RegisterElement associates an element name with a namespace
func RegisterElement(elemName, namespace string) error {
	if elemName == "" {
		return &NamespaceError{Message: "empty element name", URI: namespace}
	}
	if namespace == "" {
		return &NamespaceError{Message: "empty namespace", URI: namespace}
	}
	if _, exists := NamespacePrefix[namespace]; !exists {
		return &NamespaceError{Message: "namespace not registered", URI: namespace}
	}

	ElementNamespaces[elemName] = namespace
	return nil
}

// GetNamespacePrefix returns the prefix for a namespace URI
func GetNamespacePrefix(namespaceURI string) string {
	prefix, ok := NamespacePrefix[namespaceURI]
	if !ok {
		return ""
	}
	return prefix
}

// GetElementNamespace returns the namespace for an element
func GetElementNamespace(elemName string) string {
	ns, ok := ElementNamespaces[elemName]
	if !ok {
		return DAV // Default to DAV namespace
	}
	return ns
}

// GetElementPrefix returns the prefix for an element
func GetElementPrefix(elemName string) string {
	ns := GetElementNamespace(elemName)
	return GetNamespacePrefix(ns)
}

// AddNamespaces adds all registered namespaces to the XML document
func AddNamespaces(doc *etree.Document) {
	root := doc.Root()
	if root == nil {
		return
	}

	// Add all namespaces from registry
	for uri, prefix := range NamespacePrefix {
		root.CreateAttr("xmlns:"+prefix, uri)
	}
}

// AddSelectedNamespaces adds only the specified namespaces to the document
func AddSelectedNamespaces(doc *etree.Document, namespaces ...string) {
	root := doc.Root()
	if root == nil {
		return
	}

	// If no namespaces specified, add all
	if len(namespaces) == 0 {
		AddNamespaces(doc)
		return
	}

	// Add only requested namespaces
	for _, ns := range namespaces {
		if prefix, ok := NamespacePrefix[ns]; ok {
			root.CreateAttr("xmlns:"+prefix, ns)
		}
	}
}

// CreateElementWithNS creates an element with the appropriate namespace
func CreateElementWithNS(parent *etree.Element, name string) *etree.Element {
	return CreateElementWithNSPrefix(parent, name, true)
}

// CreateElementWithNSPrefix creates an element with configurable namespace prefix
func CreateElementWithNSPrefix(parent *etree.Element, name string, applyPrefix bool) *etree.Element {
	elem := parent.CreateElement(name)
	if applyPrefix {
		elem.Space = GetElementPrefix(name)
	}
	return elem
}

// CreateElementWithCustomNS creates an element with a specified namespace
func CreateElementWithCustomNS(parent *etree.Element, name, namespace string) *etree.Element {
	elem := parent.CreateElement(name)
	if prefix := GetNamespacePrefix(namespace); prefix != "" {
		elem.Space = prefix
	}
	return elem
}

// CreateRootElement creates a root element for a document with configurable namespace prefix
func CreateRootElement(doc *etree.Document, name string, applyPrefix bool) *etree.Element {
	elem := doc.CreateElement(name)
	if applyPrefix {
		elem.Space = GetElementPrefix(name)
	}
	return elem
}

// FindElementWithNS finds an element by name, accounting for namespaces
func FindElementWithNS(parent *etree.Element, name string) *etree.Element {
	prefix := GetElementPrefix(name)
	var found *etree.Element

	// Try with namespace prefix first
	if prefix != "" {
		prefixedName := prefix + ":" + name
		found = parent.FindElement(prefixedName)
		if found != nil {
			return found
		}
	}

	// Try without prefix
	found = parent.FindElement(name)
	return found
}
