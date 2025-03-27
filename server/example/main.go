package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/cyp0633/libcaldora/server"
	auth "github.com/cyp0633/libcaldora/server/auth/memory"
	store "github.com/cyp0633/libcaldora/server/storage/memory"
)

var (
	addr    = flag.String("addr", ":8080", "HTTP service address")
	baseURI = flag.String("base-uri", "/caldav", "Base URI for the CalDAV server")
)

func main() {
	flag.Parse()

	// Set up logger with JSON handler for structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create a new in-memory storage backend
	storage := store.New(store.WithLogger(logger))

	// Create a new in-memory auth store and add a test user
	authStore := auth.New(auth.WithLogger(logger))
	if err := authStore.AddUser("testuser", "password"); err != nil {
		logger.Error("failed to create test user", "error", err)
		os.Exit(1)
	}

	// Create the CalDAV server with auth enabled
	srv, err := server.New(server.Options{
		Storage: storage,
		BaseURI: *baseURI,
		Auth:    authStore,
		Realm:   "CalDAV Test Server",
		Logger:  logger,
	})
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Create an HTTP mux and register the CalDAV server
	mux := http.NewServeMux()
	// Handle well-known caldav redirect
	mux.HandleFunc("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		// Add trailing slash to the target
		target := *baseURI + "/" // Now redirects to "/caldav/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	mux.Handle(*baseURI+"/", srv)

	// Add basic instructions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `CalDAV Server Example

This is a basic CalDAV server implementation using libcaldora.
The server is running with the following configuration:

Base URI: %s
Authentication:
- Username: testuser
- Password: password

Test URLs:
- CalDAV discovery:     %s/.well-known/caldav
- User principal:       %s/u/testuser
- Calendar home:        %s/u/testuser/cal
- Calendar collection:  %s/u/testuser/cal/personal
- Calendar object:      %s/u/testuser/evt/123

Supported HTTP methods:
- OPTIONS: Query server capabilities
- PROPFIND: Query resource properties
- REPORT: Query calendar data
- GET: Retrieve calendar object
- PUT: Create/update calendar object
- DELETE: Remove calendar or object
- MKCOL: Create calendar collection

Note: All URLs except /.well-known/caldav require Basic Authentication.
`, *baseURI, *baseURI, *baseURI, *baseURI, *baseURI, *baseURI)
	})

	logger.Info("starting CalDAV server",
		"addr", *addr,
		"base_uri", *baseURI)

	if err := http.ListenAndServe(*addr, mux); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
