package urlbuilder_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/mallardduck/teapot-router/pkg/teapot"
	"github.com/mallardduck/teapot-router/pkg/urlbuilder"
)

// Example of using URLBuilder with teapot router for S3 API responses
func Example() {
	// Create URL builder with canonical domain (for production)
	// Use empty string "" for development to build from request
	urls := urlbuilder.New("s3.example.com")

	r := teapot.New()

	// S3 ListBuckets endpoint
	r.QueryGET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buckets := []string{"photos", "documents", "backups"}

		// Generate URLs for each bucket
		var bucketURLs []string
		for _, bucket := range buckets {
			bucketURLs = append(bucketURLs, urls.BucketURL(r, bucket))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"buckets": bucketURLs,
		})
	})).Query("list-type")

	// S3 ListObjects endpoint
	r.QueryGET("/{bucket}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := teapot.URLParam(r, "bucket")
		objects := []string{"file1.txt", "file2.jpg", "folder/file3.pdf"}

		// Generate URLs for each object
		var objectURLs []string
		for _, key := range objects {
			objectURLs = append(objectURLs, urls.ObjectURL(r, bucket, key))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bucket":  urls.BucketURL(r, bucket),
			"objects": objectURLs,
		})
	})).Query("list-type")

	// Test the ListBuckets endpoint
	req := httptest.NewRequest("GET", "/?list-type", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"buckets":["https://s3.example.com/photos","https://s3.example.com/documents","https://s3.example.com/backups"]}
}

// Example using URLBuilder with no canonical domain (development mode)
func ExampleBuilder_development() {
	// No canonical domain - URLs built from request
	urls := urlbuilder.New("")

	r := teapot.New()

	r.Func().GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
		bucket := teapot.URLParam(r, "bucket")
		key := teapot.URLParam(r, "key")

		// URL will be built from request (http://localhost:9000/...)
		objectURL := urls.ObjectURL(r, bucket, key)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"url": objectURL,
		})
	})

	// Test with local development server
	req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt", nil)
	req.Host = "localhost:9000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"url":"http://localhost:9000/mybucket/path/to/file.txt"}
}

// Example showing scheme detection with reverse proxy
func ExampleBuilder_reverseProxy() {
	urls := urlbuilder.New("")

	r := teapot.New()

	r.Func().GET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		bucket := teapot.URLParam(r, "bucket")

		// URL builder detects HTTPS from X-Forwarded-Proto header
		bucketURL := urls.BucketURL(r, bucket)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"bucket": bucketURL,
		})
	})

	// Simulate reverse proxy setting X-Forwarded-Proto
	req := httptest.NewRequest("GET", "/mybucket", nil)
	req.Host = "s3.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"bucket":"https://s3.example.com/mybucket"}
}
