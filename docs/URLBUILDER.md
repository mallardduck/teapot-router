# URLBuilder Package

## Overview

The `urlbuilder` package provides a clean, composable way to generate full URLs for S3 API responses. It handles scheme detection, canonical domain configuration, and works seamlessly with teapot's named routes.

## Installation

```bash
go get github.com/mallardduck/teapot-router/pkg/urlbuilder
```

## Quick Start

```go
import "github.com/mallardduck/teapot-router/pkg/urlbuilder"

// Production mode (with canonical domain)
urls := urlbuilder.New("s3.example.com")
bucketURL := urls.BucketURL(r, "mybucket")
// Returns: https://s3.example.com/mybucket

// Development mode (from request)
urls := urlbuilder.New("")
bucketURL := urls.BucketURL(r, "mybucket")
// Returns: http://localhost:9000/mybucket
```

## API Methods

### `BucketURL(r *http.Request, bucket string) string`

Generates a URL for S3 bucket operations.

```go
url := urls.BucketURL(r, "photos")
// https://s3.example.com/photos
```

### `ObjectURL(r *http.Request, bucket, key string) string`

Generates a URL for S3 object operations.

```go
url := urls.ObjectURL(r, "photos", "2024/vacation.jpg")
// https://s3.example.com/photos/2024/vacation.jpg
```

### `BuildURL(r *http.Request, path string) string`

Generates a full URL from any path. **This is the bridge method** for combining with named routes.

```go
// Get path from named route
path := router.MustURL("bucket.show", "bucket", "mybucket")

// Convert to full URL
fullURL := urls.BuildURL(r, path)
// https://s3.example.com/mybucket
```

## Integration with Named Routes

The `BuildURL` method bridges teapot's named routes with full URL generation:

### Basic Usage

```go
router := teapot.New()
urls := urlbuilder.New("s3.example.com")

router.GET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
    bucket := teapot.URLParam(r, "bucket")

    // Generate URL using named route
    path := router.MustURL("bucket.show", "bucket", bucket)
    fullURL := urls.BuildURL(r, path)

    json.NewEncoder(w).Encode(map[string]string{
        "url": fullURL, // https://s3.example.com/mybucket
    })
}).Name("bucket.show")
```

### Helper Function Pattern

```go
// Create reusable helper
buildFullURL := func(req *http.Request, name string, params ...string) string {
    path := router.MustURL(name, params...)
    return urls.BuildURL(req, path)
}

// Use in handlers
router.GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
    bucket := teapot.URLParam(r, "bucket")
    key := teapot.URLParam(r, "key")

    response := map[string]string{
        "object": buildFullURL(r, "object.get", "bucket", bucket, "key", key),
        "bucket": buildFullURL(r, "bucket.show", "bucket", bucket),
    }

    json.NewEncoder(w).Encode(response)
}).Name("object.get")
```

### With Resource Scaffolding

```go
router := teapot.New()
urls := urlbuilder.New("s3.example.com")

router.Resource("buckets", "/buckets", "bucket", teapot.ResourceHandlers{
    Index: func(w http.ResponseWriter, r *http.Request) {
        // Generate URLs for related resources
        indexPath, _ := router.URL("buckets.index")
        bucket1Path := router.MustURL("buckets.show", "bucket", "photos")
        bucket2Path := router.MustURL("buckets.show", "bucket", "documents")

        response := map[string]interface{}{
            "self": urls.BuildURL(r, indexPath),
            "buckets": []string{
                urls.BuildURL(r, bucket1Path),
                urls.BuildURL(r, bucket2Path),
            },
        }

        json.NewEncoder(w).Encode(response)
    },
})
```

## Scheme Detection

The URL builder uses intelligent scheme detection with the following priority:

1. **Canonical Domain** → Always HTTPS
2. **X-Forwarded-Proto Header** → Trust reverse proxy
3. **TLS Connection State** → Detect direct HTTPS
4. **Default** → HTTP

### Behind Reverse Proxy

```go
// nginx sets X-Forwarded-Proto: https
// Builder automatically detects HTTPS
urls := urlbuilder.New("")
url := urls.BucketURL(r, "bucket")
// https://example.com/bucket (from header)
```

### Direct TLS

```go
// Request came over HTTPS (r.TLS != nil)
urls := urlbuilder.New("")
url := urls.BucketURL(r, "bucket")
// https://example.com/bucket (from TLS state)
```

## Configuration Patterns

### Environment-Based

```go
func newURLBuilder() *urlbuilder.Builder {
    domain := os.Getenv("S3_CANONICAL_DOMAIN")
    return urlbuilder.New(domain)
}

// Development: S3_CANONICAL_DOMAIN=""
// Production:  S3_CANONICAL_DOMAIN="s3.example.com"
```

### Multi-Environment

```go
var urls *urlbuilder.Builder

switch os.Getenv("ENV") {
case "production":
    urls = urlbuilder.New("s3.example.com")
case "staging":
    urls = urlbuilder.New("s3-staging.example.com")
default:
    urls = urlbuilder.New("") // Development
}
```

## S3 XML Response Example

```go
router.QueryGET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
    bucket := teapot.URLParam(r, "bucket")

    // Generate URLs for XML response
    objects := []struct {
        Key string
        URL string
    }{
        {Key: "file1.txt", URL: urls.ObjectURL(r, bucket, "file1.txt")},
        {Key: "file2.jpg", URL: urls.ObjectURL(r, bucket, "file2.jpg")},
    }

    // S3 XML response with full URLs
    w.Header().Set("Content-Type", "application/xml")
    fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
    <Name>%s</Name>
    <Contents>
        <Key>%s</Key>
        <URL>%s</URL>
    </Contents>
    <Contents>
        <Key>%s</Key>
        <URL>%s</URL>
    </Contents>
</ListBucketResult>`,
        bucket,
        objects[0].Key, objects[0].URL,
        objects[1].Key, objects[1].URL)
}).Query("list-type").Action("s3:ListBucket")
```

## Best Practices

### 1. Initialize Once

```go
// Initialize at app startup
var urls = urlbuilder.New(os.Getenv("S3_CANONICAL_DOMAIN"))

// Reuse across handlers
func listBuckets(w http.ResponseWriter, r *http.Request) {
    url := urls.BucketURL(r, "mybucket")
    // ...
}
```

### 2. Combine with Named Routes

```go
// Don't hardcode paths
❌ fullURL := urls.BuildURL(r, "/buckets/"+bucket)

// Use named routes
✅ path := router.MustURL("bucket.show", "bucket", bucket)
   fullURL := urls.BuildURL(r, path)
```

### 3. Use Canonical Domain in Production

```go
// Development: auto-detect from request
urls := urlbuilder.New("")

// Production: fixed canonical domain
urls := urlbuilder.New("s3.production.com")
```

## Testing

```bash
go test ./pkg/urlbuilder/
```

**77 tests** covering:
- Bucket/Object URL generation
- Canonical domain override
- Scheme detection (TLS, X-Forwarded-Proto, default)
- Port preservation
- BuildURL with named routes
- Integration examples

## Package Design

The `urlbuilder` package is intentionally **decoupled** from the `teapot` router:

- ✅ Can be used with any router (Chi, Gorilla, stdlib, etc.)
- ✅ No circular dependencies
- ✅ Composable with teapot's named routes via `BuildURL()`
- ✅ Standalone utility for S3 API development

This design allows you to:
- Use urlbuilder without teapot
- Use teapot without urlbuilder
- Compose them together for maximum convenience

## See Also

- [Main README](../README.md) - Teapot router overview
- [Quick Wins Added](../QUICK_WINS_ADDED.md) - All features added
- [Package Documentation](https://pkg.go.dev/github.com/mallardduck/teapot-router/pkg/urlbuilder)
