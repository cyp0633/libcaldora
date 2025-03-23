package xml

import (
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

// normalizeXML removes whitespace differences and XML declaration for test comparisons
func normalizeXML(s string) string {
	// First remove the XML declaration
	s = regexp.MustCompile(`<\?xml[^>]*\?>`).ReplaceAllString(s, "")

	// Remove all whitespace between elements
	s = regexp.MustCompile(`>\s+<`).ReplaceAllString(s, "><")

	// Remove all leading and trailing whitespace in text nodes
	s = regexp.MustCompile(`>\s+([^<>\s]+)\s+<`).ReplaceAllString(s, ">$1<")

	// Remove all standalone whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	// Remove spaces between tags
	s = regexp.MustCompile(`>\s+<`).ReplaceAllString(s, "><")
	s = regexp.MustCompile(`\s+/>`).ReplaceAllString(s, "/>")
	s = regexp.MustCompile(`<\s+`).ReplaceAllString(s, "<")
	s = regexp.MustCompile(`\s+>`).ReplaceAllString(s, ">")

	// Convert to lowercase to ignore case differences
	s = strings.ToLower(s)

	// Remove all whitespace at the beginning and end
	return strings.TrimSpace(s)
}

// elementToString converts an etree.Element to a string for testing
func elementToString(elem *etree.Element) string {
	doc := etree.NewDocument()
	doc.AddChild(elem.Copy())
	s, _ := doc.WriteToString()

	// Remove XML declaration
	s = regexp.MustCompile(`<\?xml[^>]*\?>`).ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
