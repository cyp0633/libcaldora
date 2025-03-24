package auth

import (
	"context"
	"fmt"
)

// Principal represents an authenticated user or entity
type Principal struct {
	ID string
}

// Credentials represents authentication credentials
type Credentials struct {
	Username string
	Password string
}

// ErrorType represents the type of authentication error
type ErrorType string

const (
	ErrInvalidCredentials ErrorType = "invalid_credentials"
	ErrUnauthorized       ErrorType = "unauthorized"
	ErrForbidden          ErrorType = "forbidden"
)

// Error represents an authentication-related error
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Authenticator defines the interface for authentication providers
type Authenticator interface {
	// Authenticate validates credentials and returns a Principal if successful
	Authenticate(ctx context.Context, creds Credentials) (*Principal, error)

	// ValidateAccess checks if a principal has access to a given path
	ValidateAccess(ctx context.Context, principal *Principal, path string) error
}
