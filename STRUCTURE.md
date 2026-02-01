# Project Structure

## Directory Layout

```
teapot-router/
├── pkg/
│   └── teapot/           # Public API
│       ├── router.go              # Main router implementation
│       └── router_test.go         # Unit tests
├── internal/
│   └── core/             # Private implementation
│       ├── context.go             # Context value helpers
│       ├── dispatcher.go          # Query multiplexing dispatcher
│       ├── query_matchers.go      # Query parameter matchers
│       └── route.go               # Route metadata and pattern translation
├── tests/                # Integration tests
│   ├── s3_example_test.go         # Complete S3 API example
│   └── chi_wildcard_test.go       # Chi wildcard behavior tests
├── docs/                 # Documentation
│   └── s3-api-router-info.md
├── go.mod
├── go.sum
├── PLAN.md              # Implementation plan
├── README.md            # User-facing documentation
└── STRUCTURE.md         # This file
```

## Package Organization

### `pkg/teapot/` - Public API
The main public interface that users import:

```go
import "github.com/mallardduck/teapot-router/pkg/teapot"
```

**Exports:**
- `Router` - Main router type
- `RouteBuilder` - Fluent route builder
- `RouteInfo` - Route metadata
- `New()` - Create new router
- `GetAction(r)` - Get S3 action from request
- `GetRouteName(r)` - Get route name from request
- `URLParam(r, key)` - Get URL parameter

### `internal/core/` - Private Implementation
Internal implementation details not exposed to users:

**Files:**
- `context.go` - Context key definitions and helpers
- `dispatcher.go` - HTTP dispatcher for query multiplexing
- `query_matchers.go` - Query parameter matching logic
- `route.go` - Route metadata and pattern translation

**Key Types:**
- `Dispatcher` - Handles multiple routes with same method+pattern
- `Route` - Internal route representation
- `QueryMatcher` - Interface for query parameter matching
- `QueryExistsMatcher` - Matches query param existence
- `QueryValueMatcher` - Matches query param values

### `tests/` - Integration Tests
Full integration tests demonstrating complete usage:

- `s3_example_test.go` - Complete S3-compatible API implementation
- `chi_wildcard_test.go` - Chi router wildcard behavior reference

## Testing Strategy

### Unit Tests (`pkg/teapot/router_test.go`)
- Test individual router features in isolation
- Test edge cases and error conditions
- Use `package teapot_test` for black-box testing

### Integration Tests (`tests/`)
- Test complete workflows
- Demonstrate real-world usage patterns
- S3 API scenarios

## Key Design Decisions

### 1. Dispatcher Pattern
Each unique method+pattern combination gets one dispatcher registered with Chi:
- Dispatcher collects all routes for that method+pattern
- Routes sorted by query parameter specificity (most specific first)
- At request time, dispatcher finds first matching route based on query params

### 2. Pattern Translation
User-friendly `{key:.*}` syntax is translated to Chi's `/*` wildcard:
- Original pattern stored for URL generation
- Chi pattern used for actual routing
- Wildcard parameters remapped in dispatcher

### 3. Context Injection
S3 actions and route names injected into request context:
- Available via `GetAction(r)` and `GetRouteName(r)`
- Useful for authorization, logging, metrics

### 4. Query Multiplexing
Multiple routes can share the same method+pattern:
- Distinguished by query parameters
- More specific matchers (value > existence) win
- Multiple matchers = higher priority

## Import Paths

Users import only the public API:
```go
import "github.com/mallardduck/teapot-router/pkg/teapot"
```

Internal packages are not accessible outside the module.

## Running Tests

```bash
# All tests
go test ./...

# Package tests only (unit tests)
go test ./pkg/teapot/

# Integration tests only
go test ./tests/

# Verbose output
go test -v ./...

# Specific test
go test -run TestS3StyleAPI ./tests/
```

## Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```
