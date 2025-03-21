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
	compFilter := elem.CreateElement("comp-filter")
	compFilter.Space = "C"
	compFilter.CreateAttr("name", f.ComponentName)

	if f.TimeRange != nil {
		tr := compFilter.CreateElement("time-range")
		tr.Space = "C"
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
	if prop := root.FindElement("D:prop"); prop != nil {
		for _, p := range prop.ChildElements() {
			r.Query.Props = append(r.Query.Props, p.Tag)
		}
	}

	// Parse filter element
	if filter := root.FindElement("C:filter"); filter != nil {
		if err := r.parseFilter(filter, &r.Query.Filter); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReportRequest) parseFilter(elem *etree.Element, filter *Filter) error {
	if compFilter := elem.SelectElement("comp-filter"); compFilter != nil {
		filter.ComponentName = compFilter.SelectAttrValue("name", "")

		// Parse time-range if present
		if tr := compFilter.SelectElement("time-range"); tr != nil {
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
		if nested := compFilter.SelectElement("comp-filter"); nested != nil {
			filter.SubFilter = &Filter{}
			return r.parseFilter(compFilter, filter.SubFilter)
		}
	}

	return nil
}

func (r *ReportRequest) parseCalendarMultiget(root *etree.Element) error {
	r.MultiGet = &CalendarMultiget{}

	// Parse prop element
	if prop := root.FindElement("D:prop"); prop != nil {
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
	if tr := root.SelectElement("time-range"); tr != nil {
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
	root := doc.CreateElement("")

	// Set appropriate root element and add namespaces
	switch {
	case r.Query != nil:
		root.Tag = "calendar-query"
		root.Space = "C"
		root.CreateAttr("xmlns:D", DAV)
		root.CreateAttr("xmlns:C", CalDAV)

		// Add prop element
		prop := root.CreateElement("prop")
		prop.Space = "D"
		for _, p := range r.Query.Props {
			if p == "calendar-data" {
				elem := prop.CreateElement(p)
				elem.Space = "C"
			} else {
				elem := prop.CreateElement(p)
				elem.Space = "D"
			}
		}

		// Add filter element
		filter := root.CreateElement("filter")
		filter.Space = "C"
		r.Query.Filter.toElement(filter)

	case r.MultiGet != nil:
		root.Tag = "calendar-multiget"
		root.Space = "C"
		root.CreateAttr("xmlns:D", DAV)
		root.CreateAttr("xmlns:C", CalDAV)

		// Add prop element
		prop := root.CreateElement("prop")
		prop.Space = "D"
		for _, p := range r.MultiGet.Props {
			if p == "calendar-data" {
				elem := prop.CreateElement(p)
				elem.Space = "C"
			} else {
				elem := prop.CreateElement(p)
				elem.Space = "D"
			}
		}

		// Add href elements
		for _, href := range r.MultiGet.Hrefs {
			hrefElem := root.CreateElement("href")
			hrefElem.Space = "D"
			hrefElem.SetText(href)
		}

	case r.FreeBusy != nil:
		root.Tag = "free-busy-query"
		root.Space = "C"
		root.CreateAttr("xmlns:C", CalDAV)

		// Add time-range element
		tr := root.CreateElement("time-range")
		tr.Space = "C"
		r.FreeBusy.TimeRange.toElement(tr)
	}

	return doc
}
