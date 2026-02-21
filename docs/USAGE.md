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

## Route Groups

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

