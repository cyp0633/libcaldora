package davclient

import (
"encoding/xml"
"errors"
"testing"
"time"

"github.com/emersion/go-ical"
)

// mockClient implements the minimum functionality needed for testing
type mockClient struct {
mockExecuteCalendarQuery func(*calendarQuery) ([]ical.Event, error)
}

func (m *mockClient) executeCalendarQuery(query *calendarQuery) ([]ical.Event, error) {
return m.mockExecuteCalendarQuery(query)
}

// Helper function to create a test event
func createTestEvent(uid, summary string) ical.Event {
event := ical.NewEvent()
event.Props.SetText("UID", uid)
event.Props.SetText("SUMMARY", summary)
return *event
}

func TestObjectFilter_BuildCalendarQuery(t *testing.T) {
	// Setup base client
	client := &davClient{}

	tests := []struct {
		name     string
		setup    func(*objectFilter)
		validate func(*testing.T, *calendarQuery)
	}{
		{
			name: "basic filter",
			setup: func(f *objectFilter) {
				// No additional setup needed
			},
			validate: func(t *testing.T, q *calendarQuery) {
				if q.Filter.CompFilter.CompFilter.Name != "VEVENT" {
					t.Errorf("expected VEVENT comp-filter, got %s", q.Filter.CompFilter.CompFilter.Name)
				}
			},
		},
		{
			name: "time range filter",
			setup: func(f *objectFilter) {
				start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
				f.TimeRange(start, end)
			},
			validate: func(t *testing.T, q *calendarQuery) {
				tr := q.Filter.CompFilter.CompFilter.TimeRange
				if tr == nil {
					t.Fatal("expected time-range, got nil")
				}
				if tr.Start != "20240101T000000Z" {
					t.Errorf("expected start 20240101T000000Z, got %s", tr.Start)
				}
				if tr.End != "20241231T235959Z" {
					t.Errorf("expected end 20241231T235959Z, got %s", tr.End)
				}
			},
		},
		{
			name: "summary filter",
			setup: func(f *objectFilter) {
				f.Summary("Test Event")
			},
			validate: func(t *testing.T, q *calendarQuery) {
				props := q.Filter.CompFilter.CompFilter.PropFilters
				if len(props) != 1 {
					t.Fatalf("expected 1 prop-filter, got %d", len(props))
				}
				if props[0].Name != "SUMMARY" {
					t.Errorf("expected SUMMARY prop-filter, got %s", props[0].Name)
				}
				if props[0].TextMatch.Text != "Test Event" {
					t.Errorf("expected 'Test Event' text match, got %s", props[0].TextMatch.Text)
				}
			},
		},
		{
			name: "multiple property filters",
			setup: func(f *objectFilter) {
				f.Summary("Test Event").
					Location("Test Location").
					Description("Test Description").
					Status("CONFIRMED")
			},
			validate: func(t *testing.T, q *calendarQuery) {
				props := q.Filter.CompFilter.CompFilter.PropFilters
				expectedProps := map[string]string{
					"SUMMARY":     "Test Event",
					"LOCATION":    "Test Location",
					"DESCRIPTION": "Test Description",
					"STATUS":      "CONFIRMED",
				}
				if len(props) != len(expectedProps) {
					t.Fatalf("expected %d prop-filters, got %d", len(expectedProps), len(props))
				}
				for _, prop := range props {
					expected, ok := expectedProps[prop.Name]
					if !ok {
						t.Errorf("unexpected prop-filter: %s", prop.Name)
						continue
					}
					if prop.TextMatch.Text != expected {
						t.Errorf("expected %s text match to be '%s', got '%s'", prop.Name, expected, prop.TextMatch.Text)
					}
				}
			},
		},
		{
			name: "status negation filter",
			setup: func(f *objectFilter) {
				f.NotStatus("CANCELLED")
			},
			validate: func(t *testing.T, q *calendarQuery) {
				props := q.Filter.CompFilter.CompFilter.PropFilters
				if len(props) != 1 {
					t.Fatalf("expected 1 prop-filter, got %d", len(props))
				}
				if props[0].Name != "STATUS" {
					t.Errorf("expected STATUS prop-filter, got %s", props[0].Name)
				}
				if !props[0].TextMatch.NegateCondition {
					t.Error("expected negate-condition to be true")
				}
				if props[0].TextMatch.Text != "CANCELLED" {
					t.Errorf("expected 'CANCELLED' text match, got %s", props[0].TextMatch.Text)
				}
			},
		},
		{
			name: "priority filter",
			setup: func(f *objectFilter) {
				f.Priority(1)
			},
			validate: func(t *testing.T, q *calendarQuery) {
				props := q.Filter.CompFilter.CompFilter.PropFilters
				if len(props) != 1 {
					t.Fatalf("expected 1 prop-filter, got %d", len(props))
				}
				if props[0].Name != "PRIORITY" {
					t.Errorf("expected PRIORITY prop-filter, got %s", props[0].Name)
				}
				if props[0].TextMatch.Text != "1" {
					t.Errorf("expected '1' text match, got %s", props[0].TextMatch.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset filter for each test
			testFilter := &objectFilter{
				client:     client,
				objectType: "VEVENT",
			}

			// Apply test-specific setup
			tt.setup(testFilter)

			// Build query
			query, err := testFilter.buildCalendarQuery()
			if err != nil {
				t.Fatalf("buildCalendarQuery() error = %v", err)
			}

			// Run validation
			tt.validate(t, query)

			// Additional validation: try marshaling to XML
			_, err = xml.MarshalIndent(query, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal XML: %v", err)
			}
		})
	}
}

func TestObjectFilter_Chaining(t *testing.T) {
	client := &davClient{}
	filter := &objectFilter{client: client}

	// Test method chaining
	result := filter.
		TimeRange(time.Now(), time.Now().Add(24*time.Hour)).
		HasAlarm().
		ObjectType("VEVENT").
		Priority(1).
		Categories("Work").
		Status("CONFIRMED").
		Summary("Meeting").
		Description("Team sync").
		Location("Office").
		Organizer("john@example.com").
		Limit(10)

	if result == nil {
		t.Error("method chaining should return non-nil result")
	}

	f := result.(*objectFilter)

	// Verify all fields are set correctly
	if f.timeRange == nil {
		t.Error("TimeRange not set")
	}
	if !f.hasAlarm {
		t.Error("HasAlarm not set")
	}
	if f.objectType != "VEVENT" {
		t.Errorf("expected objectType VEVENT, got %s", f.objectType)
	}
	if *f.priority != 1 {
		t.Errorf("expected priority 1, got %d", *f.priority)
	}
	if len(f.categories) != 1 || f.categories[0] != "Work" {
		t.Error("Categories not set correctly")
	}
	if f.status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", f.status)
	}
	if f.summary != "Meeting" {
		t.Errorf("expected summary Meeting, got %s", f.summary)
	}
	if f.description != "Team sync" {
		t.Errorf("expected description Team sync, got %s", f.description)
	}
	if f.location != "Office" {
		t.Errorf("expected location Office, got %s", f.location)
	}
	if f.organizer != "john@example.com" {
		t.Errorf("expected organizer john@example.com, got %s", f.organizer)
	}
	if f.limit != 10 {
		t.Errorf("expected limit 10, got %d", f.limit)
	}
}

func TestObjectFilter_Do(t *testing.T) {
	tests := []struct {
		name          string
		filter        *objectFilter
		executeResult []ical.Event
		executeErr    error
		wantErr       bool
		wantEvents    int
	}{
		{
			name: "successful query",
			filter: &objectFilter{
				objectType: "VEVENT",
				limit:      2,
			},
executeResult: []ical.Event{
createTestEvent("1", "Event 1"),
createTestEvent("2", "Event 2"),
createTestEvent("3", "Event 3"),
},
			wantEvents: 2,
		},
		{
			name: "query with error",
			filter: &objectFilter{
				objectType: "VEVENT",
			},
			executeErr: errors.New("query failed"),
			wantErr:    true,
		},
		{
			name: "filter with error",
			filter: &objectFilter{
				err: errors.New("previous error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockClient{
				mockExecuteCalendarQuery: func(query *calendarQuery) ([]ical.Event, error) {
					if tt.executeErr != nil {
						return nil, tt.executeErr
					}
					return tt.executeResult, nil
				},
			}

			// Set mock client
			tt.filter.client = mockClient

			events, err := tt.filter.Do()

			if (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(events) != tt.wantEvents {
				t.Errorf("Do() got %d events, want %d", len(events), tt.wantEvents)
			}
		})
	}
}
