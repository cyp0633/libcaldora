package server

import "context"

// BasicAuthInput contains credentials and request metadata for Basic auth.
type BasicAuthInput struct {
	Username   string
	Password   string
	UserAgent  string
	RemoteAddr string
}

// BasicAuthProvider validates Basic-auth credentials and returns user ID.
//
// Returning an empty user ID is treated as authentication failure.
type BasicAuthProvider interface {
	AuthenticateBasic(ctx context.Context, input BasicAuthInput) (userID string, err error)
}
