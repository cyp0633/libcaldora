package server

import (
	"testing"

	"github.com/cyp0633/libcaldora/internal/xml/propfind"
	"github.com/cyp0633/libcaldora/internal/xml/props"
	"github.com/cyp0633/libcaldora/server/storage"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Use storage types for testing
type Calendar = storage.Calendar
type CalendarObject = storage.CalendarObject

// Mock URL Converter
type MockURLConverter struct {
	mock.Mock
}

func (m *MockURLConverter) EncodePath(r Resource) (string, error) {
	args := m.Called(r)
	return args.String(0), args.Error(1)
}

func (m *MockURLConverter) ParsePath(path string) (Resource, error) {
	args := m.Called(path)
	return args.Get(0).(Resource), args.Error(1)
}

func TestHandlePropfindHomeSet(t *testing.T) {
	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
	}

	resource := Resource{
		UserID:       "user1",
		ResourceType: storage.ResourceHomeSet,
	}

	ctx := &RequestContext{
		Resource: resource,
	}

	// Test case 1: Basic properties
	t.Run("Basic properties", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		// Create request map with some basic properties using mo.Ok instead of mo.None
		req := propfind.ResponseMap{
			"displayname":                      mo.Ok[props.PropertyEncoder](nil),
			"calendar-home-set":                mo.Ok[props.PropertyEncoder](nil),
			"supported-calendar-component-set": mo.Ok[props.PropertyEncoder](nil),
			"calendar-user-type":               mo.Ok[props.PropertyEncoder](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Verify response has expected elements
		root := doc.Root()
		assert.NotNil(t, root)

		// Check if properties were properly set
		response := root.FindElement("//d:response")
		assert.NotNil(t, response)

		// Check href
		href := response.FindElement("//d:href")
		assert.NotNil(t, href)
		assert.Equal(t, "/calendars/user1/", href.Text())

		// Check calendar user type
		calUserType := response.FindElement("//d:prop/cal:calendar-user-type")
		assert.NotNil(t, calUserType)
		assert.Equal(t, "individual", calUserType.Text())

		// Check displayname
		displayName := response.FindElement("//d:prop/d:displayname")
		assert.NotNil(t, displayName)
		assert.Equal(t, "Calendar Home", displayName.Text())

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 2: Principal URL property
	t.Run("Principal URL property", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		principalResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}
		mockURLConverter.On("EncodePath", principalResource).Return("/principals/user1/", nil)

		// Create request map with principal-url property
		req := propfind.ResponseMap{
			"principal-url": mo.Ok[props.PropertyEncoder](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if principal-url was properly set
		root := doc.Root()
		principalURL := root.FindElement("//d:propstat/d:prop/d:principal-url/d:href")
		assert.NotNil(t, principalURL)
		assert.Equal(t, "/principals/user1/", principalURL.Text())

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 3: ACL property
	t.Run("ACL property", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		principalResource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}
		mockURLConverter.On("EncodePath", principalResource).Return("/principals/user1/", nil)

		// Create request map with acl property
		req := propfind.ResponseMap{
			"acl": mo.Ok[props.PropertyEncoder](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if acl was properly set
		root := doc.Root()

		// Verify ACL contains read and write privileges
		ace := root.FindElement("//d:propstat/d:prop/d:acl/d:ace")
		assert.NotNil(t, ace)

		// Verify principal href
		principal := ace.FindElement("//d:principal/d:href")
		assert.NotNil(t, principal)
		assert.Equal(t, "/principals/user1/", principal.Text())

		// Verify grant privileges
		readPriv := ace.FindElement("//d:grant/d:privilege/d:read")
		assert.NotNil(t, readPriv)

		writePriv := ace.FindElement("//d:grant/d:privilege/d:write")
		assert.NotNil(t, writePriv)

		mockURLConverter.AssertExpectations(t)
	})

	// Test case 4: Calendar limits properties
	t.Run("Calendar limits properties", func(t *testing.T) {
		// Setup expectations
		mockURLConverter.On("EncodePath", resource).Return("/calendars/user1/", nil)

		// Create request map with calendar limits properties
		req := propfind.ResponseMap{
			"max-resource-size":          mo.Ok[props.PropertyEncoder](nil),
			"min-date-time":              mo.Ok[props.PropertyEncoder](nil),
			"max-date-time":              mo.Ok[props.PropertyEncoder](nil),
			"max-instances":              mo.Ok[props.PropertyEncoder](nil),
			"max-attendees-per-instance": mo.Ok[props.PropertyEncoder](nil),
		}

		// Call function
		doc, err := h.handlePropfindHomeSet(req, ctx.Resource)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		// Check if properties were properly set
		root := doc.Root()

		maxSize := root.FindElement("//d:propstat/d:prop/cal:max-resource-size")
		assert.NotNil(t, maxSize)
		assert.Equal(t, "10485760", maxSize.Text())

		maxInstances := root.FindElement("//d:propstat/d:prop/cal:max-instances")
		assert.NotNil(t, maxInstances)
		assert.Equal(t, "100000", maxInstances.Text())

		maxAttendees := root.FindElement("//d:propstat/d:prop/cal:max-attendees-per-instance")
		assert.NotNil(t, maxAttendees)
		assert.Equal(t, "100", maxAttendees.Text())

		mockURLConverter.AssertExpectations(t)
	})
}

func TestFetchChildren(t *testing.T) {
	// Setup
	mockURLConverter := new(MockURLConverter)
	mockStorage := new(storage.MockStorage)

	h := &CaldavHandler{
		URLConverter: mockURLConverter,
		Storage:      mockStorage,
	}

	// Test case 1: Depth 0 returns empty
	t.Run("Depth 0", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		resources, err := h.fetchChildren(0, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 2: storage.ResourceObject has no children
	t.Run("Resource Object", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 3: storage.ResourcePrincipal has no children
	t.Run("Resource Principal", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourcePrincipal,
		}

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Empty(t, resources)
	})

	// Test case 4: storage.ResourceCollection with children
	t.Run("Resource Collection", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		// Mock storage response
		mockStorage.On("GetObjectPathsInCollection", "cal1").Return([]string{
			"/calendars/user1/cal1/event1.ics",
			"/calendars/user1/cal1/event2.ics",
		}, nil)

		// Mock URL converter for each path
		event1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		event2Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event2",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event1.ics").Return(event1Resource, nil)
		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event2.ics").Return(event2Resource, nil)

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "event1", resources[0].ObjectID)
		assert.Equal(t, "event2", resources[1].ObjectID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})

	// Test case 5: storage.ResourceHomeSet with children
	t.Run("Resource HomeSet", func(t *testing.T) {
		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		// Mock storage response - using proper Calendar struct
		mockStorage.On("GetUserCalendars", "user1").Return([]Calendar{
			{Path: "/calendars/user1/cal1/"},
			{Path: "/calendars/user1/cal2/"},
		}, nil)

		// Mock URL converter for each path
		cal1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		cal2Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal2",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/").Return(cal1Resource, nil)
		mockURLConverter.On("ParsePath", "/calendars/user1/cal2/").Return(cal2Resource, nil)

		resources, err := h.fetchChildren(1, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "cal1", resources[0].CalendarID)
		assert.Equal(t, "cal2", resources[1].CalendarID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})

	// Test case 6: Recursive fetching with depth > 1
	t.Run("Recursive fetching", func(t *testing.T) {
		// Setup fresh mocks for this specific test
		mockURLConverter := new(MockURLConverter)
		mockStorage := new(storage.MockStorage)

		h := &CaldavHandler{
			URLConverter: mockURLConverter,
			Storage:      mockStorage,
		}

		parent := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceHomeSet,
		}

		// Mock HomeSet -> Calendar responses
		mockStorage.On("GetUserCalendars", "user1").Return([]Calendar{
			{Path: "/calendars/user1/cal1/"},
		}, nil)

		cal1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceCollection,
			CalendarID:   "cal1",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/").Return(cal1Resource, nil)

		// Mock Calendar -> Events responses
		mockStorage.On("GetObjectPathsInCollection", "cal1").Return([]string{
			"/calendars/user1/cal1/event1.ics",
		}, nil)

		event1Resource := Resource{
			UserID:       "user1",
			ResourceType: storage.ResourceObject,
			CalendarID:   "cal1",
			ObjectID:     "event1",
		}

		mockURLConverter.On("ParsePath", "/calendars/user1/cal1/event1.ics").Return(event1Resource, nil)

		resources, err := h.fetchChildren(2, parent)

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "cal1", resources[0].CalendarID)
		assert.Equal(t, "event1", resources[1].ObjectID)

		mockStorage.AssertExpectations(t)
		mockURLConverter.AssertExpectations(t)
	})
}
