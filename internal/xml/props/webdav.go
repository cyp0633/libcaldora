package props

import (
	"strconv"
	"time"

	"github.com/beevik/etree"
	"github.com/cyp0633/libcaldora/server/storage"
)

type DisplayName struct {
	Value string
}

func (p DisplayName) Encode() *etree.Element {
	elem := createElement("displayname")
	elem.SetText(p.Value)
	return elem
}

func (p *DisplayName) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type Resourcetype struct {
	// Primary resource type from storage package
	Type ResourceType
	// Optional sub-type for calendar objects (vevent, vtodo, etc)
	ObjectType string
}

type ResourceType storage.ResourceType

const (
	ResourcePrincipal ResourceType = iota
	ResourceHomeSet
	ResourceCollection
	ResourceObject
)

func (p Resourcetype) Encode() *etree.Element {
	elem := createElement("resourcetype")

	// Handle the primary resource type based on storage.ResourceType
	switch p.Type {
	case ResourcePrincipal:
		// User Principal: <d:resourcetype><d:principal/></d:resourcetype>
		principalElem := createElement("principal")
		elem.AddChild(principalElem)

	case ResourceHomeSet:
		// Calendar Home Set: <d:resourcetype><d:collection/><cal:calendar-home-set/></d:resourcetype>
		collElem := createElement("collection")
		elem.AddChild(collElem)

		homeSetElem := createElementWithPrefix("calendar-home-set", "cal")
		elem.AddChild(homeSetElem)

	case ResourceCollection:
		// Calendar Collection: <d:resourcetype><d:collection/><cal:calendar/></d:resourcetype>
		collElem := createElement("collection")
		elem.AddChild(collElem)

		calElem := createElement("calendar")
		elem.AddChild(calElem)

	case ResourceObject:
		// Calendar Object: <d:resourcetype><d:vevent/></d:resourcetype> or other types
		if p.ObjectType != "" {
			// For specific object types like vevent, vtodo, etc.
			objTypeElem := createElement(p.ObjectType)
			elem.AddChild(objTypeElem)
		}

		// Handle freebusy case
		if p.ObjectType == "freebusy" {
			freebusy := createElement("freebusy")
			elem.AddChild(freebusy)
		}

		// Handle scheduling case
		if p.ObjectType == "schedule-interaction" {
			scheduleElem := createElement("schedule-interaction")
			elem.AddChild(scheduleElem)
		}
	}
	return elem
}

func (p *Resourcetype) Decode(elem *etree.Element) error {
	// Default to ResourceObject if not specified
	p.Type = ResourceObject
	p.ObjectType = ""

	// Check for principal
	if elem.FindElement("principal") != nil {
		p.Type = ResourcePrincipal
		return nil
	}

	// Check for collection
	collElem := elem.FindElement("collection")
	if collElem != nil {
		// Check for calendar home set
		if elem.FindElement("calendar-home-set") != nil {
			p.Type = ResourceHomeSet
			return nil
		}

		// Check for calendar collection
		if elem.FindElement("calendar") != nil {
			p.Type = ResourceCollection
			return nil
		}
	}

	// Handle object types (VEVENT, VTODO, etc.)
	for _, child := range elem.ChildElements() {
		if child.Tag == "vevent" || child.Tag == "vtodo" || child.Tag == "vjournal" ||
			child.Tag == "freebusy" || child.Tag == "schedule-interaction" {
			p.ObjectType = child.Tag
			break
		}
	}

	return nil
}

type GetEtag struct {
	Value string
}

func (p GetEtag) Encode() *etree.Element {
	elem := createElement("getetag")
	elem.SetText(p.Value)
	return elem
}

func (p *GetEtag) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type GetLastModified struct {
	Value time.Time
}

func (p GetLastModified) Encode() *etree.Element {
	elem := createElement("getlastmodified")
	// Format to RFC1123 format: "Wed, 05 Apr 2025 14:30:00 GMT"
	elem.SetText(p.Value.UTC().Format(time.RFC1123))
	return elem
}

func (p *GetLastModified) Decode(elem *etree.Element) error {
	t, err := time.Parse(time.RFC1123, elem.Text())
	if err != nil {
		// Try alternative formats if RFC1123 fails
		t, err = time.Parse(time.RFC3339, elem.Text())
		if err != nil {
			return err
		}
	}
	p.Value = t.UTC() // Convert to UTC to ensure consistent time zone
	return nil
}

type GetContentType struct {
	Value string
}

func (p GetContentType) Encode() *etree.Element {
	elem := createElement("getcontenttype")
	elem.SetText(p.Value)
	return elem
}

func (p *GetContentType) Decode(elem *etree.Element) error {
	p.Value = elem.Text()
	return nil
}

type Owner struct {
	Value string
}

func (p Owner) Encode() *etree.Element {
	elem := createElement("owner")
	href := createElement("href")
	href.SetText(p.Value)
	elem.AddChild(href)
	return elem
}

func (p *Owner) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Value = href.Text()
	}
	return nil
}

type CurrentUserPrincipal struct {
	Value string
}

func (p CurrentUserPrincipal) Encode() *etree.Element {
	elem := createElement("current-user-principal")
	href := createElement("href")
	href.SetText(p.Value)
	elem.AddChild(href)
	return elem
}

func (p *CurrentUserPrincipal) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Value = href.Text()
	}
	return nil
}

type PrincipalURL struct {
	Value string
}

func (p PrincipalURL) Encode() *etree.Element {
	elem := createElement("principal-url")
	hrefElem := createElement("href")
	elem.AddChild(hrefElem)
	hrefElem.SetText(p.Value)
	return elem
}

func (p *PrincipalURL) Decode(elem *etree.Element) error {
	href := elem.FindElement("href")
	if href != nil {
		p.Value = href.Text()
	}
	return nil
}

type SupportedReportSet struct {
	Reports []ReportType
}

type ReportType int

const (
	ReportTypePropfind ReportType = iota
	ReportTypeCalendarQuery
	ReportTypeCalendarMultiget
	ReportTypeFreebusyQuery
	ReportTypeScheduleQuery
	ReportTypeScheduleMultiget
	ReportTypeSearch
)

func (p SupportedReportSet) Encode() *etree.Element {
	elem := createElement("supported-report-set")

	for _, report := range p.Reports {
		// Create the supported-report element
		supportedReportElem := createElement("supported-report")
		elem.AddChild(supportedReportElem)

		// Create the report element
		reportElem := createElement("report")
		supportedReportElem.AddChild(reportElem)

		// Add the specific report type
		var reportTypeElem *etree.Element
		switch report {
		case ReportTypePropfind:
			reportTypeElem = createElement("propfind")
		case ReportTypeCalendarQuery:
			reportTypeElem = createElement("calendar-query")
		case ReportTypeCalendarMultiget:
			reportTypeElem = createElement("calendar-multiget")
		case ReportTypeFreebusyQuery:
			reportTypeElem = createElement("free-busy-query")
		case ReportTypeScheduleQuery:
			reportTypeElem = createElement("schedule-query")
		case ReportTypeScheduleMultiget:
			reportTypeElem = createElement("schedule-multiget")
		case ReportTypeSearch:
			reportTypeElem = createElement("search")
		}

		if reportTypeElem != nil {
			reportElem.AddChild(reportTypeElem)
		}
	}

	return elem
}

func (p *SupportedReportSet) Decode(elem *etree.Element) error {
	p.Reports = []ReportType{}

	// Find all supported-report elements
	supportedReports := elem.FindElements("supported-report")
	for _, sr := range supportedReports {
		reportElem := sr.FindElement("report")
		if reportElem == nil {
			continue
		}

		// Check which report type is present
		if reportElem.FindElement("propfind") != nil {
			p.Reports = append(p.Reports, ReportTypePropfind)
		}
		if reportElem.FindElement("calendar-query") != nil {
			p.Reports = append(p.Reports, ReportTypeCalendarQuery)
		}
		if reportElem.FindElement("calendar-multiget") != nil {
			p.Reports = append(p.Reports, ReportTypeCalendarMultiget)
		}
		if reportElem.FindElement("free-busy-query") != nil {
			p.Reports = append(p.Reports, ReportTypeFreebusyQuery)
		}
		if reportElem.FindElement("schedule-query") != nil {
			p.Reports = append(p.Reports, ReportTypeScheduleQuery)
		}
		if reportElem.FindElement("schedule-multiget") != nil {
			p.Reports = append(p.Reports, ReportTypeScheduleMultiget)
		}
		if reportElem.FindElement("search") != nil {
			p.Reports = append(p.Reports, ReportTypeSearch)
		}
	}

	return nil
}

type ACE struct {
	Principal string
	Grant     []string
	Deny      []string
}

type ACL struct {
	Aces []ACE
}

func (p ACL) Encode() *etree.Element {
	elem := createElement("acl")

	for _, aceEntry := range p.Aces {
		aceElem := createElement("ace")
		elem.AddChild(aceElem)

		// Principal
		principalElem := createElement("principal")
		aceElem.AddChild(principalElem)

		hrefElem := createElement("href")
		principalElem.AddChild(hrefElem)
		hrefElem.SetText(aceEntry.Principal)

		// Grant privileges
		if len(aceEntry.Grant) > 0 {
			grantElem := createElement("grant")
			aceElem.AddChild(grantElem)

			for _, privilege := range aceEntry.Grant {
				privElem := createElement("privilege")
				grantElem.AddChild(privElem)

				privTypeElem := createElement(privilege)
				privElem.AddChild(privTypeElem)
			}
		}

		// Deny privileges
		if len(aceEntry.Deny) > 0 {
			denyElem := createElement("deny")
			aceElem.AddChild(denyElem)

			for _, privilege := range aceEntry.Deny {
				privElem := createElement("privilege")
				denyElem.AddChild(privElem)

				privTypeElem := createElement(privilege)
				privElem.AddChild(privTypeElem)
			}
		}
	}

	return elem
}

func (p *ACL) Decode(elem *etree.Element) error {
	p.Aces = []ACE{}

	aceElems := elem.FindElements("ace")
	for _, aceElem := range aceElems {
		ace := ACE{}

		// Get principal
		principalElem := aceElem.FindElement("principal")
		if principalElem != nil {
			hrefElem := principalElem.FindElement("href")
			if hrefElem != nil {
				ace.Principal = hrefElem.Text()
			}
		}

		// Get grant privileges
		grantElem := aceElem.FindElement("grant")
		if grantElem != nil {
			ace.Grant = []string{}
			privElems := grantElem.FindElements("privilege")
			for _, privElem := range privElems {
				for _, child := range privElem.ChildElements() {
					ace.Grant = append(ace.Grant, child.Tag)
				}
			}
		}

		// Get deny privileges
		denyElem := aceElem.FindElement("deny")
		if denyElem != nil {
			ace.Deny = []string{}
			privElems := denyElem.FindElements("privilege")
			for _, privElem := range privElems {
				for _, child := range privElem.ChildElements() {
					ace.Deny = append(ace.Deny, child.Tag)
				}
			}
		}

		p.Aces = append(p.Aces, ace)
	}

	return nil
}

type CurrentUserPrivilegeSet struct {
	Privileges []string
}

func (p CurrentUserPrivilegeSet) Encode() *etree.Element {
	elem := createElement("current-user-privilege-set")

	for _, privilege := range p.Privileges {
		privElem := createElement("privilege")
		elem.AddChild(privElem)

		privTypeElem := createElement(privilege)
		privElem.AddChild(privTypeElem)
	}

	return elem
}

func (p *CurrentUserPrivilegeSet) Decode(elem *etree.Element) error {
	p.Privileges = []string{}

	privElems := elem.FindElements("privilege")
	for _, privElem := range privElems {
		for _, child := range privElem.ChildElements() {
			p.Privileges = append(p.Privileges, child.Tag)
		}
	}

	return nil
}

type QuotaAvailableBytes struct {
	Value int64
}

func (p QuotaAvailableBytes) Encode() *etree.Element {
	elem := createElement("quota-available-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

func (p *QuotaAvailableBytes) Decode(elem *etree.Element) error {
	val, err := strconv.ParseInt(elem.Text(), 10, 64)
	if err != nil {
		return err
	}
	p.Value = val
	return nil
}

type QuotaUsedBytes struct {
	Value int64
}

func (p QuotaUsedBytes) Encode() *etree.Element {
	elem := createElement("quota-used-bytes")
	elem.SetText(strconv.FormatInt(p.Value, 10))
	return elem
}

func (p *QuotaUsedBytes) Decode(elem *etree.Element) error {
	val, err := strconv.ParseInt(elem.Text(), 10, 64)
	if err != nil {
		return err
	}
	p.Value = val
	return nil
}
