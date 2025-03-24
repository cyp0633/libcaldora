package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/cyp0633/libcaldora/server/auth"
	"github.com/cyp0633/libcaldora/server/handlers"
	"github.com/cyp0633/libcaldora/server/storage"
)

// Server represents a CalDAV server
type Server struct {
	storage storage.Storage
	baseURI string
	handler http.Handler
	logger  *slog.Logger
}

// Options configures a CalDAV server
type Options struct {
	Storage storage.Storage
	BaseURI string
	Auth    auth.Authenticator
	Realm   string
	Logger  *slog.Logger // Optional logger, defaults to slog.Default()
}

// New creates a new CalDAV server
func New(opts Options) (*Server, error) {
	if opts.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	s := &Server{
		storage: opts.Storage,
		baseURI: opts.BaseURI,
		logger:  logger,
	}

	// Create router with handlers
	router := handlers.NewRouter(s.storage, s.baseURI, s.logger)

	// Apply authentication middleware if configured
	var handler http.Handler = router
	if opts.Auth != nil {
		handler = auth.Middleware(auth.MiddlewareOptions{
			Authenticator: opts.Auth,
			Realm:         opts.Realm,
			Logger:        s.logger,
		})(handler)
	}

	s.handler = handler
	return s, nil
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}
