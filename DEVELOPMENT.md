# Development Guide

This guide covers everything you need to know to develop and contribute to teapot-router.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Benchmarking](#benchmarking)
- [Code Organization](#code-organization)
- [Common Tasks](#common-tasks)
- [Contributing](#contributing)

## Prerequisites

- Go 1.25 or later
- Git
- A code editor (GoLand, VS Code, or similar)

## Getting Started

1. Clone the repository:

```bash
git clone https://github.com/mallardduck/teapot-router.git
cd teapot-router
```

2. Install dependencies:

```bash
go mod download
```

3. Verify the setup by running tests:

```bash
go test ./...
```

4. Run benchmarks to verify performance:

```bash
go test -bench=. ./tests/teapot/
```

## Project Structure

```
teapot-router/
├── pkg/                      # Public packages (importable by users)
│   ├── teapot/              # Main router package
│   │   ├── router.go                 # Router implementation
│   │   ├── router_test.go            # Unit tests
│   │   ├── routes_helpers.go         # Route listing/formatting helpers
│   │   ├── routes_test.go            # Route introspection tests
│   │   ├── resource_test.go          # Resource routing tests
│   │   ├── helpers_test.go           # Helper function tests
│   │   ├── validation_test.go        # Route validation tests
│   │   └── optimized_handler.go      # Performance-optimized handlers
│   └── urlbuilder/          # URL builder utility
│       ├── doc.go                    # Package documentation
│       ├── urlbuilder.go             # URL builder implementation
│       └── urlbuilder_test.go        # URL builder tests
├── internal/                # Private packages (internal use only)
│   └── core/               # Core implementation details
│       ├── context.go               # Context key definitions
│       ├── dispatcher.go            # Query multiplexing dispatcher
│       ├── query_matchers.go        # Query parameter matchers
│       └── route.go                 # Route metadata & pattern translation
├── tests/                   # Integration and benchmark tests
│   ├── s3_example_test.go          # S3 API integration test
│   ├── chi_wildcard_test.go        # Chi wildcard behavior reference
│   ├── finalize_example_test.go    # Finalized examples
│   ├── teapot/                     # Teapot-specific tests
│   │   ├── finalize_bench_test.go  # Finalized benchmarks
│   │   ├── minimal_bench_test.go   # Minimal benchmarks
│   │   └── router_bench_test.go    # Router benchmarks
│   └── urlbuilder/                 # URL builder tests
├── examples/                # Example applications
│   └── routes-cli/         # CLI tool to list routes (like Laravel's route:list)
│       └── main.go
├── docs/                    # Documentation
│   ├── s3-api-router-info.md       # S3 API routing documentation
│   ├── URLBUILDER.md               # URL builder documentation
│   ├── ROUTES-LISTING.md           # Route listing documentation
│   ├── TEST-ORGANIZATION.md        # Test organization guide
│   ├── feature-comparison.md       # Feature comparison
│   ├── optimization-options.md     # Optimization strategies
│   ├── option-comparison.md        # Option comparisons
│   └── option-2a-results.md        # Results from option 2a
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── README.md               # User-facing documentation
├── PLAN.md                 # Implementation plan
├── STRUCTURE.md            # Project structure documentation
├── SUMMARY.md              # Implementation summary
└── DEVELOPMENT.md          # This file
```

### Package Organization

**Public API (`pkg/`)**

- Users import these packages directly
- All exported types, functions, and methods must be well-documented
- API stability is critical once we reach v1.0

**Private Implementation (`internal/`)**

- Not accessible outside this module
- Can be refactored freely without breaking users
- Houses implementation details

**Integration Tests (`tests/`)**

- Test complete workflows and real-world scenarios
- Include S3 API examples and benchmarks
- Separate from unit tests for clearer organization

## Development Workflow

### Making Changes

1. Create a feature branch:

```bash
git checkout -b feature/your-feature-name
```

2. Make your changes following the code organization guidelines below

3. Write tests for new functionality:
    - Unit tests in the same package as the code (`pkg/teapot/*_test.go`)
    - Integration tests in `tests/` for complex scenarios

4. Run tests to verify:

```bash
go test ./...
```

5. Run benchmarks if you changed performance-critical code:

```bash
go test -bench=. ./tests/teapot/
```

6. Format and lint your code:

```bash
go fmt ./...
go vet ./...
```

7. Commit your changes with a clear message:

```bash
git commit -m "feat: add new feature description"
```

### Commit Message Format

Use conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `docs:` - Documentation changes
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

## Testing

### Test Organization

The project follows a clear separation between different test types:

**Unit Tests** (`pkg/teapot/*_test.go`, `pkg/urlbuilder/*_test.go`)

- Located next to source code in package directories
- Test individual functions/methods in isolation
- Fast execution, no external dependencies
- Use `package teapot_test` for black-box testing

**Integration Tests** (`tests/*_test.go`)

- Located in top-level `tests/` directory
- Test multiple components working together
- Realistic usage scenarios (S3 API workflows, etc.)
- Package-external testing (black-box)

**Benchmarks** (`tests/teapot/*_bench_test.go`)

- Performance measurements and comparisons
- Located in `tests/teapot/` subdirectory
- Run separately with `-bench` flag

**Examples** (`tests/urlbuilder/*_example_test.go`)

- Demonstrate API usage patterns
- Serve as executable documentation
- Located in `tests/*/` subdirectories

### Running Tests

```bash
# Run all tests
go test ./...

# Unit tests only
go test ./pkg/teapot/
go test ./pkg/urlbuilder/

# Integration tests only
go test ./tests/

# Run specific test
go test -run TestNamedRoutes ./pkg/teapot/

# Run tests verbosely
go test -v ./...
```

### Test Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# Coverage by package
go test ./pkg/teapot/ -cover
go test ./pkg/urlbuilder/ -cover

# Unit tests only
go test ./pkg/... -coverprofile=coverage-unit.out

# Integration tests only
go test ./tests/... -coverprofile=coverage-integration.out
```

**Coverage Goals:**

- Aim for >90% coverage on public API
- 100% coverage on critical paths (query multiplexing, route matching)
- Document any intentionally untested code

### Writing Tests

Example unit test structure:

```go
func TestFeatureName(t *testing.T) {
// Setup
r := teapot.New()

// Exercise
r.GET("/path", handler).Name("route.name")

// Verify
if url := r.MustURL("route.name"); url != "/path" {
t.Errorf("expected /path, got %s", url)
}
}
```

Example table-driven test:

```go
func TestQueryMatching(t *testing.T) {
tests := []struct {
name     string
query    string
expected string
}{
{"no query", "", "handler1"},
{"with acl", "?acl", "handler2"},
{"with versioning", "?versioning", "handler3"},
}

for _, tt := range tests {
t.Run(tt.name, func (t *testing.T) {
// test implementation
})
}
}
```

## Benchmarking

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./tests/teapot/

# Run specific benchmark
go test -bench=BenchmarkQueryMultiplexing ./tests/teapot/

# Run benchmarks with memory stats
go test -bench=. -benchmem ./tests/teapot/

# Run benchmarks multiple times for accuracy
go test -bench=. -count=5 ./tests/teapot/

# Compare benchmarks (save baseline first)
go test -bench=. ./tests/teapot/ > old.txt
# Make changes...
go test -bench=. ./tests/teapot/ > new.txt
benchcmp old.txt new.txt
```

### Writing Benchmarks

Benchmarks live in `tests/teapot/*_bench_test.go`:

```go
func BenchmarkFeatureName(b *testing.B) {
// Setup (not timed)
r := teapot.New()
r.GET("/users/{id}", handler).Name("users.show")

b.ResetTimer() // Start timing

for i := 0; i < b.N; i++ {
// Code to benchmark
r.MustURL("users.show", "id", "123")
}
}
```

### Performance Goals

- Route registration: O(1)
- Query matching: O(n) where n = routes with same method+pattern
- URL generation: O(1) hash map lookup
- Pattern translation: O(m) where m = pattern length (one-time)

Most S3 endpoints have 1-5 query variants, so query matching is effectively O(1) in practice.

## Code Organization

### Public API Guidelines

**Exported Types and Functions** (`pkg/teapot/`)

- Must have godoc comments
- Must maintain backward compatibility (after v1.0)
- Should be intuitive and follow Go conventions
- Keep the API surface minimal

**Internal Implementation** (`internal/core/`)

- Can change freely
- Focus on clarity and maintainability
- Comment complex logic thoroughly

### Code Style

Follow standard Go conventions:

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable names
- Keep functions focused and small
- Comment exported symbols with proper godoc

### Architecture Patterns

**Dispatcher Pattern**

- Each unique method+pattern gets one dispatcher
- Dispatcher collects routes and sorts by query specificity
- Matches at request time based on query parameters

**Pattern Translation**

- User-friendly `{key:.*}` syntax translates to Chi's `/*`
- Original pattern preserved for URL generation
- Wildcard values remapped to named parameters

**Context Injection**

- S3 actions and route names stored in request context
- Accessible via helper functions
- No pollution of URL parameters

## Common Tasks

### Adding a New Feature

1. Design the API (update PLAN.md if significant)
2. Add tests first (TDD approach)
3. Implement the feature
4. Update documentation (README.md, godoc comments)
5. Add integration test if needed
6. Run full test suite

### Adding Route Helpers

Route helpers live in `pkg/teapot/routes_helpers.go`:

- `FormatRoutesTable()` - Pretty table format
- `FormatRoutesJSON()` - JSON output
- `FormatRoutesCompact()` - Compact format

Add new formatters here and test in `routes_test.go`.

### Optimizing Performance

1. Write a benchmark for the current implementation
2. Profile to find bottlenecks:

```bash
go test -cpuprofile=cpu.prof -bench=. ./tests/teapot/
go tool pprof cpu.prof
```

3. Optimize the code
4. Run benchmarks to verify improvement
5. Ensure tests still pass

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/go-chi/chi/v5

# Verify tests still pass
go test ./...
```

### Generating Documentation

```bash
# View godoc locally
godoc -http=:6060
# Then visit http://localhost:6060/pkg/github.com/mallardduck/teapot-router/

# Generate package documentation
go doc -all pkg/teapot
```

### Running Examples

```bash
# Routes CLI example (like Laravel's route:list)
go run examples/routes-cli/main.go

# With JSON output
go run examples/routes-cli/main.go --json

# With compact output
go run examples/routes-cli/main.go --compact
```

### Debugging

Use standard Go debugging tools:

```bash
# Print debug output
go test -v ./pkg/teapot/

# Run with race detector
go test -race ./...

# Debug with delve
dlv test ./pkg/teapot/ -- -test.run TestSpecificTest
```

## Contributing

### Before Submitting a PR

1. Run the full test suite:

```bash
go test ./...
```

2. Run benchmarks for performance-critical changes:

```bash
go test -bench=. ./tests/teapot/
```

3. Format and vet your code:

```bash
go fmt ./...
go vet ./...
```

4. Update documentation:
    - Add godoc comments to exported symbols
    - Update README.md for user-facing changes
    - Update relevant docs in `docs/`

5. Write clear commit messages following the conventional commit format

### PR Guidelines

- Keep PRs focused on a single feature or fix
- Include tests for new functionality
- Update documentation
- Reference any related issues
- Add benchmarks for performance-sensitive code

### Code Review Process

- All code must pass CI checks (when set up)
- At least one approval required
- Address all review comments
- Squash commits before merging (if needed)

## Additional Resources

- [README.md](README.md) - User-facing documentation
- [STRUCTURE.md](STRUCTURE.md) - Detailed structure documentation
- [PLAN.md](PLAN.md) - Implementation plan and design decisions
- [docs/](docs/) - Additional documentation
    - [URLBUILDER.md](docs/URLBUILDER.md) - URL builder guide
    - [ROUTES-LISTING.md](docs/ROUTES-LISTING.md) - Route listing guide
    - [TEST-ORGANIZATION.md](docs/TEST-ORGANIZATION.md) - Test organization
    - [s3-api-router-info.md](docs/s3-api-router-info.md) - S3 routing info

## Getting Help

- Check existing [documentation](docs/)
- Look at [tests](tests/) for usage examples
- Review the [implementation plan](PLAN.md)
- Open an issue for bugs or feature requests

## License

MIT
