# S3 Router Package Summary

## Goal
Build a Go HTTP router package on top of Chi that adds S3-specific features:

1. **Named Routes** (Laravel-style) - Route names for URL generation
2. **Query Parameter Multiplexing** - Same path+method routes to different handlers based on query params (e.g., `GET /{bucket}?acl` vs `GET /{bucket}?versioning`)
3. **S3 Action Context** - Inject S3 action names into request context for authorization/logging

## Desired API Syntax

```go
r := s3router.New()

// Basic route with name and S3 action
r.GET("/{bucket}", handlers.ListObjects).
    Name("bucket.list").
    Action("s3:ListBucket")

// Query parameter multiplexing (key S3 feature)
r.GET("/{bucket}", handlers.GetBucketAcl).
    Name("bucket.acl").
    Action("s3:GetBucketAcl").
    Query("acl")  // Matches when ?acl is present

r.GET("/{bucket}", handlers.GetBucketVersioning).
    Name("bucket.versioning").
    Action("s3:GetBucketVersioning").
    Query("versioning")

// Named groups (adds name prefix)
r.NamedGroup("/{bucket}", "bucket", func(r *s3router.Router) {
    r.GET("", h).Name("list")  // becomes "bucket.list"
    r.GET("", h).Name("acl").Query("acl")  // becomes "bucket.acl"
})

// Route-specific middleware
r.GET("/private", h).Name("private").With(authMiddleware)

// URL generation
url := r.MustURL("bucket.list", "bucket", "mybucket")  // "/mybucket"

// Access in handlers
func Handler(w http.ResponseWriter, r *http.Request) {
    action := s3router.GetAction(r)      // "s3:GetBucketAcl"
    name := s3router.GetRouteName(r)     // "bucket.acl"
    bucket := s3router.URLParam(r, "bucket")
}
```

## Key Design Decisions

1. **Fluent Builder Pattern** - `r.GET(...).Name(...).Action(...).Query(...)`
2. **Query Matching Priority** - More specific matchers win (2 query params beats 1)
3. **Query existence vs value** - `.Query("acl")` = key exists, `.QueryValue("type", "full")` = exact match
4. **Same name, different methods allowed** - Like Laravel resources

## Files Created

- `/home/claude/s3router/DESIGN.md` - Full API design doc
- `/home/claude/s3router/router_test.go` - Comprehensive test suite (TDD)
- `/home/claude/s3router/router.go` - Implementation (partially complete, has bugs)
- `/home/claude/s3router/go.mod` - Module file

## Known Issues in Current Implementation

1. **Lazy registration conflicts with Chi groups** - The approach of collecting routes then registering later doesn't work well with Chi's middleware group isolation
2. **Need immediate registration** - Routes should register with Chi immediately, but we need a "dispatcher" pattern for query multiplexing

## Recommended Approach

Use a **dispatcher per method+pattern** that:
1. Registers immediately with Chi when first route for that method+pattern is created
2. Collects all routes with same method+pattern into a dispatcher
3. Dispatcher sorts by query specificity and handles routing at request time

```go
// Each unique method+pattern gets one dispatcher registered with Chi
type dispatcher struct {
    routes []*route  // sorted by query matcher count (most specific first)
}

func (d *dispatcher) ServeHTTP(w, r) {
    for _, rt := range d.routes {
        if matchesQuery(r, rt.QueryMatchers) {
            // inject context, apply middleware, call handler
            return
        }
    }
    http.NotFound(w, r)
}
```

## Test Coverage

The test file covers:
- Basic HTTP methods (GET, POST, PUT, DELETE, HEAD)
- URL parameters and wildcards
- S3 action/name context injection
- Query multiplexing (basic, multiple params, priority, value matching)
- Route groups and named groups
- URL generation
- Middleware (global, route-specific, group)
- S3-specific scenarios (bucket ops, object ops)
- Edge cases (empty names, trailing slashes, etc.)

## Dependencies

- `github.com/go-chi/chi/v5` - Base router