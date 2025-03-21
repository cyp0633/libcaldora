package storage

import (
	"testing"
)

func TestParseResourcePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *ResourcePath
		wantErr bool
	}{
		{
			name: "valid user principal",
			path: "/u/user123",
			want: &ResourcePath{
				Type:   ResourceTypePrincipal,
				UserID: "user123",
			},
		},
		{
			name: "valid calendar home",
			path: "/u/user123/cal",
			want: &ResourcePath{
				Type:   ResourceTypeCalendarHome,
				UserID: "user123",
			},
		},
		{
			name: "valid calendar",
			path: "/u/user123/cal/calendar456",
			want: &ResourcePath{
				Type:       ResourceTypeCalendar,
				UserID:     "user123",
				CalendarID: "calendar456",
			},
		},
		{
			name: "valid calendar object",
			path: "/u/user123/evt/event789",
			want: &ResourcePath{
				Type:     ResourceTypeObject,
				UserID:   "user123",
				ObjectID: "event789",
			},
		},
		{
			name:    "invalid path format",
			path:    "/invalid/path",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "invalid user path",
			path:    "/u/",
			wantErr: true,
		},
		{
			name:    "invalid calendar home path",
			path:    "/u/user123/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResourcePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResourcePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("ParseResourcePath() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.UserID != tt.want.UserID {
					t.Errorf("ParseResourcePath() UserID = %v, want %v", got.UserID, tt.want.UserID)
				}
				if got.CalendarID != tt.want.CalendarID {
					t.Errorf("ParseResourcePath() CalendarID = %v, want %v", got.CalendarID, tt.want.CalendarID)
				}
				if got.ObjectID != tt.want.ObjectID {
					t.Errorf("ParseResourcePath() ObjectID = %v, want %v", got.ObjectID, tt.want.ObjectID)
				}
			}
		})
	}
}
