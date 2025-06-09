package recurrence

import (
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_HasOccurrenceInRange(t *testing.T) {
	engine := NewEngine()

	// Base event: Daily meeting from 9-10 AM starting Jan 1, 2024
	masterStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		recurrence RecurrenceInfo
		rangeStart time.Time
		rangeEnd   time.Time
		expected   bool
	}{
		{
			name: "Non-recurring event in range",
			recurrence: RecurrenceInfo{
				RRULE: "",
			},
			rangeStart: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			rangeEnd:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			expected:   true,
		},
		{
			name: "Non-recurring event out of range",
			recurrence: RecurrenceInfo{
				RRULE: "",
			},
			rangeStart: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			rangeEnd:   time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			expected:   false,
		},
		{
			name: "Daily recurring event with occurrence in range",
			recurrence: RecurrenceInfo{
				RRULE: "FREQ=DAILY;COUNT=7",
			},
			rangeStart: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			rangeEnd:   time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			expected:   true,
		},
		{
			name: "Daily recurring event with no occurrence in range",
			recurrence: RecurrenceInfo{
				RRULE: "FREQ=DAILY;COUNT=3",
			},
			rangeStart: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			rangeEnd:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.HasOccurrenceInRange(
				masterStart, masterEnd,
				tt.recurrence,
				tt.rangeStart, tt.rangeEnd,
			)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRecurrenceInfoFromComponent(t *testing.T) {
	// This is tested implicitly through the integration
	// For now, we'll just ensure the function doesn't crash
	comp := &ical.Component{
		Name:  "VEVENT",
		Props: make(ical.Props),
	}

	info := ExtractRecurrenceInfoFromComponent(comp)
	assert.Equal(t, "", info.RRULE)
	assert.Empty(t, info.RDATE)
	assert.Empty(t, info.EXDATE)
	assert.Nil(t, info.RecurrenceID)
}
