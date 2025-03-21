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

// PropertyFilter represents a property filter
type PropertyFilter struct {
	Name      string
	Test      string // "equals", "contains", etc.
	TextMatch string
}

// Filter represents a calendar query filter
type Filter struct {
	Test          string // "allof", "anyof"
	ComponentName string // "VEVENT", "VTODO", etc.
	TimeRange     *TimeRange
	PropFilters   []PropertyFilter
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
		r.Query = &CalendarQuery{}
		return r.parseCalendarQuery(root)
	case "calendar-multiget":
		r.MultiGet = &CalendarMultiget{}
		return r.parseCalendarMultiget(root)
	case "free-busy-query":
		r.FreeBusy = &FreeBusyQuery{}
		return r.parseFreeBusyQuery(root)
	default:
		return fmt.Errorf("unsupported report type: %s", root.Tag)
	}
}

func (r *ReportRequest) parseCalendarQuery(root *etree.Element) error {
	for _, child := range root.ChildElements() {
		switch child.Tag {
		case TagProp:
			for _, prop := range child.ChildElements() {
				r.Query.Props = append(r.Query.Props, prop.Tag)
			}
		case "filter":
			compFilter := child.SelectElement("comp-filter")
			if compFilter != nil {
				r.Query.Filter.ComponentName = compFilter.SelectAttrValue("name", "")
				r.Query.Filter.Test = compFilter.SelectAttrValue("test", "anyof")

				timeRange := compFilter.SelectElement("time-range")
				if timeRange != nil {
					r.Query.Filter.TimeRange = &TimeRange{}
					if start := timeRange.SelectAttrValue("start", ""); start != "" {
						t, _ := time.Parse("20060102T150405Z", start)
						r.Query.Filter.TimeRange.Start = &t
					}
					if end := timeRange.SelectAttrValue("end", ""); end != "" {
						t, _ := time.Parse("20060102T150405Z", end)
						r.Query.Filter.TimeRange.End = &t
					}
				}

				for _, propFilter := range compFilter.SelectElements("prop-filter") {
					pf := PropertyFilter{
						Name: propFilter.SelectAttrValue("name", ""),
						Test: propFilter.SelectAttrValue("test", "contains"),
					}
					if textMatch := propFilter.SelectElement("text-match"); textMatch != nil {
						pf.TextMatch = textMatch.Text()
					}
					r.Query.Filter.PropFilters = append(r.Query.Filter.PropFilters, pf)
				}
			}
		}
	}
	return nil
}

func (r *ReportRequest) parseCalendarMultiget(root *etree.Element) error {
	for _, child := range root.ChildElements() {
		switch child.Tag {
		case TagProp:
			for _, prop := range child.ChildElements() {
				r.MultiGet.Props = append(r.MultiGet.Props, prop.Tag)
			}
		case TagHref:
			r.MultiGet.Hrefs = append(r.MultiGet.Hrefs, child.Text())
		}
	}
	return nil
}

func (r *ReportRequest) parseFreeBusyQuery(root *etree.Element) error {
	timeRange := root.SelectElement("time-range")
	if timeRange == nil {
		return fmt.Errorf("missing time-range element")
	}

	start := timeRange.SelectAttrValue("start", "")
	end := timeRange.SelectAttrValue("end", "")

	if start != "" {
		t, err := time.Parse("20060102T150405Z", start)
		if err != nil {
			return fmt.Errorf("invalid start time: %v", err)
		}
		r.FreeBusy.TimeRange.Start = &t
	}

	if end != "" {
		t, err := time.Parse("20060102T150405Z", end)
		if err != nil {
			return fmt.Errorf("invalid end time: %v", err)
		}
		r.FreeBusy.TimeRange.End = &t
	}

	return nil
}
