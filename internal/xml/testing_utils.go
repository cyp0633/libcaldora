package xml

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// elementToString returns a string representation of an element for debugging
func elementToString(e *etree.Element) string {
	doc := etree.NewDocument()
	doc.AddChild(e)
	s, _ := doc.WriteToString()
	return s
}

// normalizeXML removes XML declaration and normalizes whitespace
func normalizeXML(s string) string {
	// Remove XML declaration
	if idx := strings.Index(s, "?>"); idx != -1 {
		s = s[idx+2:]
	}
	// Remove whitespace and newlines
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), "")
	// Additional debugging check
	if strings.Contains(s, "\n") {
		fmt.Printf("Warning: String still contains newlines after normalization: %q\n", s)
	}
	return s
}
