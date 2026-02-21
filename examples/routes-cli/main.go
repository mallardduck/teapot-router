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
	// Could also do:
	// funcRouter := router.Func()

	// ==================== SERVICE-LEVEL OPERATIONS ====================
	// S3 service-level operations (no bucket in path)
	router.Func().GET("/", listBuckets).Name("s3.service.list-buckets").Action("api:s3:ListBuckets")

	// ==================== BUCKET OPERATIONS ====================
	// Mix of direct routes (PUT, DELETE, HEAD, GET) and query-based routes (QueryGET, QueryPUT).
	// The router automatically promotes to dispatcher-based routing when needed.

	router.Func().PUT("/{bucket}", createBucket).Name("s3.bucket.create").Action("api:s3:CreateBucket")
	router.Func().DELETE("/{bucket}", deleteBucket).Name("s3.bucket.delete").Action("api:s3:DeleteBucket")
	router.Func().HEAD("/{bucket}", headBucket).Name("s3.bucket.head").Action("api:s3:HeadBucket")
	router.Func().GET("/{bucket}", listObjectsV1).Name("s3.bucket.list-objects-v1").Action("api:s3:ListObjects")

	// Query-based bucket operations
	// ListObjectsV2 (preferred over v1)
	router.Func().QueryGET("/{bucket}", listObjectsV2).QueryValue("list-type", "2").Name("s3.bucket.list-objects-v2").Action("api:s3:ListObjectsV2")

	// Bucket configuration endpoints
	router.Func().QueryGET("/{bucket}", getBucketLocation).Query("location").Name("s3.bucket.get-location").Action("api:s3:GetBucketLocation")
	router.Func().QueryGET("/{bucket}", getBucketVersioning).Query("versioning").Name("s3.bucket.get-versioning").Action("api:s3:GetBucketVersioning")
	router.Func().QueryPUT("/{bucket}", putBucketVersioning).Query("versioning").Name("s3.bucket.put-versioning").Action("api:s3:PutBucketVersioning")
	router.Func().QueryGET("/{bucket}", getBucketAcl).Query("acl").Name("s3.bucket.get-acl").Action("api:s3:GetBucketAcl")
	router.Func().QueryPUT("/{bucket}", putBucketAcl).Query("acl").Name("s3.bucket.put-acl").Action("api:s3:PutBucketAcl")

	// Bucket policy endpoints
	router.Func().QueryGET("/{bucket}", getBucketPolicy).Query("policy").Name("s3.bucket.get-policy").Action("api:s3:GetBucketPolicy")
	router.Func().QueryPUT("/{bucket}", putBucketPolicy).Query("policy").Name("s3.bucket.put-policy").Action("api:s3:PutBucketPolicy")
	router.Func().QueryDELETE("/{bucket}", deleteBucketPolicy).Query("policy").Name("s3.bucket.delete-policy").Action("api:s3:DeleteBucketPolicy")

	// Bucket CORS endpoints
	router.Func().QueryGET("/{bucket}", getBucketCors).Query("cors").Name("s3.bucket.get-cors").Action("api:s3:GetBucketCors")
	router.Func().QueryPUT("/{bucket}", putBucketCors).Query("cors").Name("s3.bucket.put-cors").Action("api:s3:PutBucketCors")

	// Bucket lifecycle configuration
	// Note: Legacy GetBucketLifecycle/PutBucketLifecycle share the same path and query param
	//       as the modern *Configuration variants; one route per method covers both.
	router.Func().QueryGET("/{bucket}", getBucketLifecycleConfiguration).Query("lifecycle").Name("s3.bucket.get-lifecycle-configuration").Action("api:s3:GetBucketLifecycleConfiguration")
	router.Func().QueryPUT("/{bucket}", putBucketLifecycleConfiguration).Query("lifecycle").Name("s3.bucket.put-lifecycle-configuration").Action("api:s3:PutBucketLifecycleConfiguration")

	// Public access block
	router.Func().QueryGET("/{bucket}", getPublicAccessBlock).Query("publicAccessBlock").Name("s3.bucket.get-public-access-block").Action("api:s3:GetPublicAccessBlock")
	router.Func().QueryPUT("/{bucket}", putPublicAccessBlock).Query("publicAccessBlock").Name("s3.bucket.put-public-access-block").Action("api:s3:PutPublicAccessBlock")

	// Object lock configuration
	router.Func().QueryPUT("/{bucket}", putObjectLockConfiguration).Query("object-lock").Name("s3.bucket.put-object-lock-configuration").Action("api:s3:PutObjectLockConfiguration")

	// Logging, events, payment, and analytics
	router.Func().QueryPUT("/{bucket}", putBucketLogging).Query("logging").Name("s3.bucket.put-logging").Action("api:s3:PutBucketLogging")
	router.Func().QueryPUT("/{bucket}", putBucketNotification).Query("notification").Name("s3.bucket.put-notification").Action("api:s3:PutBucketNotification")
	router.Func().QueryGET("/{bucket}", getBucketRequestPayment).Query("requestPayment").Name("s3.bucket.get-request-payment").Action("api:s3:GetBucketRequestPayment")
	router.Func().QueryGET("/{bucket}", getBucketAnalyticsConfiguration).Query("analytics").Name("s3.bucket.get-analytics-configuration").Action("api:s3:GetBucketAnalyticsConfiguration")

	// List object versions (for versioned buckets)
	router.Func().QueryGET("/{bucket}", listObjectVersions).Query("versions").Name("s3.bucket.list-object-versions").Action("api:s3:ListObjectVersions")

	// List multipart uploads in bucket
	router.Func().QueryGET("/{bucket}", listMultipartUploads).Query("uploads").Name("s3.bucket.list-multipart-uploads").Action("api:s3:ListMultipartUploads")

	// Bulk delete objects
	router.Func().QueryPOST("/{bucket}", deleteObjects).Query("delete").Name("s3.bucket.delete-objects").Action("api:s3:DeleteObjects")

	// ==================== OBJECT OPERATIONS ====================
	router.Func().GET("/{bucket}/{key:.*}", getObject).Name("s3.object.get").Action("api:s3:GetObject")
	router.Func().DELETE("/{bucket}/{key:.*}", deleteObject).Name("s3.object.delete").Action("api:s3:DeleteObject")
	router.Func().HEAD("/{bucket}/{key:.*}", headObject).Name("s3.object.head").Action("api:s3:HeadObject")

	// PUT /{bucket}/{key} dispatches on X-Amz-Copy-Source header.
	// UploadPart / UploadPartCopy also live here: same method+path, and header
	// presence distinguishes the copy variant.  The remaining QueryPUT routes
	// below (acl, tagging, …) are added to this same dispatcher automatically.
	router.Dispatch("PUT", "/{bucket}/{key:.*}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
		d.FuncDefault(putObject).Name("s3.object.put").Action("api:s3:PutObject")
		d.When(m.HeaderExists("X-Amz-Copy-Source")).FuncDo(copyObject).Name("s3.object.copy").Action("api:s3:CopyObject")

		d.When(m.QueryExists("partNumber"), m.QueryExists("uploadId")).FuncDo(uploadPart).Name("s3.multipart.upload-part").Action("api:s3:UploadPart")
		d.When(m.QueryExists("partNumber"), m.QueryExists("uploadId"), m.HeaderExists("X-Amz-Copy-Source")).FuncDo(uploadPartCopy).Name("s3.multipart.upload-part-copy").Action("api:s3:UploadPartCopy")
	})

	// Query-based object operations
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectAcl).Query("acl").Name("s3.object.get-acl").Action("api:s3:GetObjectAcl")
	router.Func().QueryPUT("/{bucket}/{key:.*}", putObjectAcl).Query("acl").Name("s3.object.put-acl").Action("api:s3:PutObjectAcl")

	// Object tagging
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectTagging).Query("tagging").Name("s3.object.get-tagging").Action("api:s3:GetObjectTagging")
	router.Func().QueryPUT("/{bucket}/{key:.*}", putObjectTagging).Query("tagging").Name("s3.object.put-tagging").Action("api:s3:PutObjectTagging")

	// Object legal hold and retention (compliance)
	// Note: PutObjectRetention appears under both "Object" and "Locking" scopes in the
	//       S3 API docs; it is a single route here, as is PutObjectLegalHold.
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectLegalHold).Query("legal-hold").Name("s3.object.get-legal-hold").Action("api:s3:GetObjectLegalHold")
	router.Func().QueryPUT("/{bucket}/{key:.*}", putObjectLegalHold).Query("legal-hold").Name("s3.object.put-legal-hold").Action("api:s3:PutObjectLegalHold")
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectRetention).Query("retention").Name("s3.object.get-retention").Action("api:s3:GetObjectRetention")
	router.Func().QueryPUT("/{bucket}/{key:.*}", putObjectRetention).Query("retention").Name("s3.object.put-retention").Action("api:s3:PutObjectRetention")

	// Object attributes and torrent (legacy)
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectAttributes).Query("attributes").Name("s3.object.get-attributes").Action("api:s3:GetObjectAttributes")
	router.Func().QueryGET("/{bucket}/{key:.*}", getObjectTorrent).Query("torrent").Name("s3.object.get-torrent").Action("api:s3:GetObjectTorrent")

	// ==================== MULTIPART UPLOAD OPERATIONS ====================
	// Initiate multipart upload
	router.Func().QueryPOST("/{bucket}/{key:.*}", createMultipartUpload).Query("uploads").Name("s3.multipart.create").Action("api:s3:CreateMultipartUpload")

	// UploadPart / UploadPartCopy are in the PUT dispatch block above.

	// Complete multipart upload
	router.Func().QueryPOST("/{bucket}/{key:.*}", completeMultipartUpload).Query("uploadId").Name("s3.multipart.complete").Action("api:s3:CompleteMultipartUpload")

	// Abort multipart upload
	router.Func().QueryDELETE("/{bucket}/{key:.*}", abortMultipartUpload).Query("uploadId").Name("s3.multipart.abort").Action("api:s3:AbortMultipartUpload")

	// List parts of a multipart upload
	router.Func().QueryGET("/{bucket}/{key:.*}", listParts).Query("uploadId").Name("s3.multipart.list-parts").Action("api:s3:ListParts")

	// ==================== DEBUG ROUTES ====================
	// Debug route (conditionally registered)
	if isDebug() {
		router.GET("/.internal/routes", teapot.NewListRoutesHandler(router, nil)).Name("debug.routes")
	}

	// TODO add a handler to output the ico file
	router.Func().GET("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {}).Name("favicon")

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

// Bucket configuration and management
func getBucketPolicy(_ http.ResponseWriter, _ *http.Request)                 {}
func putBucketPolicy(_ http.ResponseWriter, _ *http.Request)                 {}
func deleteBucketPolicy(_ http.ResponseWriter, _ *http.Request)              {}
func getBucketCors(_ http.ResponseWriter, _ *http.Request)                   {}
func putBucketCors(_ http.ResponseWriter, _ *http.Request)                   {}
func getBucketLifecycleConfiguration(_ http.ResponseWriter, _ *http.Request) {}
func putBucketLifecycleConfiguration(_ http.ResponseWriter, _ *http.Request) {}
func getPublicAccessBlock(_ http.ResponseWriter, _ *http.Request)            {}
func putPublicAccessBlock(_ http.ResponseWriter, _ *http.Request)            {}
func putObjectLockConfiguration(_ http.ResponseWriter, _ *http.Request)      {}
func putBucketLogging(_ http.ResponseWriter, _ *http.Request)                {}
func putBucketNotification(_ http.ResponseWriter, _ *http.Request)           {}
func getBucketRequestPayment(_ http.ResponseWriter, _ *http.Request)         {}
func getBucketAnalyticsConfiguration(_ http.ResponseWriter, _ *http.Request) {}

// Object operations
func getObject(_ http.ResponseWriter, _ *http.Request)    {}
func putObject(_ http.ResponseWriter, _ *http.Request)    {}
func copyObject(_ http.ResponseWriter, _ *http.Request)   {}
func deleteObject(_ http.ResponseWriter, _ *http.Request) {}
func headObject(_ http.ResponseWriter, _ *http.Request)   {}
func getObjectAcl(_ http.ResponseWriter, _ *http.Request) {}
func putObjectAcl(_ http.ResponseWriter, _ *http.Request) {}

// Object tagging, compliance, and metadata
func getObjectTagging(_ http.ResponseWriter, _ *http.Request)    {}
func putObjectTagging(_ http.ResponseWriter, _ *http.Request)    {}
func getObjectLegalHold(_ http.ResponseWriter, _ *http.Request)  {}
func putObjectLegalHold(_ http.ResponseWriter, _ *http.Request)  {}
func getObjectRetention(_ http.ResponseWriter, _ *http.Request)  {}
func putObjectRetention(_ http.ResponseWriter, _ *http.Request)  {}
func getObjectAttributes(_ http.ResponseWriter, _ *http.Request) {}
func getObjectTorrent(_ http.ResponseWriter, _ *http.Request)    {}

// Multipart upload operations
func createMultipartUpload(_ http.ResponseWriter, _ *http.Request)   {}
func uploadPart(_ http.ResponseWriter, _ *http.Request)              {}
func uploadPartCopy(_ http.ResponseWriter, _ *http.Request)          {}
func completeMultipartUpload(_ http.ResponseWriter, _ *http.Request) {}
func abortMultipartUpload(_ http.ResponseWriter, _ *http.Request)    {}
func listParts(_ http.ResponseWriter, _ *http.Request)               {}

func isDebug() bool { return os.Getenv("DEBUG") == "true" }
