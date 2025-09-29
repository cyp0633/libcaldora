# libcaldora Contributor Notes

## Project overview
- `davclient/` contains the public CalDAV client. Treat exported APIs as stable—if you need to change a method signature or struct, also update the README examples and affected callers/tests.
- `internal/` holds HTTP/XML helpers that should remain implementation details. Reuse helpers here instead of duplicating HTTP or XML parsing logic in `davclient` or `server`.
- `server/` is an in-progress CalDAV server implementation. Handlers depend on the storage interfaces in `server/storage/` and the example app under `server/example/` doubles as a manual smoke test.

## Coding style
- Format all Go code with `gofmt` (or `go fmt ./...`) before committing. The repository follows standard Go formatting (tabs for indentation, grouped imports, etc.).
- Keep exported identifiers documented with Go-style doc comments when you add or modify public surface area.
- Prefer structured logging via Go's `slog` package. Follow the existing pattern of passing loggers through options/config structs instead of using global loggers.
- When adding CalDAV-specific behavior, favor small, composable helpers in the relevant package over large handlers with duplicated logic.

## Testing & validation
- Run `go test ./...` before submitting changes. Network-backed integration tests in `davclient/integration_test.go` are skipped automatically unless the `CALDAV_*` environment variables are provided, so local test runs should still pass without those credentials.
- If you modify the server handlers or storage interfaces, exercise the example server (`go run ./server/example`) to ensure basic PROPFIND/REPORT/PUT flows still work.

## Dependency management
- Keep the dependency footprint small. Update both `go.mod` and `go.sum` with `go mod tidy` if you add or upgrade modules, and avoid introducing unused dependencies.

## Documentation
- Update `README.md` and `server/example/README.md` when public usage patterns or example credentials change.
- Inline comments explaining CalDAV or XML nuances are encouraged where behavior is non-obvious.
