## Routes Listing

Teapot provides built-in support for viewing all registered routes, both via HTTP endpoints (for debugging) and CLI
commands (like Laravel's `php artisan route:list`).

## HTTP Debug Endpoint

### Quick Start

```go
router := teapot.New()

// Register your routes
router.GET("/api/users", listUsers).Name("users.index")
router.POST("/api/users", createUser).Name("users.store")

// Conditionally register debug route
if debug {
router.RegisterDebugRoute("/.internal/routes", "debug.routes")
}

http.ListenAndServe(":8080", router)
```

Visit `http://localhost:8080/.internal/routes` to see all routes.

### Aggregated Route List

If you have multiple routers (e.g., an app router on port 80 and an admin router on port 8080), you can merge their routes for a unified listing:

```go
appRouter := setupApp()
adminRouter := setupAdmin()

// Aggregate routes from both
combined := teapot.AggregateRoutes(appRouter, adminRouter)

// Create a handler for the aggregated routes
handler := teapot.NewListRoutesHandlerWithRoutes(combined, nil)
appRouter.GET("/.internal/routes", handler).Name("all.routes")
```

### Response Formats

The debug endpoint supports both JSON and HTML:

**JSON** (with `Accept: application/json` header):

```json
{
  "count": 3,
  "routes": [
    {
      "Method": "GET",
      "Pattern": "/api/users",
      "Name": "users.index",
      "Action": ""
    },
    {
      "Method": "POST",
      "Pattern": "/api/users",
      "Name": "users.store",
      "Action": ""
    }
  ]
}
```

**HTML** (default for browsers):

- Clean table layout
- Color-coded HTTP methods
- Sortable by method/pattern/name
- Shows route actions for S3 routes

### Advanced Usage

```go
// Option 1: Use convenience method
router.RegisterDebugRoute("/.internal/routes", "debug.routes")

// Option 2: Manual registration (more control)
handler := router.RoutesHandler()
router.GET("/.internal/routes", handler).Name("debug.routes")

// Option 3: Environment-based
if os.Getenv("DEBUG") == "true" {
router.RegisterDebugRoute("/.internal/routes", "debug.routes")
}

// Option 4: With middleware
router.GET("/.internal/routes", router.RoutesHandler()).
Name("debug.routes").
With(authMiddleware)

// Option 5: Custom filtering
// Only show routes that don't start with /.internal/
filter := func(route teapot.RouteInfo) bool {
    return !strings.HasPrefix(route.Pattern, "/.internal/")
}
router.GET("/routes", teapot.NewListRoutesHandler(router, filter))
```

## CLI Route Listing

For Laravel-style CLI commands, teapot provides formatting helpers.

### Example CLI Command

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var jsonOutput = flag.Bool("json", false, "Output routes as JSON")

func main() {
	flag.Parse()

	// Create router and register routes
	router := setupRoutes()

	// Get all routes
	routes := router.Routes()

	// Format output
	var err error
	if *jsonOutput {
		err = teapot.FormatRoutesJSON(os.Stdout, routes)
	} else {
		err = teapot.FormatRoutesTable(os.Stdout, routes)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupRoutes() *teapot.Router {
	router := teapot.New()

	// Register your routes here
	router.GET("/api/users", listUsers).Name("users.index")
	router.POST("/api/users", createUser).Name("users.store")

	return router
}
```

### Table Output

```bash
$ go run cmd/routes/main.go

METHOD  PATTERN             NAME               ACTION
------  -------             ----               ------
GET     /                   home               -
GET     /api/users          api.users.index    -
POST    /api/users          api.users.store    -
DELETE  /api/users/{id}     api.users.destroy  -
GET     /api/users/{id}     api.users.show     -
PUT     /api/users/{id}     api.users.update   -
GET     /{bucket}           s3.objects.list    s3:ListObjects
PUT     /{bucket}           s3.buckets.create  s3:CreateBucket
```

### JSON Output

```bash
$ go run cmd/routes/main.go --json

{
  "count": 8,
  "routes": [
    {
      "Method": "GET",
      "Pattern": "/",
      "Name": "home",
      "Action": ""
    },
    {
      "Method": "GET",
      "Pattern": "/api/users",
      "Name": "api.users.index",
      "Action": ""
    }
  ]
}
```

### Compact Output

```bash
$ go run cmd/routes/main.go --compact

GET     /                                        home
GET     /api/users                               api.users.index
POST    /api/users                               api.users.store
DELETE  /api/users/{id}                          api.users.destroy
GET     /api/users/{id}                          api.users.show
PUT     /api/users/{id}                          api.users.update
```

## Formatting Helpers

### FormatRoutesTable

Pretty table output for CLI commands:

```go
routes := router.Routes()
err := teapot.FormatRoutesTable(os.Stdout, routes)
```

### AggregateRoutes

Merge routes from multiple routers for unified output:

```go
allRoutes := teapot.AggregateRoutes(mainRouter, adminRouter, internalRouter)
err := teapot.FormatRoutesTable(os.Stdout, allRoutes)
```

### FormatRoutesJSON

JSON output with count and sorted routes:

```go
routes := router.Routes()
err := teapot.FormatRoutesJSON(os.Stdout, routes)
```

### FormatRoutesCompact

Compact one-line-per-route format:

```go
routes := router.Routes()
err := teapot.FormatRoutesCompact(os.Stdout, routes)
```

## Integration with Cobra

```go
package cmd

import (
	"fmt"
	"os"
	"github.com/mallardduck/teapot-router/pkg/teapot"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List all registered HTTP routes",
	RunE:  runRoutes,
}

func init() {
	rootCmd.AddCommand(routesCmd)
	routesCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
}

func runRoutes(cmd *cobra.Command, args []string) error {
	// Create router without starting server
	router := setupRoutes()
	routes := router.Routes()

	if jsonOutput {
		return teapot.FormatRoutesJSON(os.Stdout, routes)
	}
	return teapot.FormatRoutesTable(os.Stdout, routes)
}

func setupRoutes() *teapot.Router {
	router := teapot.New()
	// Register routes (same as server)
	registerAPIRoutes(router)
	registerWebRoutes(router)
	return router
}
```

## Filtering and Searching

You can filter routes before formatting:

```go
routes := router.Routes()

// Filter by method
var getRoutes []teapot.RouteInfo
for _, r := range routes {
if r.Method == "GET" {
getRoutes = append(getRoutes, r)
}
}
teapot.FormatRoutesTable(os.Stdout, getRoutes)

// Filter by pattern
var apiRoutes []teapot.RouteInfo
for _, r := range routes {
if strings.HasPrefix(r.Pattern, "/api/") {
apiRoutes = append(apiRoutes, r)
}
}

// Filter by action (S3 routes)
var s3Routes []teapot.RouteInfo
for _, r := range routes {
if r.Action != "" {
s3Routes = append(s3Routes, r)
}
}
```

## Sorting

All formatting helpers automatically sort routes:

1. By pattern (alphabetically)
2. By method (alphabetically) for same pattern

You can also sort manually:

```go
import "sort"

routes := router.Routes()

// Sort by name
sort.Slice(routes, func (i, j int) bool {
return routes[i].Name < routes[j].Name
})

// Sort by method only
sort.Slice(routes, func (i, j int) bool {
return routes[i].Method < routes[j].Method
})
```

## Best Practices

### 1. Conditional Debug Routes

Only register debug routes in development:

```go
if os.Getenv("APP_ENV") != "production" {
router.RegisterDebugRoute("/.internal/routes", "debug.routes")
}
```

### 2. Protect with Middleware

Add authentication for production debug endpoints:

```go
if debug {
router.GET("/.internal/routes", router.RoutesHandler()).
Name("debug.routes").
With(adminAuth)
}
```

### 3. CLI for CI/CD

Use the CLI command in CI/CD to verify routes:

```bash
# Check route count
ROUTE_COUNT=$(go run cmd/routes/main.go --json | jq '.count')
if [ "$ROUTE_COUNT" -lt 10 ]; then
    echo "Error: Expected at least 10 routes"
    exit 1
fi

# Check for required routes
go run cmd/routes/main.go --json | jq '.routes[].Name' | grep -q "api.users.index"
```

### 4. Listing Routes Without Real Handler Dependencies

In larger applications, handlers are often constructed with live dependencies (database
connections, service clients, config, etc.) that are unavailable or expensive to build in a
standalone CLI tool. The recommended pattern is to extract route registration into a shared
function that accepts a handler interface, then provide a stub implementation backed by
`teapot.NoopHandler` when listing:

```go
// Handlers defines the contract your route registration depends on.
type Handlers interface {
    ListUsers() http.Handler
    ShowUser()  http.Handler
    CreateUser() http.Handler
}

// RegisterRoutes is the single source of truth for all routes.
// It is used by both the real server and the listing tool.
func RegisterRoutes(r *teapot.Router, h Handlers) {
    r.GET("/users",       h.ListUsers()).Name("users.index")
    r.GET("/users/{id}", h.ShowUser()).Name("users.show")
    r.POST("/users",      h.CreateUser()).Name("users.store")
}

// Real implementation — constructed with live dependencies in main.go.
type appHandlers struct{ db *sql.DB }
func (h *appHandlers) ListUsers()  http.Handler { return NewListUsersHandler(h.db) }
func (h *appHandlers) ShowUser()   http.Handler { return NewShowUserHandler(h.db) }
func (h *appHandlers) CreateUser() http.Handler { return NewCreateUserHandler(h.db) }

// Stub implementation — no dependencies required.
type stubHandlers struct{}
func (stubHandlers) ListUsers()  http.Handler { return teapot.NoopHandler }
func (stubHandlers) ShowUser()   http.Handler { return teapot.NoopHandler }
func (stubHandlers) CreateUser() http.Handler { return teapot.NoopHandler }

// cmd/routes/main.go — prints routes with zero real dependencies.
func main() {
    r := teapot.New()
    RegisterRoutes(r, stubHandlers{})
    teapot.FormatRoutesTable(os.Stdout, r.Routes())
}
```

Because the stub satisfies the same interface as the real implementation, adding a new route
to `RegisterRoutes` will cause a compile error until `stubHandlers` is updated — keeping
the listing tool automatically in sync.

### 5. Documentation Generation

Generate route documentation from routes:

```go
routes := router.Routes()
f, _ := os.Create("docs/routes.md")
defer f.Close()

fmt.Fprintln(f, "# API Routes\n")
for _, r := range routes {
if strings.HasPrefix(r.Pattern, "/api/") {
fmt.Fprintf(f, "## %s %s\n", r.Method, r.Pattern)
fmt.Fprintf(f, "Name: `%s`\n\n", r.Name)
}
}
```

## Complete Example

See [examples/routes-cli/main.go](../../examples/routes-cli/main.go) for a complete working example.

## API Reference

### Router Methods

- `Routes() []RouteInfo` - Get all registered routes
- `RoutesHandler() http.HandlerFunc` - Get HTTP handler for debug endpoint
- `RegisterDebugRoute(path, name string) *RouteBuilder` - Convenience method

### Formatting Functions

- `FormatRoutesJSON(w io.Writer, routes []RouteInfo) error`
- `FormatRoutesTable(w io.Writer, routes []RouteInfo) error`
- `FormatRoutesCompact(w io.Writer, routes []RouteInfo) error`

### RouteInfo Struct

```go
type RouteInfo struct {
Method  string // HTTP method (GET, POST, etc.)
Pattern string // URL pattern (/users/{id})
Name    string // Route name (users.show)
Action  string // S3 action (s3:GetObject) or empty
}
```
