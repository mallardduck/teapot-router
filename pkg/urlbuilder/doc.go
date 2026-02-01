// Package urlbuilder provides a simple, configurable URL builder for S3 API responses.
//
// # Features
//
//   - Canonical Domain Support - Configure a canonical domain for production URLs
//   - Automatic Scheme Detection - Detects HTTPS from TLS state or X-Forwarded-Proto header
//   - Reverse Proxy Aware - Works behind nginx, Caddy, or other reverse proxies
//   - Port Preservation - Maintains non-standard ports from request
//   - Development Friendly - Works without configuration using request host
//
// # Quick Start
//
// Production (with canonical domain):
//
//	import "github.com/mallardduck/teapot-router/pkg/urlbuilder"
//
//	// Initialize with your production domain
//	urls := urlbuilder.New("s3.example.com")
//
//	// Generate bucket URL
//	bucketURL := urls.BucketURL(r, "mybucket")
//	// Returns: https://s3.example.com/mybucket
//
//	// Generate object URL
//	objectURL := urls.ObjectURL(r, "mybucket", "path/to/file.txt")
//	// Returns: https://s3.example.com/mybucket/path/to/file.txt
//
// Development (from request):
//
//	// No canonical domain - builds from request
//	urls := urlbuilder.New("")
//
//	// With request to http://localhost:9000/mybucket
//	bucketURL := urls.BucketURL(r, "mybucket")
//	// Returns: http://localhost:9000/mybucket
//
// # Scheme Detection
//
// The URL builder uses the following priority for determining the scheme (http/https):
//
//  1. Canonical Domain - If configured, always uses https
//  2. X-Forwarded-Proto Header - Trusts reverse proxy header
//  3. TLS State - Checks if request came over TLS
//  4. Default - Falls back to http
//
// # Bridge with Named Routes
//
// The BuildURL method provides seamless integration with teapot's named routes:
//
//	router := teapot.New()
//	urls := urlbuilder.New("s3.example.com")
//
//	router.GET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
//	    bucket := teapot.URLParam(r, "bucket")
//
//	    // Combine named route + URL builder
//	    path := router.MustURL("bucket.show", "bucket", bucket)
//	    fullURL := urls.BuildURL(r, path)
//	    // Returns: https://s3.example.com/mybucket
//	}).Name("bucket.show")
//
// # Environment-Based Configuration
//
//	func newURLBuilder() *urlbuilder.Builder {
//	    domain := os.Getenv("S3_CANONICAL_DOMAIN")
//	    return urlbuilder.New(domain)
//	}
//
//	// Development: S3_CANONICAL_DOMAIN=""
//	// Production:  S3_CANONICAL_DOMAIN="s3.example.com"
//
// For complete documentation and examples, see:
// https://github.com/mallardduck/teapot-router/blob/main/docs/URLBUILDER.md
package urlbuilder
