package xml

import "github.com/beevik/etree"

// ToXML converts a ReportRequest to an XML document
func (r *ReportRequest) ToXML() *etree.Document {
	doc := etree.NewDocument()
	var root *etree.Element

	switch {
	case r.Query != nil:
		root = doc.CreateElement("calendar-query")
		root.Space = CalDAV
		r.addQueryElements(root)
	case r.MultiGet != nil:
		root = doc.CreateElement("calendar-multiget")
		root.Space = CalDAV
		r.addMultigetElements(root)
	case r.FreeBusy != nil:
		root = doc.CreateElement("free-busy-query")
		root.Space = CalDAV
		r.addFreeBusyElements(root)
	}

	AddNamespaces(doc)
	return doc
}

func (r *ReportRequest) addQueryElements(root *etree.Element) {
	if r.Query == nil {
		return
	}

	if len(r.Query.Props) > 0 {
		prop := root.CreateElement(TagProp)
		for _, name := range r.Query.Props {
			elem := prop.CreateElement(name)
			if name == "calendar-data" {
				elem.Space = "C"
			}
		}
	}

	filter := root.CreateElement("filter")
	if r.Query.Filter.ComponentName != "" {
		compFilter := filter.CreateElement("comp-filter")
		compFilter.CreateAttr("name", r.Query.Filter.ComponentName)
		if r.Query.Filter.Test != "" {
			compFilter.CreateAttr("test", r.Query.Filter.Test)
		}

		if r.Query.Filter.TimeRange != nil {
			timeRange := compFilter.CreateElement("time-range")
			if r.Query.Filter.TimeRange.Start != nil {
				timeRange.CreateAttr("start", r.Query.Filter.TimeRange.Start.Format("20060102T150405Z"))
			}
			if r.Query.Filter.TimeRange.End != nil {
				timeRange.CreateAttr("end", r.Query.Filter.TimeRange.End.Format("20060102T150405Z"))
			}
		}

		for _, pf := range r.Query.Filter.PropFilters {
			propFilter := compFilter.CreateElement("prop-filter")
			propFilter.CreateAttr("name", pf.Name)
			if pf.Test != "" {
				propFilter.CreateAttr("test", pf.Test)
			}
			if pf.TextMatch != "" {
				textMatch := propFilter.CreateElement("text-match")
				textMatch.SetText(pf.TextMatch)
			}
		}
	}
}

func (r *ReportRequest) addMultigetElements(root *etree.Element) {
	if r.MultiGet == nil {
		return
	}

	if len(r.MultiGet.Props) > 0 {
		prop := root.CreateElement(TagProp)
		for _, name := range r.MultiGet.Props {
			elem := prop.CreateElement(name)
			if name == "calendar-data" {
				elem.Space = "C"
			}
		}
	}

	for _, href := range r.MultiGet.Hrefs {
		hrefElem := root.CreateElement(TagHref)
		hrefElem.SetText(href)
	}
}

func (r *ReportRequest) addFreeBusyElements(root *etree.Element) {
	if r.FreeBusy == nil {
		return
	}

	timeRange := root.CreateElement("time-range")
	if r.FreeBusy.TimeRange.Start != nil {
		timeRange.CreateAttr("start", r.FreeBusy.TimeRange.Start.Format("20060102T150405Z"))
	}
	if r.FreeBusy.TimeRange.End != nil {
		timeRange.CreateAttr("end", r.FreeBusy.TimeRange.End.Format("20060102T150405Z"))
	}
}
