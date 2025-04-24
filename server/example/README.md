# libcaldora Example CalDAV Server

This is a simple example implementation of a CalDAV server using the libcaldora library. It demonstrates how to set up a working CalDAV server that clients can connect to and perform calendar operations.

## Features

- Basic CalDAV server implementation
- In-memory storage using MockStorage
- Sample users and calendars pre-configured
- Calendar creation and event management
- Basic authentication

## Running the Server

To run the example server:

```bash
go run main.go
```

The server will start on port 8080 by default.

## Available Users

The example server comes with two pre-configured users:

- **Username:** alice, **Password:** password
- **Username:** bob, **Password:** password

Each user has multiple calendars with sample events.

## Connecting with CalDAV Clients

You can connect to this server using any CalDAV-compatible client with these settings:

- **Server URL:** `http://localhost:8080/caldav/`
- **Username:** alice or bob
- **Password:** password

### Compatible Clients

This server has been tested with:

- Thunderbird Lightning
- Apple Calendar
- GNOME Calendar
- DAVx⁵ (for Android)

## Implementation Details

This example demonstrates:

1. How to initialize and configure a CalDAV server
2. Basic authentication handling
3. Calendar collection and object management
4. Sample event creation
5. HTTP server integration

The server uses MockStorage for simplicity, which is an in-memory storage implementation. In a production environment, you would implement a persistent storage backend (database, file system, etc.).

## Customization

You can customize the server by modifying:

- `serverAddr` - Change the port the server listens on
- `caldavPrefix` - Change the URL prefix for CalDAV endpoints
- `serverRealm` - Change the authentication realm name
- `maxDepth` - Adjust the maximum depth for PROPFIND/REPORT operations

## Testing the Server

You can test the server with curl or other HTTP tools:

```bash
# Get OPTIONS (Server capabilities)
curl -v -X OPTIONS http://localhost:8080/caldav/ -u alice:password

# List calendars (PROPFIND)
curl -v -X PROPFIND http://localhost:8080/caldav/alice/cal/ -u alice:password -H "Depth: 1" -H "Content-Type: application/xml" --data '<?xml version="1.0" encoding="utf-8"?><propfind xmlns="DAV:"><prop><resourcetype/><displayname/></prop></propfind>'
```