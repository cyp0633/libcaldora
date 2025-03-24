package memory

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"sync"

	"github.com/cyp0633/libcaldora/server/auth"
	"github.com/cyp0633/libcaldora/server/storage"
)

// User represents a user in the memory store
type User struct {
	Username string
	Password string // In production this should be hashed
}

// Store implements an in-memory authentication store
type Store struct {
	mu    sync.RWMutex
	users map[string]User // map[username]User
}

// New creates a new in-memory authentication store
func New() *Store {
	return &Store{
		users: make(map[string]User),
	}
}

// AddUser adds a new user to the store
func (s *Store) AddUser(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return fmt.Errorf("user already exists: %s", username)
	}

	s.users[username] = User{
		Username: username,
		Password: password,
	}

	return nil
}

// Authenticate implements auth.Authenticator
func (s *Store) Authenticate(ctx context.Context, creds auth.Credentials) (*auth.Principal, error) {
	s.mu.RLock()
	user, exists := s.users[creds.Username]
	s.mu.RUnlock()

	if !exists {
		return nil, &auth.Error{
			Type:    auth.ErrInvalidCredentials,
			Message: "invalid username or password",
		}
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(creds.Password)) != 1 {
		return nil, &auth.Error{
			Type:    auth.ErrInvalidCredentials,
			Message: "invalid username or password",
		}
	}

	return &auth.Principal{ID: creds.Username}, nil
}

// ValidateAccess implements auth.Authenticator
func (s *Store) ValidateAccess(ctx context.Context, principal *auth.Principal, path string) error {
	if principal == nil {
		return &auth.Error{
			Type:    auth.ErrUnauthorized,
			Message: "authentication required",
		}
	}

	// Strip any base URI prefix from the path before parsing
	userPath := path
	if idx := strings.Index(path, "/u/"); idx != -1 {
		userPath = path[idx:]
	}
	resourcePath, err := storage.ParseResourcePath(userPath)
	// For paths that should be user paths (contain /u/), enforce user check
	if err == nil && resourcePath.UserID != "" && resourcePath.UserID != principal.ID {
		return &auth.Error{
			Type:    auth.ErrForbidden,
			Message: fmt.Sprintf("access denied to resource: %s", path),
		}
	}

	return nil
}
