// Example CLI command demonstrating S3 API route implementation
//
// This example showcases a comprehensive S3-compatible API implementation using
// teapot-router, demonstrating:
//   - Path-based bucket routing (/{bucket} not subdomain-based)
//   - Query parameter-based route disambiguation (QueryGET, QueryPUT, etc.)
//   - Multiple HTTP methods on the same path pattern
//   - Route naming and S3 action tagging
//   - All tiers of S3 operations (Service, Bucket, Object, Multipart)
//
// Usage:
//
//	go run examples/routes-cli/main.go           # Table format
//	go run examples/routes-cli/main.go --json    # JSON format
//	go run examples/routes-cli/main.go --compact # Compact format
//
// The output formats are similar to Laravel's `php artisan route:list` command.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

var (
	jsonOutput    = flag.Bool("json", false, "Output routes as JSON")
	compactOutput = flag.Bool("compact", false, "Output routes in compact format")
)

func main() {
	flag.Parse()

	// Create router and register routes
	// In a real app, this would be your SetupRoutes() function
	router := setupRoutes()

	// Get all routes
	routes := router.Routes()

	// Format output based on flags
	var err error
	if *jsonOutput {
		err = teapot.FormatRoutesJSON(os.Stdout, routes)
	} else if *compactOutput {
		err = teapot.FormatRoutesCompact(os.Stdout, routes)
	} else {
		err = teapot.FormatRoutesTable(os.Stdout, routes)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting routes: %v\n", err)
		os.Exit(1)
	}
}

// setupRoutes demonstrates a comprehensive S3 API implementation
// This showcases the router's capabilities for handling complex APIs with:
//   - Multiple HTTP methods on same paths
//   - Query parameter-based routing
//   - Path parameters with wildcards
//   - Named routes and actions
func setupRoutes() *teapot.Router {
	router := teapot.New()

	// ==================== SERVICE-LEVEL OPERATIONS ====================
	// S3 service-level operations (no bucket in path)
	router.GET("/", listBuckets).Name("s3.service.list-buckets").Action("api:s3:ListBuckets")

	// ==================== BUCKET OPERATIONS ====================
	// Mix of direct routes (PUT, DELETE, HEAD, GET) and query-based routes (QueryGET, QueryPUT).
	// The router automatically promotes to dispatcher-based routing when needed.

	router.PUT("/{bucket}", createBucket).Name("s3.bucket.create").Action("api:s3:CreateBucket")
	router.DELETE("/{bucket}", deleteBucket).Name("s3.bucket.delete").Action("api:s3:DeleteBucket")
	router.HEAD("/{bucket}", headBucket).Name("s3.bucket.head").Action("api:s3:HeadBucket")
	router.GET("/{bucket}", listObjectsV1).Name("s3.bucket.list-objects-v1").Action("api:s3:ListObjects")

	// Query-based bucket operations
	// ListObjectsV2 (preferred over v1)
	router.QueryGET("/{bucket}", listObjectsV2).Query("list-type").Name("s3.bucket.list-objects-v2").Action("api:s3:ListObjectsV2")

	// Bucket configuration endpoints
	router.QueryGET("/{bucket}", getBucketLocation).Query("location").Name("s3.bucket.get-location").Action("api:s3:GetBucketLocation")
	router.QueryGET("/{bucket}", getBucketVersioning).Query("versioning").Name("s3.bucket.get-versioning").Action("api:s3:GetBucketVersioning")
	router.QueryPUT("/{bucket}", putBucketVersioning).Query("versioning").Name("s3.bucket.put-versioning").Action("api:s3:PutBucketVersioning")
	router.QueryGET("/{bucket}", getBucketAcl).Query("acl").Name("s3.bucket.get-acl").Action("api:s3:GetBucketAcl")
	router.QueryPUT("/{bucket}", putBucketAcl).Query("acl").Name("s3.bucket.put-acl").Action("api:s3:PutBucketAcl")

	// List object versions (for versioned buckets)
	router.QueryGET("/{bucket}", listObjectVersions).Query("versions").Name("s3.bucket.list-object-versions").Action("api:s3:ListObjectVersions")

	// List multipart uploads in bucket
	router.QueryGET("/{bucket}", listMultipartUploads).Query("uploads").Name("s3.bucket.list-multipart-uploads").Action("api:s3:ListMultipartUploads")

	// Bulk delete objects
	router.QueryPOST("/{bucket}", deleteObjects).Query("delete").Name("s3.bucket.delete-objects").Action("api:s3:DeleteObjects")

	// ==================== OBJECT OPERATIONS ====================
	// Direct routes for operations without query params
	router.GET("/{bucket}/{key:.*}", getObject).Name("s3.object.get").Action("api:s3:GetObject")
	router.PUT("/{bucket}/{key:.*}", putObject).Name("s3.object.put").Action("api:s3:PutObject")
	router.DELETE("/{bucket}/{key:.*}", deleteObject).Name("s3.object.delete").Action("api:s3:DeleteObject")
	router.HEAD("/{bucket}/{key:.*}", headObject).Name("s3.object.head").Action("api:s3:HeadObject")
	// Note: CopyObject uses PUT /{bucket}/{key} with x-amz-copy-source header.
	//       The putObject handler detects this header and can adjust action context
	//       for logging/metrics (e.g., override to "api:s3:CopyObject")

	// Query-based object operations
	router.QueryGET("/{bucket}/{key:.*}", getObjectAcl).Query("acl").Name("s3.object.get-acl").Action("api:s3:GetObjectAcl")
	router.QueryPUT("/{bucket}/{key:.*}", putObjectAcl).Query("acl").Name("s3.object.put-acl").Action("api:s3:PutObjectAcl")

	// ==================== MULTIPART UPLOAD OPERATIONS ====================
	// Initiate multipart upload
	router.QueryPOST("/{bucket}/{key:.*}", createMultipartUpload).Query("uploads").Name("s3.multipart.create").Action("api:s3:CreateMultipartUpload")

	// Upload part (requires both partNumber and uploadId query params)
	// Note: UploadPartCopy uses the same route with x-amz-copy-source header.
	//       The uploadPart handler detects this and can adjust action context accordingly.
	router.QueryPUT("/{bucket}/{key:.*}", uploadPart).Query("partNumber").Query("uploadId").Name("s3.multipart.upload-part").Action("api:s3:UploadPart")

	// Complete multipart upload
	router.QueryPOST("/{bucket}/{key:.*}", completeMultipartUpload).Query("uploadId").Name("s3.multipart.complete").Action("api:s3:CompleteMultipartUpload")

	// Abort multipart upload
	router.QueryDELETE("/{bucket}/{key:.*}", abortMultipartUpload).Query("uploadId").Name("s3.multipart.abort").Action("api:s3:AbortMultipartUpload")

	// List parts of a multipart upload
	router.QueryGET("/{bucket}/{key:.*}", listParts).Query("uploadId").Name("s3.multipart.list-parts").Action("api:s3:ListParts")

	// ==================== DEBUG ROUTES ====================
	// Debug route (conditionally registered)
	if isDebug() {
		router.GET("/.internal/routes", teapot.NewListRoutesHandler(router, nil)).Name("debug.routes")
	}

	router.GET("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {}).Name("favicon")

	return router
}

// Mock S3 API handlers
// Service-level
func listBuckets(_ http.ResponseWriter, _ *http.Request) {}

// Bucket operations
func createBucket(_ http.ResponseWriter, _ *http.Request)         {}
func deleteBucket(_ http.ResponseWriter, _ *http.Request)         {}
func headBucket(_ http.ResponseWriter, _ *http.Request)           {}
func listObjectsV1(_ http.ResponseWriter, _ *http.Request)        {}
func listObjectsV2(_ http.ResponseWriter, _ *http.Request)        {}
func getBucketLocation(_ http.ResponseWriter, _ *http.Request)    {}
func getBucketVersioning(_ http.ResponseWriter, _ *http.Request)  {}
func putBucketVersioning(_ http.ResponseWriter, _ *http.Request)  {}
func getBucketAcl(_ http.ResponseWriter, _ *http.Request)         {}
func putBucketAcl(_ http.ResponseWriter, _ *http.Request)         {}
func listObjectVersions(_ http.ResponseWriter, _ *http.Request)   {}
func listMultipartUploads(_ http.ResponseWriter, _ *http.Request) {}
func deleteObjects(_ http.ResponseWriter, _ *http.Request)        {}

// Object operations
func getObject(_ http.ResponseWriter, _ *http.Request)    {}
func putObject(_ http.ResponseWriter, _ *http.Request)    {}
func deleteObject(_ http.ResponseWriter, _ *http.Request) {}
func headObject(_ http.ResponseWriter, _ *http.Request)   {}
func getObjectAcl(_ http.ResponseWriter, _ *http.Request) {}
func putObjectAcl(_ http.ResponseWriter, _ *http.Request) {}

// Multipart upload operations
func createMultipartUpload(_ http.ResponseWriter, _ *http.Request)   {}
func uploadPart(_ http.ResponseWriter, _ *http.Request)              {}
func completeMultipartUpload(_ http.ResponseWriter, _ *http.Request) {}
func abortMultipartUpload(_ http.ResponseWriter, _ *http.Request)    {}
func listParts(_ http.ResponseWriter, _ *http.Request)               {}

func isDebug() bool { return os.Getenv("DEBUG") == "true" }
