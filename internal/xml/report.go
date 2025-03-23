package xml

import (
	"fmt"
	"time"

	"github.com/beevik/etree"
)

// TimeRange represents a time range filter
type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

func (tr *TimeRange) toElement(elem *etree.Element) {
	if tr.Start != nil {
		elem.CreateAttr("start", tr.Start.Format("20060102T150405Z"))
	}
	if tr.End != nil {
		elem.CreateAttr("end", tr.End.Format("20060102T150405Z"))
	}
}

// Filter represents a calendar query filter
type Filter struct {
	ComponentName string
	SubFilter     *Filter
	TimeRange     *TimeRange
}

func (f *Filter) toElement(elem *etree.Element) {
	compFilter := CreateElementWithNS(elem, "comp-filter")
	compFilter.CreateAttr("name", f.ComponentName)

	if f.TimeRange != nil {
		tr := CreateElementWithNS(compFilter, "time-range")
		f.TimeRange.toElement(tr)
	}

	if f.SubFilter != nil {
		f.SubFilter.toElement(compFilter)
	}
}

// CalendarQuery represents a calendar-query REPORT request
type CalendarQuery struct {
	Props  []string
	Filter Filter
}

// CalendarMultiget represents a calendar-multiget REPORT request
type CalendarMultiget struct {
	Props []string
	Hrefs []string
}

// FreeBusyQuery represents a free-busy-query REPORT request
type FreeBusyQuery struct {
	TimeRange TimeRange
}

// ReportRequest represents a REPORT request
type ReportRequest struct {
	Query    *CalendarQuery
	MultiGet *CalendarMultiget
	FreeBusy *FreeBusyQuery
}

// Parse parses a REPORT request from an XML document
func (r *ReportRequest) Parse(doc *etree.Document) error {
	if doc == nil || doc.Root() == nil {
		return fmt.Errorf("empty document")
	}

	root := doc.Root()

	switch root.Tag {
	case "calendar-query":
		return r.parseCalendarQuery(root)
	case "calendar-multiget":
		return r.parseCalendarMultiget(root)
	case "free-busy-query":
		return r.parseFreeBusyQuery(root)
	default:
		return fmt.Errorf("unsupported report type: %s", root.Tag)
	}
}

func (r *ReportRequest) parseCalendarQuery(root *etree.Element) error {
	r.Query = &CalendarQuery{}

	// Parse prop element
	if prop := FindElementWithNS(root, "prop"); prop != nil {
		for _, p := range prop.ChildElements() {
			r.Query.Props = append(r.Query.Props, p.Tag)
		}
	}

	// Parse filter element
	if filter := FindElementWithNS(root, "filter"); filter != nil {
		if err := r.parseFilter(filter, &r.Query.Filter); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReportRequest) parseFilter(elem *etree.Element, filter *Filter) error {
	if compFilter := FindElementWithNS(elem, "comp-filter"); compFilter != nil {
		filter.ComponentName = compFilter.SelectAttrValue("name", "")

		// Parse time-range if present
		if tr := FindElementWithNS(compFilter, "time-range"); tr != nil {
			filter.TimeRange = &TimeRange{}
			if start := tr.SelectAttrValue("start", ""); start != "" {
				t, _ := time.Parse("20060102T150405Z", start)
				filter.TimeRange.Start = &t
			}
			if end := tr.SelectAttrValue("end", ""); end != "" {
				t, _ := time.Parse("20060102T150405Z", end)
				filter.TimeRange.End = &t
			}
		}

		// Parse nested comp-filter if present
		if nested := FindElementWithNS(compFilter, "comp-filter"); nested != nil {
			filter.SubFilter = &Filter{}
			return r.parseFilter(compFilter, filter.SubFilter)
		}
	}

	return nil
}

func (r *ReportRequest) parseCalendarMultiget(root *etree.Element) error {
	r.MultiGet = &CalendarMultiget{}

	// Parse prop element
	if prop := FindElementWithNS(root, "prop"); prop != nil {
		for _, p := range prop.ChildElements() {
			r.MultiGet.Props = append(r.MultiGet.Props, p.Tag)
		}
	}

	// Parse href elements
	for _, href := range root.SelectElements("href") {
		r.MultiGet.Hrefs = append(r.MultiGet.Hrefs, href.Text())
	}

	return nil
}

func (r *ReportRequest) parseFreeBusyQuery(root *etree.Element) error {
	r.FreeBusy = &FreeBusyQuery{}

	// Parse time-range element
	if tr := FindElementWithNS(root, "time-range"); tr != nil {
		if start := tr.SelectAttrValue("start", ""); start != "" {
			t, _ := time.Parse("20060102T150405Z", start)
			r.FreeBusy.TimeRange.Start = &t
		}
		if end := tr.SelectAttrValue("end", ""); end != "" {
			t, _ := time.Parse("20060102T150405Z", end)
			r.FreeBusy.TimeRange.End = &t
		}
	}

	return nil
}

// ToXML converts a ReportRequest to an XML document
func (r *ReportRequest) ToXML() *etree.Document {
	doc := etree.NewDocument()
	var root *etree.Element

	// Create appropriate root element and add namespaces
	switch {
	case r.Query != nil:
		root = CreateRootElement(doc, "calendar-query", true)
		AddSelectedNamespaces(doc, DAV, CalDAV)

		// Add prop element
		prop := CreateElementWithNS(root, "prop")
		for _, p := range r.Query.Props {
			CreateElementWithNS(prop, p)
		}

		// Add filter element
		filter := CreateElementWithNS(root, "filter")
		r.Query.Filter.toElement(filter)

	case r.MultiGet != nil:
		root = CreateRootElement(doc, "calendar-multiget", true)
		AddSelectedNamespaces(doc, DAV, CalDAV)

		// Add prop element
		prop := CreateElementWithNS(root, "prop")
		for _, p := range r.MultiGet.Props {
			CreateElementWithNS(prop, p)
		}

		// Add href elements
		for _, href := range r.MultiGet.Hrefs {
			hrefElem := CreateElementWithNS(root, "href")
			hrefElem.SetText(href)
		}

	case r.FreeBusy != nil:
		root = CreateRootElement(doc, "free-busy-query", true)
		AddSelectedNamespaces(doc, CalDAV)

		// Add time-range element
		tr := CreateElementWithNS(root, "time-range")
		r.FreeBusy.TimeRange.toElement(tr)
	}

	return doc
}
