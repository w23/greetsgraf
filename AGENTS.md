# Greetsgraf Developer Guidelines

## Build & Run Commands

```bash
# Build the application
CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -ldflags="-extldflags=-static" -tags "sqlite_omit_load_extension sqlite_fts5" -o greetsgraf

# Build with CGO enabled (for SQLite FTS5 support)
CGO_CFLAGS="-D_LARGEFILE64_SOURCE" CGO_ENABLED=1 go build -o greetsgraf

# Run the application (create database from pouet dumps)
./greetsgraf -create -prods=path/to/prods.json.gz -groups=path/to/groups.json.gz -db=greets.db

# Build and run with FTS5 index
./greetsgraf -index -db=greets.db

# Start the HTTP server
./greetsgraf -serve -listen=:8000 -db=greets.db

# Docker build
docker build -t greetsgraf .

# Go vet for static analysis
go vet ./...

# Go fmt for formatting
go fmt ./...
```

Note:
- `CGO_CFLAGS="-D_LARGEFILE64_SOURCE"` is required with musl libc. Set it if encountering `pread64 undeclared` errors.

## Testing

Test data files are available in `./test/` directory:
- `pouet-groups.json.gz` - test group data
- `pouet-prods.json.gz` - test prod data

### Running Tests

All tests use in-memory SQLite database (`:memory:?cache=shared`) with FTS5 support. This requires extra arguments for the `go test` command. Use `go-test.sh` helper script to run tests.

### Test Structure

When writing tests:
1. Use `SetupDatabase()` with `SetupArgs` to initialize the database
2. Use `httptest.NewServer(Server(db))` to create a test HTTP server
3. Use `require` and `assert` packages to check for expected values. E.g. `requre.NoError()` for error checks, `assert.Equal()` or other for less critical checks that don't block further test process.

## Code Style Guidelines

### Formatting
- Use tabs for indentation (per `.editorconfig`)
- 80-character line width
- Trailing newlines in all files
- Trim trailing whitespace

### Imports
- Standard library imports first, then third-party
- Group imports by package source
- Example:
```go
import (
    "context"
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "gorm.io/gorm"
)
```

### Naming Conventions
- Struct fields: PascalCase (e.g., `GroupName`, `ProdID`)
- Functions/variables: camelCase (e.g., `getCounts`, `dbConnection`)
- Database columns: snake_case (via GORM tags)
- Constants: PascalCase

### Type Declarations
- Use GORM for ORM operations
- Define response types explicitly for API contracts
- Use pointers for optional values in structs

### Error Handling
- Log errors with context: `log.Printf("Error: %v", err)`
- Use `respondErrJson` for HTTP errors
- Check GORM errors: `if db.Error != nil { ... }`
- Return early on errors

### Database
- Use GORM for all database operations
- Begin transactions with `tx := db.Begin()`
- Commit with `tx.Commit()` or rollback with `tx.Rollback()`
- Use foreign key constraints and indexes where appropriate

## Docker
- Base image: `golang:1.16-alpine`
- CGO enabled for SQLite
- Static binary for scratch base image
- Exposes port 8000

## API Conventions
- JSON responses with `Content-Type: application/json`
- Consistent error structure: `{"Error": "message"}`
- HTTP status codes match request outcome
- Pagination via query params where needed

## Existing Rules
- No Cursor rules or Copilot rules found
- Follow `.editorconfig` settings for all files

## Go Specific Notes

### File Organization
- Single-file application (`greetsgraf.go`) with all models, handlers, and business logic
- No package structure - all code in `package main`

### Model Definitions
- Use GORM tags for database mapping
- Define response types explicitly for API contracts
- Use `gorm:"-"` tag for computed/derived fields not stored in DB

### HTTP Handlers
- Context functions (`ProdContext`, `GroupContext`) extract URL parameters
- All handlers receive `http.ResponseWriter` and `*http.Request`
- Use `respondJson` and `respondErrJson` for consistent response formatting

### Database Operations
- Use transactions for related operations: `tx := db.Begin()`
- Always check `db.Error` after GORM operations
- Use `tx.Commit()` on success, `tx.Rollback()` on error

### JSON Handling
- Use `json.NewDecoder(r.Body).Decode(&target)` for request bodies
- Use `json.Marshal(payload)` for response payloads
- Handle decode errors with appropriate HTTP status codes
