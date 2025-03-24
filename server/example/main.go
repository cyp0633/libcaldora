package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/cyp0633/libcaldora/server"
	"github.com/cyp0633/libcaldora/server/storage/memory"
)

var (
	addr    = flag.String("addr", ":8080", "HTTP service address")
	baseURI = flag.String("base-uri", "/caldav", "Base URI for the CalDAV server")
)

func main() {
	flag.Parse()

	// Create a new in-memory storage backend
	// In a real application, you would implement your own storage backend
	store := memory.New()

	// Create the CalDAV server
	srv, err := server.New(store, *baseURI)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create an HTTP mux and register the CalDAV server
	mux := http.NewServeMux()
	// Handle well-known caldav redirect
	mux.HandleFunc("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		// Preserve query parameters in redirect
		target := *baseURI
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
`, *baseURI, *baseURI, *baseURI, *baseURI, *baseURI, *baseURI)
	})

	log.Printf("Starting CalDAV server on %s with base URI %s", *addr, *baseURI)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
