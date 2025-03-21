/*
Package server provides a CalDAV server implementation that can be integrated into Go applications.

# Basic Usage

The simplest way to use this package is with the provided in-memory storage:

	store := memory.New()
	srv, err := server.New(store, "/caldav")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/caldav/", srv)
	http.ListenAndServe(":8080", nil)

# URL Scheme

The server uses a fixed URL scheme:
  - /u/<userid> - User principal
  - /u/<userid>/cal - Calendar home
  - /u/<userid>/cal/<calendarid> - Calendar collection
  - /u/<userid>/evt/<objectid> - Calendar object (event, todo, etc.)

# Custom Storage Backend

To implement your own storage backend, implement the storage.Storage interface:

	type Storage interface {
		// User operations
		GetUser(ctx context.Context, userID string) (*User, error)

		// Calendar operations
		GetCalendar(ctx context.Context, userID, calendarID string) (*Calendar, error)
		ListCalendars(ctx context.Context, userID string) ([]*Calendar, error)
		CreateCalendar(ctx context.Context, cal *Calendar) error
		UpdateCalendar(ctx context.Context, cal *Calendar) error
		DeleteCalendar(ctx context.Context, userID, calendarID string) error

		// Calendar object operations
		GetObject(ctx context.Context, userID, objectID string) (*CalendarObject, error)
		ListObjects(ctx context.Context, userID, calendarID string, opts *ListOptions) ([]*CalendarObject, error)
		CreateObject(ctx context.Context, obj *CalendarObject) error
		UpdateObject(ctx context.Context, obj *CalendarObject) error
		DeleteObject(ctx context.Context, userID, objectID string) error
	}

Example implementation for a SQL database:

	type SQLStorage struct {
		db *sql.DB
	}

	func (s *SQLStorage) GetCalendar(ctx context.Context, userID, calendarID string) (*storage.Calendar, error) {
		cal := &storage.Calendar{}
		err := s.db.QueryRowContext(ctx,
			"SELECT id, user_id, name, description, color FROM calendars WHERE user_id = ? AND id = ?",
			userID, calendarID,
		).Scan(&cal.ID, &cal.UserID, &cal.Name, &cal.Description, &cal.Color)
		if err == sql.ErrNoRows {
			return nil, &storage.Error{Type: storage.ErrNotFound, Message: "calendar not found"}
		}
		if err != nil {
			return nil, err
		}
		return cal, nil
	}

	// ... implement other methods ...

# Error Handling

The storage package provides standard error types:

	type ErrorType string

	const (
		ErrNotFound      ErrorType = "not_found"
		ErrAlreadyExists ErrorType = "already_exists"
		ErrInvalidInput  ErrorType = "invalid_input"
	)

These errors help the server determine the appropriate HTTP status codes.

# Calendar Objects

Calendar objects (events, todos) use the go-ical package for iCalendar format handling:

	event := ical.NewEvent()
	event.Props.SetText("SUMMARY", "Meeting")
	event.Props.SetDateTime("DTSTART", startTime)
	event.Props.SetDateTime("DTEND", endTime)

	obj := &storage.CalendarObject{
		ID:         "evt123",
		CalendarID: "cal456",
		UserID:     "user789",
		ObjectType: "VEVENT",
		Event:      event,
	}

# Testing

The storage/memory package provides an in-memory implementation that's useful for testing:

	func TestMyCalDAVApp(t *testing.T) {
		store := memory.New()
		srv, _ := server.New(store, "/caldav")

		// Create test calendar
		cal := &storage.Calendar{
			ID:     "testcal",
			UserID: "testuser",
			Name:   "Test Calendar",
		}
		store.CreateCalendar(context.Background(), cal)

		// Run tests...
	}

See server/example/main.go for a complete example implementation.
*/
package server
