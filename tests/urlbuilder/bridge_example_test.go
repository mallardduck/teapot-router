package urlbuilder_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
	"github.com/mallardduck/teapot-router/pkg/urlbuilder"
)

// bucketsController implements the bucket actions for the examples below.
type bucketsController struct {
	router *teapot.Router
	urls   *urlbuilder.Builder
}

func (c *bucketsController) Index(w http.ResponseWriter, r *http.Request) {
	// Generate URLs for response using named routes
	indexPath, _ := c.router.URL("buckets.index")                       // "/buckets"
	bucket1Path := c.router.MustURL("buckets.show", "bucket", "photos") // "/buckets/photos"
	bucket2Path := c.router.MustURL("buckets.show", "bucket", "docs")   // "/buckets/docs"

	// Convert paths to full URLs
	response := map[string]any{
		"self": c.urls.BuildURL(r, indexPath),
		"buckets": []string{
			c.urls.BuildURL(r, bucket1Path),
			c.urls.BuildURL(r, bucket2Path),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (c *bucketsController) Show(w http.ResponseWriter, r *http.Request) {
	testutil.NoopResponse(w, r)
}

// Example showing how to combine teapot's named routes with URL builder
func ExampleBuilder_BuildURL() {
	// Create router with named routes
	router := teapot.New()
	urls := urlbuilder.New("s3.example.com")

	// Register S3 bucket routes
	router.Resource("buckets", &bucketsController{router: router, urls: urls}, teapot.Param("bucket"))

	// Test the endpoint
	req := httptest.NewRequest("GET", "/buckets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"buckets":["https://s3.example.com/buckets/photos","https://s3.example.com/buckets/docs"],"self":"https://s3.example.com/buckets"}
}

// Example showing a helper function that combines router + urlbuilder
func ExampleBuilder_BuildURL_helper() {
	router := teapot.New()
	urls := urlbuilder.New("s3.example.com")

	// Helper function to build full URL from named route
	buildFullURL := func(req *http.Request, name string, params ...string) string {
		path := router.MustURL(name, params...)
		return urls.BuildURL(req, path)
	}

	router.GET("/{bucket}/{key:.*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use helper to generate URLs
		selfURL := buildFullURL(r, "object.get",
			"bucket", teapot.URLParam(r, "bucket"),
			"key", teapot.URLParam(r, "key"))

		response := map[string]string{
			"url": selfURL,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})).Name("object.get")

	req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"url":"https://s3.example.com/mybucket/path/to/file.txt"}
}

// Example showing how to build related resource URLs
func ExampleBuilder_BuildURL_relatedResources() {
	router := teapot.New()
	urls := urlbuilder.New("s3.example.com")

	// Object routes
	router.GET("/{bucket}/{key:.*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := teapot.URLParam(r, "bucket")
		key := teapot.URLParam(r, "key")

		// Generate related URLs
		selfPath := router.MustURL("object.get", "bucket", bucket, "key", key)
		bucketPath := router.MustURL("bucket.show", "bucket", bucket)

		response := map[string]any{
			"self":   urls.BuildURL(r, selfPath),
			"bucket": urls.BuildURL(r, bucketPath),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})).Name("object.get")

	// Bucket routes
	router.GET("/{bucket}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler implementation
	})).Name("bucket.show")

	req := httptest.NewRequest("GET", "/photos/2024/vacation.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"bucket":"https://s3.example.com/photos","self":"https://s3.example.com/photos/2024/vacation.jpg"}
}
