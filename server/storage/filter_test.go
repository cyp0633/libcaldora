package storage

import (
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
)

// Helper functions to create test calendar objects
func createTestEvent(uid, summary string, start, end time.Time) *CalendarObject {
	comp := ical.NewComponent(ical.CompEvent)
	comp.Props.SetText(ical.PropUID, uid)
	comp.Props.SetText(ical.PropSummary, summary)
	comp.Props.SetDateTime(ical.PropDateTimeStart, start)
	comp.Props.SetDateTime(ical.PropDateTimeEnd, end)
	return &CalendarObject{
		Component: []*ical.Component{comp},
	}
}

func createTestTodo(uid, summary string, due time.Time, status string) *CalendarObject {
	comp := ical.NewComponent(ical.CompToDo)
	comp.Props.SetText(ical.PropUID, uid)
	comp.Props.SetText(ical.PropSummary, summary)
	comp.Props.SetDateTime(ical.PropDue, due)
	if status != "" {
		comp.Props.SetText(ical.PropStatus, status)
	}
	return &CalendarObject{
		Component: []*ical.Component{comp},
	}
}

func createCalendarWithEvents(events ...*CalendarObject) *CalendarObject {
	calendar := ical.NewComponent(ical.CompCalendar)
	calendar.Props.SetText(ical.PropProductID, "-//libcaldora//NONSGML v1.0//EN")
	calendar.Props.SetText(ical.PropVersion, "2.0")

	// Add events as children
	for _, event := range events {
		if len(event.Component) > 0 && event.Component[0] != nil {
			calendar.Children = append(calendar.Children, event.Component[0])
		}
	}

	return &CalendarObject{
		Component: []*ical.Component{calendar},
	}
}

func createEventWithAttendee(uid, summary string, start, end time.Time, attendees map[string]map[string]string) *CalendarObject {
	comp := ical.NewComponent(ical.CompEvent)
	comp.Props.SetText(ical.PropUID, uid)
	comp.Props.SetText(ical.PropSummary, summary)
	comp.Props.SetDateTime(ical.PropDateTimeStart, start)
	comp.Props.SetDateTime(ical.PropDateTimeEnd, end)

	// Add attendees with parameters
	for attendee, params := range attendees {
		prop := ical.NewProp(ical.PropAttendee)
		prop.Value = attendee
		for k, v := range params {
			prop.Params.Set(k, v)
		}
		comp.Props.Add(prop)
	}

	return &CalendarObject{
		Component: []*ical.Component{comp},
	}
}

// Test basic component name filtering
func TestFilter_ValidateComponentName(t *testing.T) {
	now := time.Now()
	event := createTestEvent("123", "Test Event", now, now.Add(1*time.Hour))
	todo := createTestTodo("456", "Test Todo", now.Add(2*time.Hour), "NEEDS-ACTION")

	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Matching component name - VEVENT",
			filter: Filter{
				Component: ical.CompEvent,
			},
			obj:  event,
			want: true,
		},
		{
			name: "Non-matching component name",
			filter: Filter{
				Component: ical.CompEvent,
			},
			obj:  todo,
			want: false,
		},
		{
			name: "Nil component",
			filter: Filter{
				Component: ical.CompEvent,
			},
			obj:  nil,
			want: false,
		},
		{
			name: "Nil component with IsNotDefined",
			filter: Filter{
				Component:    ical.CompEvent,
				IsNotDefined: true,
			},
			obj:  nil,
			want: true,
		},
		{
			name: "Is-not-defined true for non-matching component",
			filter: Filter{
				Component:    ical.CompJournal,
				IsNotDefined: true,
			},
			obj:  event,
			want: true,
		},
		{
			name: "Is-not-defined false for matching component",
			filter: Filter{
				Component:    ical.CompEvent,
				IsNotDefined: true,
			},
			obj:  event,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test time range filtering
func TestFilter_ValidateTimeRange(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	// Create events at different times
	event1 := createTestEvent("1", "Event 1",
		baseTime, baseTime.Add(1*time.Hour)) // 10:00-11:00
	event2 := createTestEvent("2", "Event 2",
		baseTime.Add(2*time.Hour), baseTime.Add(4*time.Hour)) // 12:00-14:00
	event3 := createTestEvent("3", "Event 3",
		baseTime.Add(5*time.Hour), baseTime.Add(7*time.Hour)) // 15:00-17:00

	// Create todos with due dates
	todo1 := createTestTodo("4", "Todo 1", baseTime.Add(3*time.Hour), "NEEDS-ACTION") // Due at 13:00

	// Create all-day event (midnight times)
	allDayStart := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	allDayEvent := createTestEvent("5", "All Day", allDayStart, allDayStart) // All day on Jan 2

	rangeStart := baseTime.Add(30 * time.Minute) // 10:30
	rangeEnd := baseTime.Add(5 * time.Hour)      // 15:00

	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Event fully within time range",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &rangeStart,
					End:   &rangeEnd,
				},
			},
			obj:  event2,
			want: true,
		},
		{
			name: "Event partially overlaps time range (start before, end within)",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &rangeStart,
					End:   &rangeEnd,
				},
			},
			obj:  event1,
			want: true,
		},
		{
			name: "Event partially overlaps time range (start within, end after)",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &rangeStart,
					End:   &rangeEnd,
				},
			},
			obj:  event3,
			want: true,
		},
		{
			name: "Todo with due date within time range",
			filter: Filter{
				Component: ical.CompToDo,
				TimeRange: &TimeRange{
					Start: &rangeStart,
					End:   &rangeEnd,
				},
			},
			obj:  todo1,
			want: true,
		},
		{
			name: "Time range with only start time",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &rangeStart,
				},
			},
			obj:  event3, // 15:00-17:00
			want: true,
		},
		{
			name: "Time range with only end time",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					End: &rangeEnd, // 15:00
				},
			},
			obj:  event1, // 10:00-11:00
			want: true,
		},
		{
			name: "Event outside time range",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &baseTime,
					End:   &rangeStart, // 10:30
				},
			},
			obj:  event2, // 12:00-14:00
			want: false,
		},
		{
			name: "All-day event with time range",
			filter: Filter{
				Component: ical.CompEvent,
				TimeRange: &TimeRange{
					Start: &allDayStart,
					End:   &rangeEnd,
				},
			},
			obj:  allDayEvent,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test property filtering
func TestFilter_ValidatePropertyFilters(t *testing.T) {
	now := time.Now()

	// Create event with specific properties
	event := createTestEvent("123", "Business Meeting", now, now.Add(1*time.Hour))
	event.Component[0].Props.SetText(ical.PropLocation, "Conference Room")
	event.Component[0].Props.SetText(ical.PropDescription, "Quarterly review meeting")

	// Create todo with different properties
	todo := createTestTodo("456", "Prepare Report", now.Add(24*time.Hour), "NEEDS-ACTION")
	todo.Component[0].Props.SetText(ical.PropDescription, "Prepare quarterly financial report")

	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Property exists",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropLocation},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Property doesn't exist",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropCategories},
				},
			},
			obj:  event,
			want: false,
		},
		{
			name: "Property is-not-defined true",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropCategories, IsNotDefined: true},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Property is-not-defined false",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropLocation, IsNotDefined: true},
				},
			},
			obj:  event,
			want: false,
		},
		{
			name: "Property text match - equals",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropLocation,
						TextMatch: &TextMatch{
							MatchType: "equals",
							Value:     "Conference Room",
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Property text match - contains",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropDescription,
						TextMatch: &TextMatch{
							MatchType: "contains",
							Value:     "review",
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Property text match with negation",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropDescription,
						TextMatch: &TextMatch{
							MatchType: "contains",
							Value:     "cancel",
							Negate:    true,
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Multiple property filters with anyof",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropCategories},
					{Name: ical.PropLocation},
				},
				Test: "anyof",
			},
			obj:  event,
			want: true,
		},
		{
			name: "Multiple property filters with allof - one missing",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropCategories},
					{Name: ical.PropLocation},
				},
				Test: "allof",
			},
			obj:  event,
			want: false,
		},
		{
			name: "Multiple property filters with allof - all present",
			filter: Filter{
				PropFilters: []PropFilter{
					{Name: ical.PropLocation},
					{Name: ical.PropSummary},
				},
				Test: "allof",
			},
			obj:  event,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test parameter filtering
func TestFilter_ValidateParameterFilters(t *testing.T) {
	now := time.Now()

	// Create event with attendees having different parameters
	attendees := map[string]map[string]string{
		"mailto:alice@example.com": {
			"PARTSTAT": "ACCEPTED",
			"ROLE":     "REQ-PARTICIPANT",
			"CN":       "Alice Smith",
		},
		"mailto:bob@example.com": {
			"PARTSTAT": "DECLINED",
			"ROLE":     "OPT-PARTICIPANT",
			"CN":       "Bob Jones",
		},
		"mailto:carol@example.com": {
			"PARTSTAT": "NEEDS-ACTION",
			"RSVP":     "TRUE",
		},
	}

	event := createEventWithAttendee("123", "Team Meeting", now, now.Add(1*time.Hour), attendees)

	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Parameter exists on property",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{Name: "PARTSTAT"},
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Parameter doesn't exist on property",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{Name: "LANGUAGE"},
						},
					},
				},
			},
			obj:  event,
			want: false,
		},
		{
			name: "Parameter is-not-defined true",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{Name: "LANGUAGE", IsNotDefined: true},
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Parameter text match",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{
								Name: "PARTSTAT",
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "ACCEPTED",
								},
							},
						},
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Multiple parameter filters with anyof",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{
								Name: "PARTSTAT",
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "MAYBE", // Doesn't exist
								},
							},
							{
								Name: "ROLE",
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "REQ-PARTICIPANT", // Does exist
								},
							},
						},
						Test: "anyof",
					},
				},
			},
			obj:  event,
			want: true,
		},
		{
			name: "Multiple parameter filters with allof - one missing",
			filter: Filter{
				PropFilters: []PropFilter{
					{
						Name: ical.PropAttendee,
						ParamFilters: []ParamFilter{
							{
								Name: "PARTSTAT",
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "ACCEPTED",
								},
							},
							{
								Name: "LANGUAGE", // Doesn't exist
							},
						},
						Test: "allof",
					},
				},
			},
			obj:  event,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test text matching with all options
func TestFilter_ValidateTextMatch(t *testing.T) {
	// Create properties with different texts to test against
	value := "This is a Sample TEXT for testing"

	tests := []struct {
		name      string
		textMatch TextMatch
		want      bool
	}{
		{
			name: "equals - exact match",
			textMatch: TextMatch{
				MatchType: "equals",
				Value:     value,
			},
			want: true,
		},
		{
			name: "equals - case mismatch without case-insensitive",
			textMatch: TextMatch{
				MatchType: "equals",
				Value:     "this is a sample text for testing",
			},
			want: false,
		},
		{
			name: "equals - case mismatch with case-insensitive",
			textMatch: TextMatch{
				MatchType: "equals",
				Value:     "this is a sample text for testing",
				Collation: "i;unicode-casemap",
			},
			want: true,
		},
		{
			name: "contains - substring match",
			textMatch: TextMatch{
				MatchType: "contains",
				Value:     "Sample TEXT",
			},
			want: true,
		},
		{
			name: "contains - case-insensitive",
			textMatch: TextMatch{
				MatchType: "contains",
				Value:     "sample text",
				Collation: "i;unicode-casemap",
			},
			want: true,
		},
		{
			name: "starts-with - positive match",
			textMatch: TextMatch{
				MatchType: "starts-with",
				Value:     "This is",
			},
			want: true,
		},
		{
			name: "starts-with - negative match",
			textMatch: TextMatch{
				MatchType: "starts-with",
				Value:     "Sample",
			},
			want: false,
		},
		{
			name: "ends-with - positive match",
			textMatch: TextMatch{
				MatchType: "ends-with",
				Value:     "for testing",
			},
			want: true,
		},
		{
			name: "ends-with - case-insensitive match",
			textMatch: TextMatch{
				MatchType: "ends-with",
				Value:     "FOR TESTING",
				Collation: "i;unicode-casemap",
			},
			want: true,
		},
		{
			name: "negated match - contains",
			textMatch: TextMatch{
				MatchType: "contains",
				Value:     "nonexistent",
				Negate:    true,
			},
			want: true,
		},
		{
			name: "negated match - equals",
			textMatch: TextMatch{
				MatchType: "equals",
				Value:     value,
				Negate:    true,
			},
			want: false,
		},
		{
			name: "default match type (contains)",
			textMatch: TextMatch{
				Value: "Sample",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateTextMatch(value, &tt.textMatch)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test nested component filtering
func TestFilter_ValidateNestedComponents(t *testing.T) {
	now := time.Now()

	// Create event1
	event1 := createTestEvent("123", "Main Event", now, now.Add(1*time.Hour))

	// Create event2 with alarm component
	event2 := createTestEvent("456", "Event with Alarm", now.Add(2*time.Hour), now.Add(3*time.Hour))
	alarm := ical.NewComponent(ical.CompAlarm)
	alarm.Props.SetText(ical.PropAction, "DISPLAY")
	alarm.Props.SetText(ical.PropDescription, "Reminder")
	event2.Component[0].Children = append(event2.Component[0].Children, alarm)

	// Create a calendar with both events as children
	calendar := createCalendarWithEvents(event1, event2)

	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Calendar with VEVENT children",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{Component: ical.CompEvent},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "VEVENT with VALARM child",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						Children: []Filter{
							{Component: ical.CompAlarm},
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "VEVENT with specific summary and VALARM child",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropSummary,
								TextMatch: &TextMatch{
									MatchType: "contains",
									Value:     "with Alarm",
								},
							},
						},
						Children: []Filter{
							{Component: ical.CompAlarm},
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "VEVENT with nonexistent child component",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						Children: []Filter{
							{Component: ical.CompJournal},
						},
					},
				},
			},
			obj:  calendar,
			want: false,
		},
		{
			name: "Multiple nested filters with anyof",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name:         ical.PropLocation,
								IsNotDefined: false,
							},
						},
					},
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropSummary,
								TextMatch: &TextMatch{
									Value: "Main Event",
								},
							},
						},
					},
				},
				Test: "anyof",
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "Multiple nested filters with allof - one fails",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropLocation, // Doesn't exist on either event
							},
						},
					},
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropSummary,
							},
						},
					},
				},
				Test: "allof",
			},
			obj:  calendar,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test complex real-world filter combinations
func TestFilter_ComplexFilters(t *testing.T) {
	// Create a time base for our events
	baseTime := time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)

	// Create events spanning different time periods with various properties
	// Work meeting with two attendees
	workMeeting := createEventWithAttendee(
		"work-123",
		"Team Planning",
		baseTime,
		baseTime.Add(2*time.Hour),
		map[string]map[string]string{
			"mailto:manager@example.com": {
				"PARTSTAT": "ACCEPTED",
				"ROLE":     "CHAIR",
				"CN":       "Team Manager",
			},
			"mailto:employee@example.com": {
				"PARTSTAT": "ACCEPTED",
				"ROLE":     "REQ-PARTICIPANT",
			},
		},
	)
	workMeeting.Component[0].Props.SetText(ical.PropLocation, "Conference Room A")
	workMeeting.Component[0].Props.SetText(ical.PropDescription, "Quarterly planning session")
	workMeeting.Component[0].Props.SetText(ical.PropCategories, "WORK,MEETING,PLANNING")

	// Personal appointment
	personalAppt := createTestEvent(
		"personal-456",
		"Dentist Appointment",
		baseTime.Add(1*24*time.Hour), // Next day
		baseTime.Add(1*24*time.Hour+1*time.Hour),
	)
	personalAppt.Component[0].Props.SetText(ical.PropLocation, "Dental Clinic")
	personalAppt.Component[0].Props.SetText(ical.PropDescription, "Regular check-up")
	personalAppt.Component[0].Props.SetText(ical.PropCategories, "PERSONAL,HEALTH")

	// TODO item
	workTodo := createTestTodo(
		"todo-789",
		"Prepare presentation",
		baseTime.Add(3*24*time.Hour), // Three days later
		"IN-PROCESS",
	)
	workTodo.Component[0].Props.SetText(ical.PropDescription, "Slides for the team meeting")
	workTodo.Component[0].Props.SetText(ical.PropCategories, "WORK,PRESENTATION")

	// Add them to a calendar
	calendar := createCalendarWithEvents(workMeeting, personalAppt, workTodo)

	// Create some complex filters
	tests := []struct {
		name   string
		filter Filter
		obj    *CalendarObject
		want   bool
	}{
		{
			name: "Find work-related events with specific attendee participation",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropCategories,
								TextMatch: &TextMatch{
									MatchType: "contains",
									Value:     "WORK",
								},
							},
							{
								Name: ical.PropAttendee,
								ParamFilters: []ParamFilter{
									{
										Name: "ROLE",
										TextMatch: &TextMatch{
											MatchType: "equals",
											Value:     "CHAIR",
										},
									},
								},
							},
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "Find personal events in time range",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropCategories,
								TextMatch: &TextMatch{
									MatchType: "contains",
									Value:     "PERSONAL",
								},
							},
						},
						TimeRange: &TimeRange{
							Start: &baseTime,                                         // From base time
							End:   &[]time.Time{baseTime.Add(2 * 24 * time.Hour)}[0], // To two days later
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "Find todos that are not completed",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompToDo,
						PropFilters: []PropFilter{
							{
								Name: ical.PropStatus,
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "COMPLETED",
									Negate:    true,
								},
							},
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "Find meetings in Conference Room A with manager participation",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							{
								Name: ical.PropSummary,
								TextMatch: &TextMatch{
									MatchType: "contains",
									Value:     "Planning",
									Collation: "i;unicode-casemap", // Case-insensitive
								},
							},
							{
								Name: ical.PropLocation,
								TextMatch: &TextMatch{
									MatchType: "equals",
									Value:     "Conference Room A",
								},
							},
							{
								Name: ical.PropAttendee,
								TextMatch: &TextMatch{
									MatchType: "contains",
									Value:     "manager@example.com",
								},
							},
						},
					},
				},
			},
			obj:  calendar,
			want: true,
		},
		{
			name: "Complex filter that shouldn't match anything",
			filter: Filter{
				Component: ical.CompCalendar,
				Children: []Filter{
					{
						Component: ical.CompEvent,
						PropFilters: []PropFilter{
							// Contradictory filters
							{
								Name: ical.PropCategories,
								TextMatch: &TextMatch{
									Value: "WORK",
								},
							},
							{
								Name: ical.PropCategories,
								TextMatch: &TextMatch{
									Value: "PERSONAL",
								},
							},
						},
						Test: "allof", // Requires all filters to match (impossible for our test data)
					},
				},
			},
			obj:  calendar,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Validate(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}
