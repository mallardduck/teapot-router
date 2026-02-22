# Usage of `teapot-router`

## Named Routes

Assign a name to any route with `.Name()`, then generate its URL path later
using `URL()` or `MustURL()`.

### Standard parameters

```go
r.GET("/users/{id}", showUser).Name("users.show")

// MustURL panics on error — suited to handler code
path := r.MustURL("users.show", "id", "42")
// Returns: "/users/42"

// URL returns an error — suited to startup / config code
path, err := r.URL("users.show", "id", "42")
```

Parameters are passed as alternating key-value pairs. Each key must match a
placeholder name in the route pattern.

### Wildcard parameters

Wildcard segments (`{key:.*}`, which match slashes) are substituted the same way:

```go
r.GET("/{bucket}/{key:.*}", getObject).Name("object.get")

path := r.MustURL("object.get", "bucket", "photos", "key", "2024/vacation.jpg")
// Returns: "/photos/2024/vacation.jpg"
```

### Error conditions

`URL()` returns an error when:
- the route name was never registered (check that `.Name()` was called)
- an odd number of `params` arguments is provided (must be key-value pairs)
- any placeholder remains unreplaced after substitution (a required param was omitted)

### Generating absolute URLs

Combine with the `urlbuilder` package to turn a path into a full URL:

```go
import "github.com/mallardduck/teapot-router/pkg/urlbuilder"

urls := urlbuilder.New("s3.example.com")

path := r.MustURL("object.get", "bucket", "photos", "key", "2024/vacation.jpg")
fullURL := urls.BuildURL(r, path)
// Returns: "https://s3.example.com/photos/2024/vacation.jpg"
```

See [URLBUILDER.md](URLBUILDER.md) for the full guide.

---

## Query-Based Routing

Route requests to different handlers based on query parameters — essential for S3-style APIs:

```go
// Same path, different handlers based on query params
r.GET("/{bucket}", listObjects).
Name("bucket.list").
Action("s3:ListBucket")

r.GET("/{bucket}", getBucketAcl).
Name("bucket.acl").
Action("s3:GetBucketAcl").
Query("acl") // Matches when ?acl is present

r.GET("/{bucket}", getBucketVersioning).
Name("bucket.versioning").
Action("s3:GetBucketVersioning").
Query("versioning") // Matches when ?versioning is present
```

Query matching options:

- `.Query("acl")` — matches if query param exists (any value)
- `.QueryValue("type", "full")` — matches if query param has exact value

More specific matchers take priority (2 query params beats 1).

---

## Grouped Dispatch

For paths with many query-parameter variants, `Dispatch` groups them into a
single block — clearer than scattering individual calls across many lines:

```go
r.Dispatch("GET", "/{bucket}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
    d.Default(listObjects).Name("bucket.list").Action("s3:ListBucket")
    d.When(m.QueryEquals("list-type", "2")).Do(listObjectsV2).Name("bucket.list-v2").Action("s3:ListObjectsV2")
    d.When(m.QueryExists("acl")).Do(getBucketAcl).Name("bucket.acl").Action("s3:GetBucketAcl")
    d.When(m.QueryExists("versioning")).Do(getVersioning).Name("bucket.versioning").Action("s3:GetBucketVersioning")
})
```

- `Default(handler)` — the fallback, matches when no other route's conditions match
- `When(matchers...).Do(handler)` — a conditional route; all matchers must match (AND)
- `.Name()`, `.Action()`, `.With()` — same fluent chain as the scattered API, available on both `Default` and `When` routes

The `m` parameter (type `teapot.Matchers`) exposes all built-in matcher constructors
so that the `dispatch` package does not need to be imported separately:

- `m.QueryExists("key")` — matches if the query param is present (any value)
- `m.QueryEquals("key", "value")` — matches if the query param equals a specific value
- `m.HeaderExists("key")` — matches if the header is present with a non-empty value
- `m.HeaderEquals("key", "value")` — matches if the header equals a specific value
- Multiple matchers in one `When` are ANDed: `When(m.QueryExists("partNumber"), m.QueryExists("uploadId"))`

Both styles coexist in the same router — use `Dispatch` where you have a dense
cluster of variants on one path, and the fluent style elsewhere.

### Router-Agnostic Dispatcher

The `dispatch` package works independently of teapot with any Go HTTP router:

```go
import "github.com/mallardduck/teapot-router/pkg/dispatch"

d := dispatch.New(func(b *dispatch.Builder) {
    b.Default(listHandler)
    b.When(dispatch.QueryEquals("format", "xml")).Do(xmlHandler)
    b.When(dispatch.QueryExists("search")).Do(searchHandler)
})

// d implements http.Handler — works with stdlib, chi, gorilla, or anything else
http.Handle("/api/items", d)
```

Same dispatching logic that `r.Dispatch` uses internally, without the
teapot-specific features (named routes, action context, URL generation).

---

## S3 Action Context

Each route can define an S3 action that's injected into the request context:

```go
r.GET("/{bucket}/{key:.*}", getObject).
Name("object.get").
Action("s3:GetObject")

func getObject(w http.ResponseWriter, r *http.Request) {
action := teapot.GetAction(r) // "s3:GetObject"
name := teapot.GetRouteName(r) // "object.get"
bucket := teapot.URLParam(r, "bucket")
key := teapot.URLParam(r, "key")

// Use action for authorization, logging, metrics...
}
```

---

## Route Groups and Sub-Routers

### Path and Name Groups

Group routes with path and name prefixes:

```go
// Path prefix only
r.Group("/api/v1", func (r *teapot.Router) {
    r.GET("/users", listUsers).Name("users.list")
})

// Path + name prefix
r.NamedGroup("/{bucket}", "bucket", func (r *teapot.Router) {
    r.GET("", listObjects).Name("list") // name: "bucket.list"
    r.GET("", getBucketAcl).Name("acl").Query("acl") // name: "bucket.acl"

    r.NamedGroup("/{key:.*}", "object", func (r *teapot.Router) {
        r.GET("", getObject).Name("get") // name: "bucket.object.get"
    })
})
```

### Sub-Routers (Live Propagation)

`SubRouter(prefix)` creates a new child router whose routes are automatically visible in the parent router with the prefix prepended. 

Unlike `Group()`, `SubRouter()` returns a separate `Router` instance that can be used independently (e.g., as its own HTTP server) while still reporting its routes to the parent for unified listing and URL generation.

```go
adminRouter := r.SubRouter("/admin")
adminRouter.GET("/dashboard", dashHandler).Name("dashboard")
// "dashboard" route is now visible in both adminRouter and r (as "/admin/dashboard")
```

---

## Mounting Handlers

### Mounting standard http.Handlers

Use `Mount(prefix, handler)` to attach any `http.Handler` to a path prefix:

```go
r.Mount("/static", http.FileServer(http.Dir("./static")))
```

### Mounting teapot Routers (Late Propagation & Homing)

If the handler being mounted is also a `*teapot.Router`, all its existing routes are automatically propagated to the parent router with the prefix prepended. 

Furthermore, **"Homing"** is established: any future routes added to the sub-router after it has been mounted will also be automatically propagated to the parent. This enables flexible router composition where routers can be developed independently or even after mounting.

```go
apiRouter := teapot.New()
apiRouter.GET("/users", listUsers).Name("users.list")

// Mount propagates all apiRouter's current routes to r
r.Mount("/api/v1", apiRouter)

// Homing: even routes added LATER are visible in the parent
apiRouter.GET("/profile", profileHandler).Name("users.profile")

// r now knows about both:
// "/api/v1/users" (name: "users.list")
// "/api/v1/profile" (name: "users.profile")
```

#### Custom Name Prefixes for Mounted Routers

When mounting a router, you can control the name prefix applied to its routes using `MountNamed` or the fluent `.Name()` method on the mount builder:

```go
// Using MountNamed
r.MountNamed("/api/v1", "v1", apiRouter)
// apiRouter route "users.list" becomes "v1.users.list" in r

// Using fluent API
r.Mount("/api/v2", apiRouter).Name("v2")
// apiRouter route "users.list" becomes "v2.users.list" in r
```

If a name is assigned via the fluent `.Name()` after mounting, any already-propagated routes will be renamed throughout the hierarchy automatically.

---

## Direct Handler Registration

Assign standard `http.Handler` (including `http.HandlerFunc` via `http.HandlerFunc(fn)`) to a method and pattern:

```go
r.Handle("GET", "/health", healthHandler).Name("health")
```

This is a more flexible alternative to `GET()`, `POST()`, etc. when you already have an `http.Handler` instance.

---

## Phantom Routes (External Services)

Use `RegisterExternal(method, pattern, name, action)` to add "phantom" routes to the router. These routes appear in documentation (route lists) and work with the URL builder, but do not dispatch requests locally. This is useful for documenting external services or third-party handlers that you cannot convert to teapot.

```go
r.RegisterExternal("GET", "https://auth.example.com/login", "auth.login", "auth:Login")

// Use for URL generation:
loginURL := r.MustURL("auth.login") // "https://auth.example.com/login"
```

---

## Middleware

Works with all standard `chi` middleware:

```go
// Global middleware
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

// Route-specific middleware
r.GET("/admin", adminHandler).
Name("admin").
With(authMiddleware)

// Group middleware
r.Group("/api", func (r *teapot.Router) {
r.Use(apiKeyMiddleware)
r.GET("/data", dataHandler).Name("api.data")
})
```

---

## Route Introspection

List all registered routes programmatically or via HTTP:

```go
// Programmatic access
for _, route := range r.Routes() {
fmt.Printf("%s %s -> %s (%s)\n",
route.Method,
route.Pattern,
route.Name,
route.Action)
}

// HTTP debug endpoint (development only)
if debug {
r.RegisterDebugRoute("/.internal/routes", "debug.routes")
}
// Visit http://localhost:8080/.internal/routes for JSON or HTML route listing
```

See [ROUTES-LISTING.md](ROUTES-LISTING.md) for CLI helpers and formatters.

---

## Performance

For production deployments, call `Finalize()` before serving to optimize route handlers:

```go
r := teapot.New()

// Register all routes
r.GET("/users", listUsers).Name("users.list")
r.POST("/users", createUser).Name("users.create")

// Optimize for production
r.Finalize()

http.ListenAndServe(":8080", r)
```

Finalize is optional but recommended — it pre-computes handler chains for direct
routes and eagerly builds all dispatchers, avoiding any per-request setup on the
first call.

---

## Complete S3-Style Example

```go
r := teapot.New()

// Service endpoint
r.GET("/", listBuckets).Name("service.list").Action("s3:ListAllMyBuckets")

// Bucket operations with query multiplexing
r.NamedGroup("/{bucket}", "bucket", func (r *teapot.Router) {
r.PUT("", createBucket).Name("create").Action("s3:CreateBucket")
r.DELETE("", deleteBucket).Name("delete").Action("s3:DeleteBucket")
r.HEAD("", headBucket).Name("head").Action("s3:HeadBucket")
r.GET("", listObjects).Name("list").Action("s3:ListBucket")

// Query-based bucket operations
r.GET("", getBucketAcl).Name("acl.get").Action("s3:GetBucketAcl").Query("acl")
r.PUT("", putBucketAcl).Name("acl.put").Action("s3:PutBucketAcl").Query("acl")
r.GET("", listObjectVersions).Name("versions").Action("s3:ListBucketVersions").Query("versions")

// Object operations
r.NamedGroup("/{key:.*}", "object", func (r *teapot.Router) {
r.GET("", getObject).Name("get").Action("s3:GetObject")
r.PUT("", putObject).Name("put").Action("s3:PutObject")
r.DELETE("", deleteObject).Name("delete").Action("s3:DeleteObject")
r.HEAD("", headObject).Name("head").Action("s3:HeadObject")

r.GET("", getObjectAcl).Name("acl.get").Action("s3:GetObjectAcl").Query("acl")
r.POST("", createMultipartUpload).Name("upload.create").Action("s3:CreateMultipartUpload").Query("uploads")
})
})

http.ListenAndServe(":8080", r)
```

The GET bucket operations above could equivalently use `Dispatch` to group all
the query variants explicitly:

```go
// Inside the NamedGroup("/{bucket}", ...) callback:
r.Dispatch("GET", "", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
    d.Default(listObjects).Name("list").Action("s3:ListBucket")
    d.When(m.QueryExists("acl")).Do(getBucketAcl).Name("acl.get").Action("s3:GetBucketAcl")
    d.When(m.QueryExists("versions")).Do(listObjectVersions).Name("versions").Action("s3:ListBucketVersions")
})
```

---

## Additional Features

**RESTful Resource Scaffolding:**

```go
r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
Index:   listPhotos,  // GET    /photos
Store:   createPhoto, // POST   /photos
Show:    showPhoto,   // GET    /photos/{photo}
Update:  updatePhoto, // PUT    /photos/{photo}
Destroy: deletePhoto, // DELETE /photos/{photo}
})
```

**URL Builder Package:**

For generating full URLs in responses (especially useful for S3 APIs):

```go
import "github.com/mallardduck/teapot-router/pkg/urlbuilder"

urls := urlbuilder.New("s3.example.com")
fullURL := urls.ObjectURL(r, "bucket", "path/to/file.txt")
// Returns: https://s3.example.com/bucket/path/to/file.txt
```

See [URLBUILDER.md](URLBUILDER.md) for complete guide.

