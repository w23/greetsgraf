# Greetsgraf Agent Guidelines

## Build, Lint, and Test Commands

### Build
```bash
go build -ldflags="-extldflags=-static" -tags "sqlite_omit_load_extension sqlite_fts5" -a -o greetsgraf
```

### Linting
```bash
# Install golangci-lint first:
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.49.0

# Run linter
golangci-lint run
```

### Testing
```bash
# Run all tests
go test ./...

# Run a specific test (example)
go test -run TestMain ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test with verbose output
go test -run TestPouetImport -v ./...
```

## Code Style Guidelines

### Imports
- Group imports by standard library, external libraries, and local packages
- Use `gofmt` for formatting (standard Go formatter)
- Import paths should be absolute and follow Go module structure
- Avoid unused imports

### Formatting
- Follow standard Go formatting with `gofmt`
- Use tabs for indentation as specified in `.editorconfig`
- Maximum line length of 120 characters
- One statement per line where possible
- Proper spacing around operators and after commas

### Types
- Use meaningful type names (e.g., `Group`, `Prod`, `Greet`)
- Use camelCase for field names in structs
- Exported types should have proper documentation comments
- Struct fields should be capitalized to export them

### Naming Conventions
- Use camelCase for variables and function names
- Use PascalCase for exported identifiers (types, functions, etc.)
- Use `ID` for identifiers instead of `Id`
- Use descriptive names over abbreviations when possible
- Avoid single letter variable names except for loop counters

### Error Handling
- Handle errors immediately after they occur
- Log errors appropriately using the standard log package
- Return errors with context when possible
- Use `errors.New()` or `fmt.Errorf()` for creating error messages
- Prefer returning errors over panicking unless it's a truly unrecoverable state

### Documentation
- Add comments to exported functions, types, and methods
- Use godoc-style comments (e.g., `// FunctionName does X`)
- Document public APIs clearly
- Use inline comments for complex logic or non-obvious implementation details

### Code Organization
- Keep functions small and focused on a single responsibility
- Group related functions together in the same file when appropriate
- Use constants for values that should not change
- Avoid deeply nested code structures
- Prefer early returns over else clauses where possible

### Go Specific Rules
- Use `var` keyword for variable declarations, not `:=` in global scope
- Use `context` package for request-scoped values and timeouts
- Use `gorm` for database operations with proper error handling
- Handle HTTP status codes correctly in API responses
- Use `http.HandlerFunc` for middleware functions

### Cursor/Copilot Rules
No specific Cursor or Copilot rules found in this repository.