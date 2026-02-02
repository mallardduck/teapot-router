# Usage of `teapot-router`

## Named Routes

Define routes with names and generate URLs later:

```go
r.GET("/objects/{key}", showObject).Name("objects.show")

url := r.MustURL("objects.show", "key", "photos/avatar.png")
// Returns: "/objects/photos/avatar.png"

// Or with error handling:
url, err := r.URL("objects.show", "key", "photos/avatar.png")
```

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

Finalize is optional but recommended — routes work without it, just with slightly more overhead.

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

