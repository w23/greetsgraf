# Greetsgraf Agent Guidelines

## Build, Lint, and Test Commands

### Build
```bash
go build -ldflags="-extldflags=-static" -tags "sqlite_omit_load_extension sqlite_fts5" -a -o greetsgraf
```
**Requirements**: Go 1.24+ with CGO enabled. Produces a static binary suitable for deployment.

### Linting
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.49.0
golangci-lint run
```
No `.golangci.yml` config file exists; linter uses default rules.

### Testing
```bash
go test ./...
go test -run TestMain ./...
go test -v ./...
go test -run TestPouetImport -v ./...
go test -run TestGreet -v ./...
```
**Test Data**: Test files use gzipped JSON dumps from pouet.net in `test/` directory.
Use `pouet.sh` script to update/populate test data files.

**Test Files**:
- `database_test.go` - Pouet database import and queries
- `greets_test.go` - Greetings database operations (create, delete, invalid handling)

---

## Code Style Guidelines

### Imports
- Group imports: standard library first, then external libraries, then local packages
- Use `gofmt` for formatting (standard Go formatter)
- Import paths should be absolute and follow Go module structure
- Remove unused imports; use blank identifier `_` only when side effects are needed

### Formatting
- Follow standard Go formatting with `gofmt`
- Use tabs for indentation (`.editorconfig` specifies `indent_style = tab`)
- Maximum line length: 120 characters
- One statement per line where possible
- Proper spacing around operators and after commas

### Types
- Use meaningful type names (e.g., `Group`, `Prod`, `Greet`, `DatabaseStats`)
- Use camelCase for struct field names (e.g., `GreeteeID`, `TotalGreets`)
- Exported types should have proper documentation comments
- Struct fields must be capitalized to export them
- Use `gorm.Model` embedded struct for database models

### Naming Conventions
- Use camelCase for variables and function names (e.g., `getGreets`, `autoMigrate`)
- Use PascalCase for exported identifiers (types, functions, etc.)
- Use `ID` not `Id` for identifiers (e.g., `ProdID`, `GroupID`)
- Use descriptive names over abbreviations when possible
- Avoid single letter variable names except for loop counters

### Error Handling
- Handle errors immediately after they occur
- Log errors appropriately using the standard `log` package
- Return errors with context using `errors.New()` or `fmt.Errorf()` with `%w` verb
- Prefer returning errors over panicking unless it's a truly unrecoverable state
- Use `gorm.ErrRecordNotFound` for checking record-not-found conditions

### Documentation
- Add comments to exported functions, types, and methods
- Use godoc-style comments: `// FunctionName does X` on single line
- Document public APIs clearly, including parameters and return values
- Use inline comments for complex logic or non-obvious implementation details
- Document database schema in struct field comments when non-standard

### Code Organization
- Keep functions small and focused on a single responsibility
- Group related functions together in the same file when appropriate
- Use constants for values that should not change
- Avoid deeply nested code structures
- Prefer early returns over else clauses where possible
- Related types and their methods should be in the same file

### Go-Specific Rules
- Use `var` keyword for global declarations, not `:=`
- Use `context` package for request-scoped values and timeouts
- Use `gorm` for database operations with proper error handling
- Handle HTTP status codes correctly in API responses (200, 201, 400, 404, 500)
- Use `http.HandlerFunc` for middleware functions
- Use `r.Context().Value()` to access request-scoped database connections

---

## Project-Specific Guidelines

### Database
- Use two separate SQLite databases: `pouet.db` (read-only) and `greets.db` (writable)
- FTS5 full-text search enabled for groups and prods
- Many-to-many relationships between groups and prods via join tables
- Use `gorm` ORM with proper error handling for all database operations

### API Endpoints (v1)
- `/v1/stats` - Get database statistics
- `/v1/groups/search?name=...` - Search groups
- `/v1/groups/greeted?limit=...` - Get most greeted groups
- `/v1/groups/{id}/greets` - Get greets for a group
- `/v1/prods/search?name=...` - Search productions
- `/v1/prods/{id}/` - Get prod details with greets
- `/v1/prods/{id}/greets` - Get greets for a prod
- `/v1/greets/` (POST) - Create a greet
- `/v1/greets/{id}` (DELETE) - Delete a greet

### Main CLI Flags
- `-create` - Create new database from pouet dumps
- `-prods` / `-groups` - Pouet data dump files (JSON format)
- `-serve` - Start HTTP server
- `-listen` - Server address (default: localhost:8000)
- `-static` - Serve static files from path
- `-index` - Build FTS5 index

---

## Cursor/Copilot Rules
No specific Cursor or Copilot rules defined in this repository.

---

## Development Best Practices

### Working with GORM
- Always check for `gorm.ErrRecordNotFound` when querying single records
- Use `db.FirstOrCreate()` when you want to insert only if not exists
- Use `db.Scopes()` for reusable query logic
- Use `db.Transaction()` for multiple related operations
- Prefer `db.Select()` to explicitly specify fields when updating

### Database Operations
- Open both databases on startup; use context to access the correct one
- Use `db.AutoMigrate()` on startup to ensure schema is up to date
- For FTS5 search, use `db.Raw()` with MATCH queries
- Always close database connections on shutdown

### HTTP Handlers
- Validate all input parameters before processing
- Return appropriate HTTP status codes (200, 201, 400, 404, 500)
- Use `json.NewEncoder().Encode()` for JSON responses
- Log request errors for debugging but don't expose internal details to clients

