package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestS3StyleAPI demonstrates a complete S3-style API implementation
// This test comprehensively covers all S3 API routes matching examples/routes-cli/main.go
func TestS3StyleAPI(t *testing.T) {
	r := teapot.New()

	// Debug logging can be enabled with: r.SetDebugLog(true)

	// Generic handler that outputs route info for verification
	// Format: "ROUTE:<name>|ACTION:<action>"
	handler := func(name, action string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "ROUTE:%s|ACTION:%s", name, action)
		}
	}

	// ==================== SERVICE-LEVEL OPERATIONS ====================
	r.GET("/", handler("s3.service.list-buckets", "s3:ListBuckets")).
		Name("s3.service.list-buckets").
		Action("s3:ListBuckets")

	// ==================== BUCKET OPERATIONS ====================
	// Mix of direct routes and query-based routes - auto-promotion handles it

	r.PUT("/{bucket}", handler("s3.bucket.create", "s3:CreateBucket")).
		Name("s3.bucket.create").
		Action("s3:CreateBucket")

	r.DELETE("/{bucket}", handler("s3.bucket.delete", "s3:DeleteBucket")).
		Name("s3.bucket.delete").
		Action("s3:DeleteBucket")

	r.HEAD("/{bucket}", handler("s3.bucket.head", "s3:HeadBucket")).
		Name("s3.bucket.head").
		Action("s3:HeadBucket")

	r.GET("/{bucket}", handler("s3.bucket.list-objects-v1", "s3:ListBucket")).
		Name("s3.bucket.list-objects-v1").
		Action("s3:ListBucket")

	// Query-based bucket operations (will auto-promote the direct routes above)
	r.QueryGET("/{bucket}", handler("s3.bucket.list-objects-v2", "s3:ListBucket")).
		Query("list-type").
		Name("s3.bucket.list-objects-v2").
		Action("s3:ListBucket")

	r.QueryGET("/{bucket}", handler("s3.bucket.get-location", "s3:GetBucketLocation")).
		Query("location").
		Name("s3.bucket.get-location").
		Action("s3:GetBucketLocation")

	r.QueryGET("/{bucket}", handler("s3.bucket.get-versioning", "s3:GetBucketVersioning")).
		Query("versioning").
		Name("s3.bucket.get-versioning").
		Action("s3:GetBucketVersioning")

	r.QueryPUT("/{bucket}", handler("s3.bucket.put-versioning", "s3:PutBucketVersioning")).
		Query("versioning").
		Name("s3.bucket.put-versioning").
		Action("s3:PutBucketVersioning")

	r.QueryGET("/{bucket}", handler("s3.bucket.get-acl", "s3:GetBucketAcl")).
		Query("acl").
		Name("s3.bucket.get-acl").
		Action("s3:GetBucketAcl")

	r.QueryPUT("/{bucket}", handler("s3.bucket.put-acl", "s3:PutBucketAcl")).
		Query("acl").
		Name("s3.bucket.put-acl").
		Action("s3:PutBucketAcl")

	r.QueryGET("/{bucket}", handler("s3.bucket.list-object-versions", "s3:ListBucketVersions")).
		Query("versions").
		Name("s3.bucket.list-object-versions").
		Action("s3:ListBucketVersions")

	r.QueryGET("/{bucket}", handler("s3.bucket.list-multipart-uploads", "s3:ListBucketMultipartUploads")).
		Query("uploads").
		Name("s3.bucket.list-multipart-uploads").
		Action("s3:ListBucketMultipartUploads")

	r.QueryPOST("/{bucket}", handler("s3.bucket.delete-objects", "s3:DeleteObject")).
		Query("delete").
		Name("s3.bucket.delete-objects").
		Action("s3:DeleteObject")

	// ==================== OBJECT OPERATIONS ====================
	// Direct routes for simple operations
	r.GET("/{bucket}/{key:.*}", handler("s3.object.get", "s3:GetObject")).
		Name("s3.object.get").
		Action("s3:GetObject")

	r.PUT("/{bucket}/{key:.*}", handler("s3.object.put", "s3:PutObject")).
		Name("s3.object.put").
		Action("s3:PutObject")

	r.DELETE("/{bucket}/{key:.*}", handler("s3.object.delete", "s3:DeleteObject")).
		Name("s3.object.delete").
		Action("s3:DeleteObject")

	r.HEAD("/{bucket}/{key:.*}", handler("s3.object.head", "s3:GetObject")).
		Name("s3.object.head").
		Action("s3:GetObject")

	// Query-based object operations (will auto-promote the direct routes above)
	r.QueryGET("/{bucket}/{key:.*}", handler("s3.object.get-acl", "s3:GetObjectAcl")).
		Query("acl").
		Name("s3.object.get-acl").
		Action("s3:GetObjectAcl")

	r.QueryPUT("/{bucket}/{key:.*}", handler("s3.object.put-acl", "s3:PutObjectAcl")).
		Query("acl").
		Name("s3.object.put-acl").
		Action("s3:PutObjectAcl")

	// ==================== MULTIPART UPLOAD OPERATIONS ====================
	r.QueryPOST("/{bucket}/{key:.*}", handler("s3.multipart.create", "s3:PutObject")).
		Query("uploads").
		Name("s3.multipart.create").
		Action("s3:PutObject")

	r.QueryPUT("/{bucket}/{key:.*}", handler("s3.multipart.upload-part", "s3:PutObject")).
		Query("partNumber").Query("uploadId").
		Name("s3.multipart.upload-part").
		Action("s3:PutObject")

	r.QueryPOST("/{bucket}/{key:.*}", handler("s3.multipart.complete", "s3:PutObject")).
		Query("uploadId").
		Name("s3.multipart.complete").
		Action("s3:PutObject")

	r.QueryDELETE("/{bucket}/{key:.*}", handler("s3.multipart.abort", "s3:AbortMultipartUpload")).
		Query("uploadId").
		Name("s3.multipart.abort").
		Action("s3:AbortMultipartUpload")

	r.QueryGET("/{bucket}/{key:.*}", handler("s3.multipart.list-parts", "s3:ListMultipartUploadParts")).
		Query("uploadId").
		Name("s3.multipart.list-parts").
		Action("s3:ListMultipartUploadParts")

	// ==================== COMPREHENSIVE ROUTE TESTS ====================
	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		// Service-level
		{"ListBuckets", "GET", "/", "ROUTE:s3.service.list-buckets|ACTION:s3:ListBuckets"},

		// Bucket operations (no query params)
		{"CreateBucket", "PUT", "/mybucket", "ROUTE:s3.bucket.create|ACTION:s3:CreateBucket"},
		{"DeleteBucket", "DELETE", "/mybucket", "ROUTE:s3.bucket.delete|ACTION:s3:DeleteBucket"},
		{"HeadBucket", "HEAD", "/mybucket", "ROUTE:s3.bucket.head|ACTION:s3:HeadBucket"},
		{"ListObjectsV1", "GET", "/mybucket", "ROUTE:s3.bucket.list-objects-v1|ACTION:s3:ListBucket"},

		// Bucket operations (with query params)
		{"ListObjectsV2", "GET", "/mybucket?list-type=2", "ROUTE:s3.bucket.list-objects-v2|ACTION:s3:ListBucket"},
		{"GetBucketLocation", "GET", "/mybucket?location", "ROUTE:s3.bucket.get-location|ACTION:s3:GetBucketLocation"},
		{"GetBucketVersioning", "GET", "/mybucket?versioning", "ROUTE:s3.bucket.get-versioning|ACTION:s3:GetBucketVersioning"},
		{"PutBucketVersioning", "PUT", "/mybucket?versioning", "ROUTE:s3.bucket.put-versioning|ACTION:s3:PutBucketVersioning"},
		{"GetBucketAcl", "GET", "/mybucket?acl", "ROUTE:s3.bucket.get-acl|ACTION:s3:GetBucketAcl"},
		{"PutBucketAcl", "PUT", "/mybucket?acl", "ROUTE:s3.bucket.put-acl|ACTION:s3:PutBucketAcl"},
		{"ListObjectVersions", "GET", "/mybucket?versions", "ROUTE:s3.bucket.list-object-versions|ACTION:s3:ListBucketVersions"},
		{"ListMultipartUploads", "GET", "/mybucket?uploads", "ROUTE:s3.bucket.list-multipart-uploads|ACTION:s3:ListBucketMultipartUploads"},
		{"DeleteObjects", "POST", "/mybucket?delete", "ROUTE:s3.bucket.delete-objects|ACTION:s3:DeleteObject"},

		// Object operations (no query params)
		{"GetObject", "GET", "/mybucket/file.txt", "ROUTE:s3.object.get|ACTION:s3:GetObject"},
		{"GetObjectNested", "GET", "/mybucket/path/to/file.txt", "ROUTE:s3.object.get|ACTION:s3:GetObject"},
		{"PutObject", "PUT", "/mybucket/file.txt", "ROUTE:s3.object.put|ACTION:s3:PutObject"},
		{"DeleteObject", "DELETE", "/mybucket/file.txt", "ROUTE:s3.object.delete|ACTION:s3:DeleteObject"},
		{"HeadObject", "HEAD", "/mybucket/file.txt", "ROUTE:s3.object.head|ACTION:s3:GetObject"},

		// Object operations (with query params)
		{"GetObjectAcl", "GET", "/mybucket/file.txt?acl", "ROUTE:s3.object.get-acl|ACTION:s3:GetObjectAcl"},
		{"PutObjectAcl", "PUT", "/mybucket/file.txt?acl", "ROUTE:s3.object.put-acl|ACTION:s3:PutObjectAcl"},

		// Multipart upload operations
		{"CreateMultipartUpload", "POST", "/mybucket/file.txt?uploads", "ROUTE:s3.multipart.create|ACTION:s3:PutObject"},
		{"UploadPart", "PUT", "/mybucket/file.txt?partNumber=1&uploadId=abc123", "ROUTE:s3.multipart.upload-part|ACTION:s3:PutObject"},
		{"CompleteMultipartUpload", "POST", "/mybucket/file.txt?uploadId=abc123", "ROUTE:s3.multipart.complete|ACTION:s3:PutObject"},
		{"AbortMultipartUpload", "DELETE", "/mybucket/file.txt?uploadId=abc123", "ROUTE:s3.multipart.abort|ACTION:s3:AbortMultipartUpload"},
		{"ListParts", "GET", "/mybucket/file.txt?uploadId=abc123", "ROUTE:s3.multipart.list-parts|ACTION:s3:ListMultipartUploadParts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := request(t, r, tt.method, tt.path)
			assert.Equal(t, tt.expected, response.Body.String(), "%s %s response mismatch", tt.method, tt.path)
		})
	}

	// Verify total route count matches expectations
	routes := r.Routes()
	expectedRouteCount := 25 // Total S3 API routes
	assert.Len(t, routes, expectedRouteCount, "expected route count mismatch")
}

// Helper to make test requests
func request(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
