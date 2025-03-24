package memory

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
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
	mu     sync.RWMutex
	users  map[string]User // map[username]User
	logger *slog.Logger
}

// New creates a new in-memory authentication store
func New(opts ...Option) *Store {
	s := &Store{
		users:  make(map[string]User),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Option represents a configuration option for the Store
type Option func(*Store)

// WithLogger sets the logger for the store
func WithLogger(logger *slog.Logger) Option {
	return func(s *Store) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// AddUser adds a new user to the store
func (s *Store) AddUser(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		s.logger.Warn("failed to add user: already exists",
			"username", username)
		return fmt.Errorf("user already exists: %s", username)
	}

	s.users[username] = User{
		Username: username,
		Password: password,
	}

	s.logger.Info("user added successfully",
		"username", username)

	return nil
}

// Authenticate implements auth.Authenticator
func (s *Store) Authenticate(ctx context.Context, creds auth.Credentials) (*auth.Principal, error) {
	s.mu.RLock()
	user, exists := s.users[creds.Username]
	s.mu.RUnlock()

	if !exists {
		s.logger.Info("authentication failed: user not found",
			"username", creds.Username)
		return nil, &auth.Error{
			Type:    auth.ErrInvalidCredentials,
			Message: "invalid username or password",
		}
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(creds.Password)) != 1 {
		s.logger.Info("authentication failed: invalid password",
			"username", creds.Username)
		return nil, &auth.Error{
			Type:    auth.ErrInvalidCredentials,
			Message: "invalid username or password",
		}
	}

	s.logger.Debug("authentication successful",
		"username", creds.Username)

	return &auth.Principal{ID: creds.Username}, nil
}

// ValidateAccess implements auth.Authenticator
func (s *Store) ValidateAccess(ctx context.Context, principal *auth.Principal, path string) error {
	if principal == nil {
		s.logger.Info("access validation failed: no principal")
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
		s.logger.Warn("access validation failed: forbidden",
			"username", principal.ID,
			"requested_user", resourcePath.UserID,
			"path", path)
		return &auth.Error{
			Type:    auth.ErrForbidden,
			Message: fmt.Sprintf("access denied to resource: %s", path),
		}
	}

	s.logger.Debug("access validation successful",
		"username", principal.ID,
		"path", path)

	return nil
}
