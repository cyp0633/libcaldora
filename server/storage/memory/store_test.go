package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/emersion/go-ical"
)

func TestStore_User(t *testing.T) {
	store := New()
	ctx := context.Background()

	// Test getting non-existent user
	_, err := store.GetUser(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error getting non-existent user")
	} else if err.(*storage.Error).Type != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Add a user manually for testing
	user := &storage.User{ID: "test123"}
	store.users["test123"] = user

	// Test getting existing user
	got, err := store.GetUser(ctx, "test123")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("got user ID %s, want %s", got.ID, user.ID)
	}
}

func TestStore_Calendar(t *testing.T) {
	store := New()
	ctx := context.Background()

	cal := &storage.Calendar{
		ID:          "cal123",
		UserID:      "user123",
		Name:        "Test Calendar",
		Description: "Test Description",
		Color:       "#FF0000",
		TimeZone:    "UTC",
		Components:  []string{"VEVENT"},
		Calendar:    ical.NewCalendar(),
	}

	// Test creating calendar
	if err := store.CreateCalendar(ctx, cal); err != nil {
		t.Errorf("unexpected error creating calendar: %v", err)
	}

	// Test creating duplicate calendar
	if err := store.CreateCalendar(ctx, cal); err == nil {
		t.Error("expected error creating duplicate calendar")
	} else if err.(*storage.Error).Type != storage.ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// Test getting calendar
	got, err := store.GetCalendar(ctx, "user123", "cal123")
	if err != nil {
		t.Errorf("unexpected error getting calendar: %v", err)
	}
	if got.ID != cal.ID || got.Name != cal.Name {
		t.Errorf("got calendar %+v, want %+v", got, cal)
	}

	// Test listing calendars
	cals, err := store.ListCalendars(ctx, "user123")
	if err != nil {
		t.Errorf("unexpected error listing calendars: %v", err)
	}
	if len(cals) != 1 {
		t.Errorf("got %d calendars, want 1", len(cals))
	}

	// Test updating calendar
	cal.Name = "Updated Calendar"
	if err := store.UpdateCalendar(ctx, cal); err != nil {
		t.Errorf("unexpected error updating calendar: %v", err)
	}
	got, _ = store.GetCalendar(ctx, "user123", "cal123")
	if got.Name != "Updated Calendar" {
		t.Errorf("got calendar name %s, want Updated Calendar", got.Name)
	}

	// Test deleting calendar
	if err := store.DeleteCalendar(ctx, "user123", "cal123"); err != nil {
		t.Errorf("unexpected error deleting calendar: %v", err)
	}
	if _, err := store.GetCalendar(ctx, "user123", "cal123"); err == nil {
		t.Error("expected error getting deleted calendar")
	}
}

func TestStore_CalendarObject(t *testing.T) {
	store := New()
	ctx := context.Background()

	// Create a calendar first
	cal := &storage.Calendar{
		ID:         "cal123",
		UserID:     "user123",
		Calendar:   ical.NewCalendar(),
		Components: []string{"VEVENT"},
	}
	store.CreateCalendar(ctx, cal)

	// Create an event
	event := ical.NewEvent()
	event.Props.SetText("SUMMARY", "Test Event")
	// Set event time within the test time range
	now := time.Now()
	event.Props.SetDateTime("DTSTART", now)
	event.Props.SetDateTime("DTEND", now.Add(30*time.Minute))

	obj := &storage.CalendarObject{
		ID:         "evt123",
		CalendarID: "cal123",
		UserID:     "user123",
		ObjectType: "VEVENT",
		Event:      event,
	}

	// Test creating object
	if err := store.CreateObject(ctx, obj); err != nil {
		t.Errorf("unexpected error creating object: %v", err)
	}

	// Test creating object in non-existent calendar
	badObj := &storage.CalendarObject{
		ID:         "evt456",
		CalendarID: "nonexistent",
		UserID:     "user123",
		Event:      event,
	}
	if err := store.CreateObject(ctx, badObj); err == nil {
		t.Error("expected error creating object in non-existent calendar")
	}

	// Test getting object
	got, err := store.GetObject(ctx, "user123", "evt123")
	if err != nil {
		t.Errorf("unexpected error getting object: %v", err)
	}
	if got.ID != obj.ID {
		t.Errorf("got object ID %s, want %s", got.ID, obj.ID)
	}

	// Test listing objects
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)
	opts := &storage.ListOptions{
		Start:          &start,
		End:            &end,
		ComponentTypes: []string{"VEVENT"},
	}

	objects, err := store.ListObjects(ctx, "user123", "cal123", opts)
	if err != nil {
		t.Errorf("unexpected error listing objects: %v", err)
	}
	if len(objects) != 1 {
		t.Errorf("got %d objects, want 1", len(objects))
	}

	// Test updating object
	event.Props.SetText("SUMMARY", "Updated Event")
	obj.Event = event
	if err := store.UpdateObject(ctx, obj); err != nil {
		t.Errorf("unexpected error updating object: %v", err)
	}
	got, _ = store.GetObject(ctx, "user123", "evt123")
	summary, _ := got.Event.Props.Text("SUMMARY")
	if summary != "Updated Event" {
		t.Errorf("got event summary %s, want Updated Event", summary)
	}

	// Test deleting object
	if err := store.DeleteObject(ctx, "user123", "evt123"); err != nil {
		t.Errorf("unexpected error deleting object: %v", err)
	}
	if _, err := store.GetObject(ctx, "user123", "evt123"); err == nil {
		t.Error("expected error getting deleted object")
	}

	// Test objects are deleted when calendar is deleted
	obj2 := &storage.CalendarObject{
		ID:         "evt789",
		CalendarID: "cal123",
		UserID:     "user123",
		Event:      event,
	}
	store.CreateObject(ctx, obj2)
	store.DeleteCalendar(ctx, "user123", "cal123")
	if _, err := store.GetObject(ctx, "user123", "evt789"); err == nil {
		t.Error("expected error getting object from deleted calendar")
	}
}
