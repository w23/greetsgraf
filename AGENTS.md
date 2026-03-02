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
### Running Tests

All tests use in-memory SQLite database with unique names per test (e.g., `file:TestName?mode=memory&cache=shared`) and FTS5 support. This requires extra arguments for the `go test` command. Use `go-test.sh` helper script to run tests, as regular `go test` without extra arguments is guaranteed to fail.

### Making new tests

All tests should:
1. Use `require` and `assert` packages to check for expected values. E.g. `require.NoError()` for error checks, `assert.Equal()` or other for less critical checks that don't block further test process.
2. When comparing response results with expected values, do a full struct type variable comparison as opposed to individual fields comparison.
3. Same goes for arrays. Compare agains the entire array, instead of checking for length and individual items.
4. For REST API tests, use the generic `makeRequest[T]` helper to DRY up HTTP request code. The function returns unmarshalled struct. Example:
```go
stats := makeRequest[StatsResponse](t, server.URL+"/v1/stats", http.StatusOK)
assert.Equal(t, 420, stats.TotalProds)
```

Top level (`func Test...`) tests should:
1. Use `setupTestDatabaseWithPouetData()` function to create a database with test Pouet data.
2. Use `setupTestServer()` to create a test HTTP server with test Pouet data.

Test data files for Pouet database contents are available in `./test/` directory:
- `pouet-groups.json.gz` - test group data
- `pouet-prods.json.gz` - test prod data

Prefer minimizing the number of top level tests. Include many similar-themed subtests within one top level test.

Subtest MUST always use the same database and server created at the top of the toplevel test.

Subtest names MUST only consist of alphanumeric latin characters only.

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
)
```

### Naming Conventions
- Struct fields: PascalCase (e.g., `GroupName`, `ProdID`)
- Functions/variables: camelCase (e.g., `getCounts`, `dbConnection`)
- Database columns: snake_case (via GORM tags)
- Constants: PascalCase

### Type Declarations
- Use database/sql for database operations
- Define response types explicitly for API contracts
- Use pointers for optional values in structs
- Use `any` instead of `interface{}`

### Error Handling
- Log errors with context: `log.Printf("Error: %v", err)`
- Use `respondErrJson` for HTTP errors
- Check database errors from Query/QueryRow/Exec/Prepare calls
- Return early on errors

### Database
- Use database/sql for all database operations
- Begin transactions with `tx := db.Begin()`
- Commit with `tx.Commit()` or rollback with `defer tx.Rollback()`
- Use prepared statements for repeated queries

## API Conventions
- JSON responses with `Content-Type: application/json`
- Consistent error structure: `{"Error": "message"}`
- HTTP status codes match request outcome
- Pagination via query params where needed

## Existing Rules
- Follow `.editorconfig` settings for all files
- Run `go vet ./...` before commits to catch static analysis issues

## Go Specific Notes

### File Organization
- Multi-file application: `greetsgraf.go` (main), `database.go` (models & DB operations), `server.go` (HTTP handlers), `pouet.go` (Pouet data storage)
- All code in `package main`

### Dual-Database Architecture
The application uses two separate SQLite databases:
1. **User Data Database** (`greets.db`): Stores greets, user data
2. **Pouet Database** (`pouet.db`): Stores static Pouet group and prod data with FTS5 indexes

### Pre-commit Checklist
- Run `go vet ./...` for static analysis
- Run `go fmt ./...` for formatting

### Model Definitions
- Define response types explicitly for API contracts
- Use `gorm:"-"` tag for computed/derived fields not stored in DB (for API compatibility)

### HTTP Handlers
- Context functions (`ProdContext`, `GroupContext`) extract URL parameters
- All handlers receive `http.ResponseWriter` and `*http.Request`
- Use `respondJson` and `respondErrJson` for consistent response formatting

### Database Operations
- Use transactions for related operations: `tx := db.Begin()`
- Always check database errors from operations
- Use `tx.Commit()` on success, `tx.Rollback()` on error

### JSON Handling
- Use `json.NewDecoder(r.Body).Decode(&target)` for request bodies
- Use `json.Marshal(payload)` for response payloads
- Handle decode errors with appropriate HTTP status codes

## Frontend Architecture

### File Structure (`./static/` directory)
- `index.html` - Main page with stats and group search
- `edit.html` - Edit page for adding/finding greets
- `style.css` - All styling (minimalistic, vanilla CSS)
- `utils.js` - Core utilities: `Tag()`, `Text()`, `sendRequest()`, `Autocomplete` class
- `common.js` - Shared autocomplete for groups
- `index.js` - Main page logic (stats, most greeted groups)
- `edit.js` - Edit page logic (prod search, greet management)

### Backend Communication
The frontend communicates with the backend via `XMLHttpRequest` (no fetch/AJAX libraries used):
- `sendRequest(method, path, query, body, successCb, errorCb)` - Main HTTP helper
- All API calls use JSON responses with `Content-Type: application/json`

### API Endpoints
- `GET /v1/stats` - Returns stats (ProdsWithGreets, TotalProds, etc.)
- `GET /v1/groups/search?name=xxx` - Search groups
- `GET /v1/groups/greeted?limit=N` - Get most greeted groups
- `GET /v1/groups/{id}/greets` - Get greets for a group
- `GET /v1/prods/search?name=xxx` - Search productions
- `GET /v1/prods/{id}` - Get prod details with greets
- `POST /v1/greets/` - Create a greet (body: `{ProdId, GroupId, Note}`)
- `DELETE /v1/greets/{id}` - Delete a greet

### Key Frontend Patterns
- **Vanilla JS only** - No frameworks, no build step
- **Tag function** - `Tag(name, attrs, body, children)` creates DOM elements
- **Autocomplete** - Custom class in `utils.js` with keyboard navigation
- **Debounce** - 200ms debounce on autocomplete searches
- **Inline styles** - Many styles inline (e.g., `style="width: 100%"`)

### CSS Architecture
- Single `style.css` file
- Uses CSS variables (currently minimal)
- Flexbox layout for main wrapper
- Custom autocomplete dropdown styling
- Simple table styling for data display

### Adding New Features
1. Add HTML elements in respective `.html` file
2. Add CSS styles to `style.css`
3. Add logic to appropriate `.js` file
4. Use `Tag()` helper for dynamic DOM creation
5. Use `sendRequest()` for API calls
6. Use `autocompleteGroup()` or `Autocomplete` for search inputs
