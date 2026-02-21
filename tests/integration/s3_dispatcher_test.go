package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

type handler struct{}

// contextHandler is the single handler wired to every route in the test router.
// It reads the route name, action, and path parameters that the router / dispatcher
// injected into the request context and writes them to the response body.
// Asserting on that output is equivalent to asserting "this request was dispatched
// to the correct route AND the dispatcher populated the context correctly."
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ROUTE:%s|ACTION:%s|BUCKET:%s|KEY:%s",
		teapot.GetRouteName(r),
		teapot.GetAction(r),
		teapot.URLParam(r, "bucket"),
		teapot.URLParam(r, "key"),
	)
}

var contextHandler = handler{}

// setupS3Router registers every S3-style route from examples/routes-cli/main.go
// (the conditional debug route and the favicon handler are intentionally omitted).
// Every endpoint uses contextHandler so the test can verify dispatch purely from
// the response body.
func setupS3Router() *teapot.Router {
	r := teapot.New()

	// ── Service ─────────────────────────────────────────────────────────────
	r.GET("/", contextHandler).
		Name("s3.service.list-buckets").
		Action("api:s3:ListBuckets")

	// ── Bucket – direct ────────────────────────────────────────────────────
	r.PUT("/{bucket}", contextHandler).
		Name("s3.bucket.create").
		Action("api:s3:CreateBucket")
	r.DELETE("/{bucket}", contextHandler).
		Name("s3.bucket.delete").
		Action("api:s3:DeleteBucket")
	r.HEAD("/{bucket}", contextHandler).
		Name("s3.bucket.head").
		Action("api:s3:HeadBucket")
	r.GET("/{bucket}", contextHandler).
		Name("s3.bucket.list-objects-v1").
		Action("api:s3:ListObjects")

	// ── Bucket – query-dispatched ──────────────────────────────────────────
	r.QueryGET("/{bucket}", contextHandler).
		QueryValue("list-type", "2").
		Name("s3.bucket.list-objects-v2").
		Action("api:s3:ListObjectsV2")

	r.QueryGET("/{bucket}", contextHandler).
		Query("location").
		Name("s3.bucket.get-location").
		Action("api:s3:GetBucketLocation")
	r.QueryGET("/{bucket}", contextHandler).
		Query("versioning").
		Name("s3.bucket.get-versioning").
		Action("api:s3:GetBucketVersioning")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("versioning").
		Name("s3.bucket.put-versioning").
		Action("api:s3:PutBucketVersioning")

	r.QueryGET("/{bucket}", contextHandler).
		Query("acl").
		Name("s3.bucket.get-acl").
		Action("api:s3:GetBucketAcl")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("acl").
		Name("s3.bucket.put-acl").
		Action("api:s3:PutBucketAcl")

	r.QueryGET("/{bucket}", contextHandler).
		Query("policy").
		Name("s3.bucket.get-policy").
		Action("api:s3:GetBucketPolicy")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("policy").
		Name("s3.bucket.put-policy").
		Action("api:s3:PutBucketPolicy")
	r.QueryDELETE("/{bucket}", contextHandler).
		Query("policy").
		Name("s3.bucket.delete-policy").
		Action("api:s3:DeleteBucketPolicy")

	r.QueryGET("/{bucket}", contextHandler).
		Query("cors").
		Name("s3.bucket.get-cors").
		Action("api:s3:GetBucketCors")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("cors").
		Name("s3.bucket.put-cors").
		Action("api:s3:PutBucketCors")

	r.QueryGET("/{bucket}", contextHandler).
		Query("lifecycle").
		Name("s3.bucket.get-lifecycle-configuration").
		Action("api:s3:GetBucketLifecycleConfiguration")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("lifecycle").
		Name("s3.bucket.put-lifecycle-configuration").
		Action("api:s3:PutBucketLifecycleConfiguration")

	r.QueryGET("/{bucket}", contextHandler).
		Query("publicAccessBlock").
		Name("s3.bucket.get-public-access-block").
		Action("api:s3:GetPublicAccessBlock")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("publicAccessBlock").
		Name("s3.bucket.put-public-access-block").
		Action("api:s3:PutPublicAccessBlock")

	r.QueryPUT("/{bucket}", contextHandler).
		Query("object-lock").
		Name("s3.bucket.put-object-lock-configuration").
		Action("api:s3:PutObjectLockConfiguration")

	r.QueryPUT("/{bucket}", contextHandler).
		Query("logging").
		Name("s3.bucket.put-logging").
		Action("api:s3:PutBucketLogging")
	r.QueryPUT("/{bucket}", contextHandler).
		Query("notification").
		Name("s3.bucket.put-notification").
		Action("api:s3:PutBucketNotification")
	r.QueryGET("/{bucket}", contextHandler).
		Query("requestPayment").
		Name("s3.bucket.get-request-payment").
		Action("api:s3:GetBucketRequestPayment")
	r.QueryGET("/{bucket}", contextHandler).
		Query("analytics").
		Name("s3.bucket.get-analytics-configuration").
		Action("api:s3:GetBucketAnalyticsConfiguration")

	r.QueryGET("/{bucket}", contextHandler).
		Query("versions").
		Name("s3.bucket.list-object-versions").
		Action("api:s3:ListObjectVersions")
	r.QueryGET("/{bucket}", contextHandler).
		Query("uploads").
		Name("s3.bucket.list-multipart-uploads").
		Action("api:s3:ListMultipartUploads")
	r.QueryPOST("/{bucket}", contextHandler).
		Query("delete").
		Name("s3.bucket.delete-objects").
		Action("api:s3:DeleteObjects")

	// ── Object – direct ────────────────────────────────────────────────────
	r.GET("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.get").
		Action("api:s3:GetObject")
	r.PUT("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.put").
		Action("api:s3:PutObject")
	r.DELETE("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.delete").
		Action("api:s3:DeleteObject")
	r.HEAD("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.head").
		Action("api:s3:HeadObject")

	// ── Object – query-dispatched ──────────────────────────────────────────
	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("acl").
		Name("s3.object.get-acl").
		Action("api:s3:GetObjectAcl")
	r.QueryPUT("/{bucket}/{key:.*}", contextHandler).
		Query("acl").
		Name("s3.object.put-acl").
		Action("api:s3:PutObjectAcl")

	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("tagging").
		Name("s3.object.get-tagging").
		Action("api:s3:GetObjectTagging")
	r.QueryPUT("/{bucket}/{key:.*}", contextHandler).
		Query("tagging").
		Name("s3.object.put-tagging").
		Action("api:s3:PutObjectTagging")

	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("legal-hold").
		Name("s3.object.get-legal-hold").
		Action("api:s3:GetObjectLegalHold")
	r.QueryPUT("/{bucket}/{key:.*}", contextHandler).
		Query("legal-hold").
		Name("s3.object.put-legal-hold").
		Action("api:s3:PutObjectLegalHold")

	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("retention").
		Name("s3.object.get-retention").
		Action("api:s3:GetObjectRetention")
	r.QueryPUT("/{bucket}/{key:.*}", contextHandler).
		Query("retention").
		Name("s3.object.put-retention").
		Action("api:s3:PutObjectRetention")

	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("attributes").
		Name("s3.object.get-attributes").
		Action("api:s3:GetObjectAttributes")
	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("torrent").
		Name("s3.object.get-torrent").
		Action("api:s3:GetObjectTorrent")

	// ── Multipart ──────────────────────────────────────────────────────────
	r.QueryPOST("/{bucket}/{key:.*}", contextHandler).
		Query("uploads").
		Name("s3.multipart.create").
		Action("api:s3:CreateMultipartUpload")
	r.QueryPUT("/{bucket}/{key:.*}", contextHandler).
		Query("partNumber").Query("uploadId").
		Name("s3.multipart.upload-part").
		Action("api:s3:UploadPart")
	r.QueryPOST("/{bucket}/{key:.*}", contextHandler).
		Query("uploadId").
		Name("s3.multipart.complete").
		Action("api:s3:CompleteMultipartUpload")
	r.QueryDELETE("/{bucket}/{key:.*}", contextHandler).
		Query("uploadId").
		Name("s3.multipart.abort").
		Action("api:s3:AbortMultipartUpload")
	r.QueryGET("/{bucket}/{key:.*}", contextHandler).
		Query("uploadId").
		Name("s3.multipart.list-parts").
		Action("api:s3:ListParts")

	return r
}

// TestS3DispatcherIntegration sends one HTTP request per S3 route and verifies
// that the dispatcher resolved it to the correct handler with the correct
// context (route name, action, and extracted path parameters).
//
// Route registration order deliberately mirrors examples/routes-cli/main.go so
// that auto-promotion timing and dispatcher state match production usage.
func TestS3DispatcherIntegration(t *testing.T) {
	router := setupS3Router()

	// Guard: if route registration is broken the rest of the test is noise.
	require.Len(t, router.Routes(), 47,
		"route count mismatch – a route registration likely failed or was missed")

	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		// ── Service ───────────────────────────────────────────────────────
		{
			"Service/ListBuckets",
			"GET", "/",
			"ROUTE:s3.service.list-buckets|ACTION:api:s3:ListBuckets|BUCKET:|KEY:",
		},

		// ── Bucket – direct ─────────────────────────────────────────────
		{
			"Bucket/Create",
			"PUT", "/test-bucket",
			"ROUTE:s3.bucket.create|ACTION:api:s3:CreateBucket|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/Delete",
			"DELETE", "/test-bucket",
			"ROUTE:s3.bucket.delete|ACTION:api:s3:DeleteBucket|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/Head",
			"HEAD", "/test-bucket",
			"ROUTE:s3.bucket.head|ACTION:api:s3:HeadBucket|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/ListObjectsV1",
			"GET", "/test-bucket",
			"ROUTE:s3.bucket.list-objects-v1|ACTION:api:s3:ListObjects|BUCKET:test-bucket|KEY:",
		},

		// ── Bucket – query-dispatched ───────────────────────────────────
		{
			"Bucket/ListObjectsV2",
			"GET", "/test-bucket?list-type=2",
			"ROUTE:s3.bucket.list-objects-v2|ACTION:api:s3:ListObjectsV2|BUCKET:test-bucket|KEY:",
		},
		// list-type with a value other than "2" must fall back to the v1 route;
		// this exercises the QueryValue matcher's value-equality check.
		{
			"Bucket/ListObjectsV2_FallbackOnWrongValue",
			"GET", "/test-bucket?list-type=1",
			"ROUTE:s3.bucket.list-objects-v1|ACTION:api:s3:ListObjects|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetLocation",
			"GET", "/test-bucket?location",
			"ROUTE:s3.bucket.get-location|ACTION:api:s3:GetBucketLocation|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/GetVersioning",
			"GET", "/test-bucket?versioning",
			"ROUTE:s3.bucket.get-versioning|ACTION:api:s3:GetBucketVersioning|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutVersioning",
			"PUT", "/test-bucket?versioning",
			"ROUTE:s3.bucket.put-versioning|ACTION:api:s3:PutBucketVersioning|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetAcl",
			"GET", "/test-bucket?acl",
			"ROUTE:s3.bucket.get-acl|ACTION:api:s3:GetBucketAcl|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutAcl",
			"PUT", "/test-bucket?acl",
			"ROUTE:s3.bucket.put-acl|ACTION:api:s3:PutBucketAcl|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetPolicy",
			"GET", "/test-bucket?policy",
			"ROUTE:s3.bucket.get-policy|ACTION:api:s3:GetBucketPolicy|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutPolicy",
			"PUT", "/test-bucket?policy",
			"ROUTE:s3.bucket.put-policy|ACTION:api:s3:PutBucketPolicy|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/DeletePolicy",
			"DELETE", "/test-bucket?policy",
			"ROUTE:s3.bucket.delete-policy|ACTION:api:s3:DeleteBucketPolicy|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetCors",
			"GET", "/test-bucket?cors",
			"ROUTE:s3.bucket.get-cors|ACTION:api:s3:GetBucketCors|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutCors",
			"PUT", "/test-bucket?cors",
			"ROUTE:s3.bucket.put-cors|ACTION:api:s3:PutBucketCors|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetLifecycleConfiguration",
			"GET", "/test-bucket?lifecycle",
			"ROUTE:s3.bucket.get-lifecycle-configuration|ACTION:api:s3:GetBucketLifecycleConfiguration|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutLifecycleConfiguration",
			"PUT", "/test-bucket?lifecycle",
			"ROUTE:s3.bucket.put-lifecycle-configuration|ACTION:api:s3:PutBucketLifecycleConfiguration|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/GetPublicAccessBlock",
			"GET", "/test-bucket?publicAccessBlock",
			"ROUTE:s3.bucket.get-public-access-block|ACTION:api:s3:GetPublicAccessBlock|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutPublicAccessBlock",
			"PUT", "/test-bucket?publicAccessBlock",
			"ROUTE:s3.bucket.put-public-access-block|ACTION:api:s3:PutPublicAccessBlock|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/PutObjectLockConfiguration",
			"PUT", "/test-bucket?object-lock",
			"ROUTE:s3.bucket.put-object-lock-configuration|ACTION:api:s3:PutObjectLockConfiguration|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/PutLogging",
			"PUT", "/test-bucket?logging",
			"ROUTE:s3.bucket.put-logging|ACTION:api:s3:PutBucketLogging|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/PutNotification",
			"PUT", "/test-bucket?notification",
			"ROUTE:s3.bucket.put-notification|ACTION:api:s3:PutBucketNotification|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/GetRequestPayment",
			"GET", "/test-bucket?requestPayment",
			"ROUTE:s3.bucket.get-request-payment|ACTION:api:s3:GetBucketRequestPayment|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/GetAnalyticsConfiguration",
			"GET", "/test-bucket?analytics",
			"ROUTE:s3.bucket.get-analytics-configuration|ACTION:api:s3:GetBucketAnalyticsConfiguration|BUCKET:test-bucket|KEY:",
		},

		{
			"Bucket/ListObjectVersions",
			"GET", "/test-bucket?versions",
			"ROUTE:s3.bucket.list-object-versions|ACTION:api:s3:ListObjectVersions|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/ListMultipartUploads",
			"GET", "/test-bucket?uploads",
			"ROUTE:s3.bucket.list-multipart-uploads|ACTION:api:s3:ListMultipartUploads|BUCKET:test-bucket|KEY:",
		},
		{
			"Bucket/DeleteObjects",
			"POST", "/test-bucket?delete",
			"ROUTE:s3.bucket.delete-objects|ACTION:api:s3:DeleteObjects|BUCKET:test-bucket|KEY:",
		},

		// ── Object – direct ─────────────────────────────────────────────
		{
			"Object/Get",
			"GET", "/test-bucket/file.txt",
			"ROUTE:s3.object.get|ACTION:api:s3:GetObject|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/Put",
			"PUT", "/test-bucket/file.txt",
			"ROUTE:s3.object.put|ACTION:api:s3:PutObject|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/Delete",
			"DELETE", "/test-bucket/file.txt",
			"ROUTE:s3.object.delete|ACTION:api:s3:DeleteObject|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/Head",
			"HEAD", "/test-bucket/file.txt",
			"ROUTE:s3.object.head|ACTION:api:s3:HeadObject|BUCKET:test-bucket|KEY:file.txt",
		},

		// ── Object – query-dispatched ───────────────────────────────────
		{
			"Object/GetAcl",
			"GET", "/test-bucket/file.txt?acl",
			"ROUTE:s3.object.get-acl|ACTION:api:s3:GetObjectAcl|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/PutAcl",
			"PUT", "/test-bucket/file.txt?acl",
			"ROUTE:s3.object.put-acl|ACTION:api:s3:PutObjectAcl|BUCKET:test-bucket|KEY:file.txt",
		},

		{
			"Object/GetTagging",
			"GET", "/test-bucket/file.txt?tagging",
			"ROUTE:s3.object.get-tagging|ACTION:api:s3:GetObjectTagging|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/PutTagging",
			"PUT", "/test-bucket/file.txt?tagging",
			"ROUTE:s3.object.put-tagging|ACTION:api:s3:PutObjectTagging|BUCKET:test-bucket|KEY:file.txt",
		},

		{
			"Object/GetLegalHold",
			"GET", "/test-bucket/file.txt?legal-hold",
			"ROUTE:s3.object.get-legal-hold|ACTION:api:s3:GetObjectLegalHold|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/PutLegalHold",
			"PUT", "/test-bucket/file.txt?legal-hold",
			"ROUTE:s3.object.put-legal-hold|ACTION:api:s3:PutObjectLegalHold|BUCKET:test-bucket|KEY:file.txt",
		},

		{
			"Object/GetRetention",
			"GET", "/test-bucket/file.txt?retention",
			"ROUTE:s3.object.get-retention|ACTION:api:s3:GetObjectRetention|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/PutRetention",
			"PUT", "/test-bucket/file.txt?retention",
			"ROUTE:s3.object.put-retention|ACTION:api:s3:PutObjectRetention|BUCKET:test-bucket|KEY:file.txt",
		},

		{
			"Object/GetAttributes",
			"GET", "/test-bucket/file.txt?attributes",
			"ROUTE:s3.object.get-attributes|ACTION:api:s3:GetObjectAttributes|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Object/GetTorrent",
			"GET", "/test-bucket/file.txt?torrent",
			"ROUTE:s3.object.get-torrent|ACTION:api:s3:GetObjectTorrent|BUCKET:test-bucket|KEY:file.txt",
		},

		// ── Multipart ───────────────────────────────────────────────────
		{
			"Multipart/Create",
			"POST", "/test-bucket/file.txt?uploads",
			"ROUTE:s3.multipart.create|ACTION:api:s3:CreateMultipartUpload|BUCKET:test-bucket|KEY:file.txt",
		},
		// UploadPart requires both partNumber and uploadId; the dispatcher must
		// match both matchers before selecting this route over the put-object
		// fallback.
		{
			"Multipart/UploadPart",
			"PUT", "/test-bucket/file.txt?partNumber=1&uploadId=abc123",
			"ROUTE:s3.multipart.upload-part|ACTION:api:s3:UploadPart|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Multipart/Complete",
			"POST", "/test-bucket/file.txt?uploadId=abc123",
			"ROUTE:s3.multipart.complete|ACTION:api:s3:CompleteMultipartUpload|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Multipart/Abort",
			"DELETE", "/test-bucket/file.txt?uploadId=abc123",
			"ROUTE:s3.multipart.abort|ACTION:api:s3:AbortMultipartUpload|BUCKET:test-bucket|KEY:file.txt",
		},
		{
			"Multipart/ListParts",
			"GET", "/test-bucket/file.txt?uploadId=abc123",
			"ROUTE:s3.multipart.list-parts|ACTION:api:s3:ListParts|BUCKET:test-bucket|KEY:file.txt",
		},

		// ── Nested-key spot checks ──────────────────────────────────────
		// The {key:.*} wildcard must capture the entire remaining path; these
		// cases verify that both direct and query-dispatched object routes do so.
		{
			"Object/Get_NestedKey",
			"GET", "/test-bucket/path/to/nested.txt",
			"ROUTE:s3.object.get|ACTION:api:s3:GetObject|BUCKET:test-bucket|KEY:path/to/nested.txt",
		},
		{
			"Object/GetAcl_NestedKey",
			"GET", "/test-bucket/a/b/c.txt?acl",
			"ROUTE:s3.object.get-acl|ACTION:api:s3:GetObjectAcl|BUCKET:test-bucket|KEY:a/b/c.txt",
		},
		{
			"Multipart/Create_NestedKey",
			"POST", "/test-bucket/uploads/large-file.bin?uploads",
			"ROUTE:s3.multipart.create|ACTION:api:s3:CreateMultipartUpload|BUCKET:test-bucket|KEY:uploads/large-file.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "%s %s", tt.method, tt.path)
			assert.Equal(t, tt.expected, w.Body.String(), "%s %s", tt.method, tt.path)
		})
	}
}

// routeNotImplemented is the shared stub handler that mimics the pattern where
// an application assigns a single "not yet built" handler to multiple routes.
// It writes 501 and includes the dispatcher-injected context in the body so the
// test can verify the correct route was reached.  A 200 response for a URL that
// should hit this handler means the fallback was served instead — that is the
// exact failure mode being tested.
var routeNotImplemented = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "ROUTE:%s|ACTION:%s|BUCKET:%s|KEY:%s",
		teapot.GetRouteName(r),
		teapot.GetAction(r),
		teapot.URLParam(r, "bucket"),
		teapot.URLParam(r, "key"),
	)
})

// TestS3DispatcherWithStubHandlers runs the full S3 route set with a
// deliberate split between "implemented" routes (contextHandler, 200) and
// "stub" routes (routeNotImplemented, 501).  The specific routes that are
// stubbed match the pattern the user reported: the routes that worked in their
// app (?location, ?policy, ?list-type=2) keep their real handler; the routes
// that fell through (?versioning, ?acl, ?cors, …) are wired to the shared stub.
//
// Every stub assertion checks status == 501.  A 200 means the fallback was
// dispatched instead — the bug.
func TestS3DispatcherWithStubHandlers(t *testing.T) {
	r := teapot.New()

	// ── Bucket – GET ──────────────────────────────────────────────────────
	// Fallback – implemented
	r.GET("/{bucket}", contextHandler).
		Name("s3.bucket.list-objects-v1").
		Action("api:s3:ListObjects")

	// "Working" routes – keep real handler (these are the ones that worked
	// for the user even when other routes were stubbed)
	r.QueryGET("/{bucket}", contextHandler).
		QueryValue("list-type", "2").
		Name("s3.bucket.list-objects-v2").
		Action("api:s3:ListObjectsV2")
	r.QueryGET("/{bucket}", contextHandler).
		Query("location").
		Name("s3.bucket.get-location").
		Action("api:s3:GetBucketLocation")
	r.QueryGET("/{bucket}", contextHandler).
		Query("policy").
		Name("s3.bucket.get-policy").
		Action("api:s3:GetBucketPolicy")

	// Stub routes – shared routeNotImplemented handler
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("versioning").
		Name("s3.bucket.get-versioning").
		Action("api:s3:GetBucketVersioning")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("acl").
		Name("s3.bucket.get-acl").
		Action("api:s3:GetBucketAcl")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("cors").
		Name("s3.bucket.get-cors").
		Action("api:s3:GetBucketCors")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("lifecycle").
		Name("s3.bucket.get-lifecycle-configuration").
		Action("api:s3:GetBucketLifecycleConfiguration")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("publicAccessBlock").
		Name("s3.bucket.get-public-access-block").
		Action("api:s3:GetPublicAccessBlock")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("requestPayment").
		Name("s3.bucket.get-request-payment").
		Action("api:s3:GetBucketRequestPayment")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("analytics").
		Name("s3.bucket.get-analytics-configuration").
		Action("api:s3:GetBucketAnalyticsConfiguration")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("versions").
		Name("s3.bucket.list-object-versions").
		Action("api:s3:ListObjectVersions")
	r.QueryGET("/{bucket}", routeNotImplemented).
		Query("uploads").
		Name("s3.bucket.list-multipart-uploads").
		Action("api:s3:ListMultipartUploads")

	// ── Bucket – PUT ──────────────────────────────────────────────────────
	r.PUT("/{bucket}", contextHandler).
		Name("s3.bucket.create").
		Action("api:s3:CreateBucket")

	r.QueryPUT("/{bucket}", contextHandler). // one real PUT query route
							Query("acl").
							Name("s3.bucket.put-acl").
							Action("api:s3:PutBucketAcl")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("versioning").
		Name("s3.bucket.put-versioning").
		Action("api:s3:PutBucketVersioning")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("cors").
		Name("s3.bucket.put-cors").
		Action("api:s3:PutBucketCors")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("policy").
		Name("s3.bucket.put-policy").
		Action("api:s3:PutBucketPolicy")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("lifecycle").
		Name("s3.bucket.put-lifecycle-configuration").
		Action("api:s3:PutBucketLifecycleConfiguration")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("publicAccessBlock").
		Name("s3.bucket.put-public-access-block").
		Action("api:s3:PutPublicAccessBlock")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("object-lock").
		Name("s3.bucket.put-object-lock-configuration").
		Action("api:s3:PutObjectLockConfiguration")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("logging").
		Name("s3.bucket.put-logging").
		Action("api:s3:PutBucketLogging")
	r.QueryPUT("/{bucket}", routeNotImplemented).
		Query("notification").
		Name("s3.bucket.put-notification").
		Action("api:s3:PutBucketNotification")

	// ── Bucket – DELETE ───────────────────────────────────────────────────
	r.DELETE("/{bucket}", contextHandler).
		Name("s3.bucket.delete").
		Action("api:s3:DeleteBucket")
	r.QueryDELETE("/{bucket}", routeNotImplemented).
		Query("policy").
		Name("s3.bucket.delete-policy").
		Action("api:s3:DeleteBucketPolicy")

	// ── Bucket – POST (no fallback – stub is the only handler) ───────────
	r.QueryPOST("/{bucket}", routeNotImplemented).
		Query("delete").
		Name("s3.bucket.delete-objects").
		Action("api:s3:DeleteObjects")

	// ── Bucket – HEAD (direct only, no query variants) ───────────────────
	r.HEAD("/{bucket}", contextHandler).
		Name("s3.bucket.head").
		Action("api:s3:HeadBucket")

	// ── Object – GET ──────────────────────────────────────────────────────
	r.GET("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.get").
		Action("api:s3:GetObject")
	r.QueryGET("/{bucket}/{key:.*}", contextHandler). // one real
								Query("acl").
								Name("s3.object.get-acl").
								Action("api:s3:GetObjectAcl")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("tagging").
		Name("s3.object.get-tagging").
		Action("api:s3:GetObjectTagging")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("legal-hold").
		Name("s3.object.get-legal-hold").
		Action("api:s3:GetObjectLegalHold")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("retention").
		Name("s3.object.get-retention").
		Action("api:s3:GetObjectRetention")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("attributes").
		Name("s3.object.get-attributes").
		Action("api:s3:GetObjectAttributes")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("torrent").
		Name("s3.object.get-torrent").
		Action("api:s3:GetObjectTorrent")
	r.QueryGET("/{bucket}/{key:.*}", routeNotImplemented).
		Query("uploadId").
		Name("s3.multipart.list-parts").
		Action("api:s3:ListParts")

	// ── Object – PUT ──────────────────────────────────────────────────────
	r.PUT("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.put").
		Action("api:s3:PutObject")
	r.QueryPUT("/{bucket}/{key:.*}", routeNotImplemented).
		Query("acl").
		Name("s3.object.put-acl").
		Action("api:s3:PutObjectAcl")
	r.QueryPUT("/{bucket}/{key:.*}", routeNotImplemented).
		Query("tagging").
		Name("s3.object.put-tagging").
		Action("api:s3:PutObjectTagging")
	r.QueryPUT("/{bucket}/{key:.*}", routeNotImplemented).
		Query("legal-hold").
		Name("s3.object.put-legal-hold").
		Action("api:s3:PutObjectLegalHold")
	r.QueryPUT("/{bucket}/{key:.*}", routeNotImplemented).
		Query("retention").
		Name("s3.object.put-retention").
		Action("api:s3:PutObjectRetention")
	r.QueryPUT("/{bucket}/{key:.*}", routeNotImplemented).
		Query("partNumber").Query("uploadId").
		Name("s3.multipart.upload-part").
		Action("api:s3:UploadPart")

	// ── Object – DELETE ───────────────────────────────────────────────────
	r.DELETE("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.delete").
		Action("api:s3:DeleteObject")
	r.QueryDELETE("/{bucket}/{key:.*}", routeNotImplemented).
		Query("uploadId").
		Name("s3.multipart.abort").
		Action("api:s3:AbortMultipartUpload")

	// ── Object – POST (no fallback) ───────────────────────────────────────
	r.QueryPOST("/{bucket}/{key:.*}", routeNotImplemented).
		Query("uploads").
		Name("s3.multipart.create").
		Action("api:s3:CreateMultipartUpload")
	r.QueryPOST("/{bucket}/{key:.*}", routeNotImplemented).
		Query("uploadId").
		Name("s3.multipart.complete").
		Action("api:s3:CompleteMultipartUpload")

	// ── Object – HEAD (direct only) ───────────────────────────────────────
	r.HEAD("/{bucket}/{key:.*}", contextHandler).
		Name("s3.object.head").
		Action("api:s3:HeadObject")

	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		// ── Fallbacks (real handler → 200) ──────────────────────────────
		{
			"Fallback/BucketGet", "GET", "/tb", 200,
			"ROUTE:s3.bucket.list-objects-v1|ACTION:api:s3:ListObjects|BUCKET:tb|KEY:",
		},
		{
			"Fallback/BucketPut", "PUT", "/tb", 200,
			"ROUTE:s3.bucket.create|ACTION:api:s3:CreateBucket|BUCKET:tb|KEY:",
		},
		{
			"Fallback/BucketDelete", "DELETE", "/tb", 200,
			"ROUTE:s3.bucket.delete|ACTION:api:s3:DeleteBucket|BUCKET:tb|KEY:",
		},
		{
			"Fallback/ObjectGet", "GET", "/tb/f.txt", 200,
			"ROUTE:s3.object.get|ACTION:api:s3:GetObject|BUCKET:tb|KEY:f.txt",
		},
		{
			"Fallback/ObjectPut", "PUT", "/tb/f.txt", 200,
			"ROUTE:s3.object.put|ACTION:api:s3:PutObject|BUCKET:tb|KEY:f.txt",
		},
		{
			"Fallback/ObjectDelete", "DELETE", "/tb/f.txt", 200,
			"ROUTE:s3.object.delete|ACTION:api:s3:DeleteObject|BUCKET:tb|KEY:f.txt",
		},

		// ── Implemented query routes (200) ──────────────────────────────
		{
			"Impl/ListObjectsV2", "GET", "/tb?list-type=2", 200,
			"ROUTE:s3.bucket.list-objects-v2|ACTION:api:s3:ListObjectsV2|BUCKET:tb|KEY:",
		},
		{
			"Impl/GetLocation", "GET", "/tb?location", 200,
			"ROUTE:s3.bucket.get-location|ACTION:api:s3:GetBucketLocation|BUCKET:tb|KEY:",
		},
		{
			"Impl/GetPolicy", "GET", "/tb?policy", 200,
			"ROUTE:s3.bucket.get-policy|ACTION:api:s3:GetBucketPolicy|BUCKET:tb|KEY:",
		},
		{
			"Impl/PutAcl", "PUT", "/tb?acl", 200,
			"ROUTE:s3.bucket.put-acl|ACTION:api:s3:PutBucketAcl|BUCKET:tb|KEY:",
		},
		{
			"Impl/ObjectGetAcl", "GET", "/tb/f.txt?acl", 200,
			"ROUTE:s3.object.get-acl|ACTION:api:s3:GetObjectAcl|BUCKET:tb|KEY:f.txt",
		},
		// QueryValue with wrong value must fall back to v1 (not 404, not v2)
		{
			"Impl/ListObjectsV2_WrongValue", "GET", "/tb?list-type=1", 200,
			"ROUTE:s3.bucket.list-objects-v1|ACTION:api:s3:ListObjects|BUCKET:tb|KEY:",
		},

		// ── Stub query routes (501) – core regression surface ──────────
		// A 200 here means the fallback was served instead of the stub.
		// Bucket GET stubs
		{
			"Stub/BucketGetVersioning", "GET", "/tb?versioning", 501,
			"ROUTE:s3.bucket.get-versioning|ACTION:api:s3:GetBucketVersioning|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetAcl", "GET", "/tb?acl", 501,
			"ROUTE:s3.bucket.get-acl|ACTION:api:s3:GetBucketAcl|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetCors", "GET", "/tb?cors", 501,
			"ROUTE:s3.bucket.get-cors|ACTION:api:s3:GetBucketCors|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetLifecycle", "GET", "/tb?lifecycle", 501,
			"ROUTE:s3.bucket.get-lifecycle-configuration|ACTION:api:s3:GetBucketLifecycleConfiguration|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetPublicAccessBlock", "GET", "/tb?publicAccessBlock", 501,
			"ROUTE:s3.bucket.get-public-access-block|ACTION:api:s3:GetPublicAccessBlock|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetRequestPayment", "GET", "/tb?requestPayment", 501,
			"ROUTE:s3.bucket.get-request-payment|ACTION:api:s3:GetBucketRequestPayment|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketGetAnalytics", "GET", "/tb?analytics", 501,
			"ROUTE:s3.bucket.get-analytics-configuration|ACTION:api:s3:GetBucketAnalyticsConfiguration|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketListObjectVersions", "GET", "/tb?versions", 501,
			"ROUTE:s3.bucket.list-object-versions|ACTION:api:s3:ListObjectVersions|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketListMultipartUploads", "GET", "/tb?uploads", 501,
			"ROUTE:s3.bucket.list-multipart-uploads|ACTION:api:s3:ListMultipartUploads|BUCKET:tb|KEY:",
		},
		// Bucket PUT stubs
		{
			"Stub/BucketPutVersioning", "PUT", "/tb?versioning", 501,
			"ROUTE:s3.bucket.put-versioning|ACTION:api:s3:PutBucketVersioning|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutCors", "PUT", "/tb?cors", 501,
			"ROUTE:s3.bucket.put-cors|ACTION:api:s3:PutBucketCors|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutPolicy", "PUT", "/tb?policy", 501,
			"ROUTE:s3.bucket.put-policy|ACTION:api:s3:PutBucketPolicy|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutLifecycle", "PUT", "/tb?lifecycle", 501,
			"ROUTE:s3.bucket.put-lifecycle-configuration|ACTION:api:s3:PutBucketLifecycleConfiguration|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutPublicAccessBlock", "PUT", "/tb?publicAccessBlock", 501,
			"ROUTE:s3.bucket.put-public-access-block|ACTION:api:s3:PutPublicAccessBlock|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutObjectLock", "PUT", "/tb?object-lock", 501,
			"ROUTE:s3.bucket.put-object-lock-configuration|ACTION:api:s3:PutObjectLockConfiguration|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutLogging", "PUT", "/tb?logging", 501,
			"ROUTE:s3.bucket.put-logging|ACTION:api:s3:PutBucketLogging|BUCKET:tb|KEY:",
		},
		{
			"Stub/BucketPutNotification", "PUT", "/tb?notification", 501,
			"ROUTE:s3.bucket.put-notification|ACTION:api:s3:PutBucketNotification|BUCKET:tb|KEY:",
		},
		// Bucket DELETE stub
		{
			"Stub/BucketDeletePolicy", "DELETE", "/tb?policy", 501,
			"ROUTE:s3.bucket.delete-policy|ACTION:api:s3:DeleteBucketPolicy|BUCKET:tb|KEY:",
		},
		// Bucket POST stub (no fallback — stub is the only handler)
		{
			"Stub/BucketDeleteObjects", "POST", "/tb?delete", 501,
			"ROUTE:s3.bucket.delete-objects|ACTION:api:s3:DeleteObjects|BUCKET:tb|KEY:",
		},

		// Object GET stubs
		{
			"Stub/ObjectGetTagging", "GET", "/tb/f.txt?tagging", 501,
			"ROUTE:s3.object.get-tagging|ACTION:api:s3:GetObjectTagging|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectGetLegalHold", "GET", "/tb/f.txt?legal-hold", 501,
			"ROUTE:s3.object.get-legal-hold|ACTION:api:s3:GetObjectLegalHold|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectGetRetention", "GET", "/tb/f.txt?retention", 501,
			"ROUTE:s3.object.get-retention|ACTION:api:s3:GetObjectRetention|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectGetAttributes", "GET", "/tb/f.txt?attributes", 501,
			"ROUTE:s3.object.get-attributes|ACTION:api:s3:GetObjectAttributes|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectGetTorrent", "GET", "/tb/f.txt?torrent", 501,
			"ROUTE:s3.object.get-torrent|ACTION:api:s3:GetObjectTorrent|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/MultipartListParts", "GET", "/tb/f.txt?uploadId=abc", 501,
			"ROUTE:s3.multipart.list-parts|ACTION:api:s3:ListParts|BUCKET:tb|KEY:f.txt",
		},
		// Object PUT stubs
		{
			"Stub/ObjectPutAcl", "PUT", "/tb/f.txt?acl", 501,
			"ROUTE:s3.object.put-acl|ACTION:api:s3:PutObjectAcl|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectPutTagging", "PUT", "/tb/f.txt?tagging", 501,
			"ROUTE:s3.object.put-tagging|ACTION:api:s3:PutObjectTagging|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectPutLegalHold", "PUT", "/tb/f.txt?legal-hold", 501,
			"ROUTE:s3.object.put-legal-hold|ACTION:api:s3:PutObjectLegalHold|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/ObjectPutRetention", "PUT", "/tb/f.txt?retention", 501,
			"ROUTE:s3.object.put-retention|ACTION:api:s3:PutObjectRetention|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/MultipartUploadPart", "PUT", "/tb/f.txt?partNumber=1&uploadId=abc", 501,
			"ROUTE:s3.multipart.upload-part|ACTION:api:s3:UploadPart|BUCKET:tb|KEY:f.txt",
		},
		// Object DELETE stub
		{
			"Stub/MultipartAbort", "DELETE", "/tb/f.txt?uploadId=abc", 501,
			"ROUTE:s3.multipart.abort|ACTION:api:s3:AbortMultipartUpload|BUCKET:tb|KEY:f.txt",
		},
		// Object POST stubs (no fallback)
		{
			"Stub/MultipartCreate", "POST", "/tb/f.txt?uploads", 501,
			"ROUTE:s3.multipart.create|ACTION:api:s3:CreateMultipartUpload|BUCKET:tb|KEY:f.txt",
		},
		{
			"Stub/MultipartComplete", "POST", "/tb/f.txt?uploadId=abc", 501,
			"ROUTE:s3.multipart.complete|ACTION:api:s3:CompleteMultipartUpload|BUCKET:tb|KEY:f.txt",
		},

		// ── HEAD (direct, real handler) ─────────────────────────────────
		{
			"Head/Bucket", "HEAD", "/tb", 200,
			"ROUTE:s3.bucket.head|ACTION:api:s3:HeadBucket|BUCKET:tb|KEY:",
		},
		{
			"Head/Object", "HEAD", "/tb/f.txt", 200,
			"ROUTE:s3.object.head|ACTION:api:s3:HeadObject|BUCKET:tb|KEY:f.txt",
		},

		// ── Nested keys with stub handler ───────────────────────────────
		{
			"Stub/ObjectGetTagging_NestedKey", "GET", "/tb/a/b/c.txt?tagging", 501,
			"ROUTE:s3.object.get-tagging|ACTION:api:s3:GetObjectTagging|BUCKET:tb|KEY:a/b/c.txt",
		},
		{
			"Stub/MultipartCreate_NestedKey", "POST", "/tb/up/large.bin?uploads", 501,
			"ROUTE:s3.multipart.create|ACTION:api:s3:CreateMultipartUpload|BUCKET:tb|KEY:up/large.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code,
				"%s %s: wrong status — 200 here means the fallback was served instead of the expected handler",
				tt.method, tt.path)
			assert.Equal(t, tt.wantBody, w.Body.String(), "%s %s", tt.method, tt.path)
		})
	}
}

// TestDispatcherPromotionWithSharedHandler isolates the auto-promotion code
// path that the user hit.  Each sub-test varies the registration order to
// exercise every branch:
//
//   - direct-first:  handleDirect registers optimizedHandler, then handleQuery
//     auto-promotes via a second mux.Method() call.  This is the order in the
//     user's app and in examples/routes-cli/main.go.
//   - query-first:   handleQuery creates the dispatcher first; handleDirect
//     later finds it and adds the fallback without a second mux.Method() call.
//   - many-stubs:    10 query routes share the same stub function on a single
//     dispatcher.  Between AddRoute and the subsequent .Query() call each new
//     route briefly has specificity 0 (same as the fallback); sort.Slice is
//     not stable, so the relative order is undefined until UpdateSpecificity
//     re-sorts.  This pins that the final sorted order is always correct.
//   - wildcard:      same as direct-first but on a {key:.*} pattern, which
//     translates to chi's /* — the second mux.Method() call overwrites on that
//     pattern variant.
func TestDispatcherPromotionWithSharedHandler(t *testing.T) {
	stub := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "stub:%s", teapot.GetRouteName(r))
	}
	fallback := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "fallback:%s", teapot.GetRouteName(r))
	}

	t.Run("direct first then query – user registration order", func(t *testing.T) {
		// Mirrors the exact snippet from the user's app:
		//   r.PUT("/{bucket}", deps.createBucket)…
		//   r.GET("/{bucket}", deps.listObjects)…
		//   r.QueryGET("/{bucket}", deps.getBucketVersioning).Query("versioning")…
		//   r.QueryPUT("/{bucket}", deps.putBucketVersioning).Query("versioning")…
		r := teapot.New()
		r.Func().PUT("/{bucket}", fallback).Name("put-create").Action("put-create")
		r.Func().GET("/{bucket}", fallback).Name("get-list").Action("get-list")
		r.Func().QueryGET("/{bucket}", stub).Query("versioning").Name("get-versioning").Action("get-versioning")
		r.Func().QueryPUT("/{bucket}", stub).Query("versioning").Name("put-versioning").Action("put-versioning")

		tests := []struct {
			method   string
			path     string
			wantCode int
			wantBody string
		}{
			{"GET", "/b", 200, "fallback:get-list"},
			{"PUT", "/b", 200, "fallback:put-create"},
			{"GET", "/b?versioning", 501, "stub:get-versioning"},
			{"PUT", "/b?versioning", 501, "stub:put-versioning"},
		}
		for _, tt := range tests {
			t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
				req := httptest.NewRequest(tt.method, tt.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, tt.wantCode, w.Code, "status")
				assert.Equal(t, tt.wantBody, w.Body.String(), "body")
			})
		}
	})

	t.Run("query first then direct – reversed order", func(t *testing.T) {
		// Dispatcher is created by the first QueryGET.  The subsequent GET
		// goes through handleDirect's "dispatcher already exists" branch —
		// no second mux.Method() call.
		r := teapot.New()
		r.Func().QueryGET("/{bucket}", stub).Query("versioning").Name("get-versioning").Action("get-versioning")
		r.Func().QueryGET("/{bucket}", stub).Query("acl").Name("get-acl").Action("get-acl")
		r.Func().QueryGET("/{bucket}", stub).Query("cors").Name("get-cors").Action("get-cors")
		r.Func().GET("/{bucket}", fallback).Name("get-list").Action("get-list")

		tests := []struct {
			path     string
			wantCode int
			wantBody string
		}{
			{"/b", 200, "fallback:get-list"},
			{"/b?versioning", 501, "stub:get-versioning"},
			{"/b?acl", 501, "stub:get-acl"},
			{"/b?cors", 501, "stub:get-cors"},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, tt.wantCode, w.Code, "status")
				assert.Equal(t, tt.wantBody, w.Body.String(), "body")
			})
		}
	})

	t.Run("many stubs on one dispatcher – sort stability", func(t *testing.T) {
		// 10 query routes all backed by the same stub, registered after the
		// direct fallback so every one goes through the auto-promotion path.
		// sort.Slice is unstable; this verifies that after UpdateSpecificity
		// each route is reachable regardless of its position among peers with
		// equal specificity.
		r := teapot.New()
		r.Func().GET("/{bucket}", fallback).Name("list").Action("list")

		queryKeys := []string{
			"versioning", "acl", "cors", "policy", "lifecycle",
			"location", "requestPayment", "analytics", "versions", "uploads",
		}
		for _, key := range queryKeys {
			r.Func().QueryGET("/{bucket}", stub).Query(key).Name(key).Action(key)
		}

		// Every query key must reach its own stub
		for _, key := range queryKeys {
			t.Run(key, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/b?"+key, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, 501, w.Code, "expected stub (501); 200 means fallback was hit")
				assert.Equal(t, "stub:"+key, w.Body.String())
			})
		}
		// Fallback must still work
		t.Run("fallback", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/b", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "fallback:list", w.Body.String())
		})
	})

	t.Run("wildcard pattern – chi /* re-registration", func(t *testing.T) {
		// {key:.*} translates to /* in chi.  The direct route registers first,
		// then the first QueryGET auto-promotes and calls mux.Method() a
		// second time on the /* pattern.  Verifies the overwrite takes effect
		// on the translated pattern, not just plain {param} patterns.
		r := teapot.New()
		r.Func().GET("/{bucket}/{key:.*}", fallback).Name("get-object").Action("get-object")
		r.Func().QueryGET("/{bucket}/{key:.*}", stub).Query("tagging").Name("get-tagging").Action("get-tagging")
		r.Func().QueryGET("/{bucket}/{key:.*}", stub).Query("retention").Name("get-retention").Action("get-retention")
		r.Func().QueryGET("/{bucket}/{key:.*}", stub).Query("acl").Name("get-acl").Action("get-acl")

		tests := []struct {
			path     string
			wantCode int
			wantBody string
		}{
			{"/b/k", 200, "fallback:get-object"},
			{"/b/k?tagging", 501, "stub:get-tagging"},
			{"/b/k?retention", 501, "stub:get-retention"},
			{"/b/k?acl", 501, "stub:get-acl"},
			// nested key – wildcard must capture full remainder
			{"/b/a/b/c?tagging", 501, "stub:get-tagging"},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, tt.wantCode, w.Code, "status")
				assert.Equal(t, tt.wantBody, w.Body.String(), "body")
			})
		}
	})
}

// TestNilHandlerPanicsAtDispatch verifies that a nil http.HandlerFunc (the
// zero value of an uninitialized struct field) passed during route
// registration does NOT panic at registration time, but DOES panic when the
// route is actually dispatched to.
//
// This is distinct from the fallback-served-instead pattern tested by
// TestS3DispatcherWithStubHandlers.  A nil handler panics — a recovery
// middleware would turn that into a 500.  It does NOT silently fall through
// to the next route in the dispatcher.
//
// Code paths exercised:
//   - optimizedHandler.slowPath — direct route, never promoted to dispatcher
//   - Dispatcher.executeRoute   — query-dispatched route or promoted fallback
func TestNilHandlerPanicsAtDispatch(t *testing.T) {
	// Simulate the zero-value deps struct the user described.
	// All fields are nil http.HandlerFunc — exactly what you get when you
	// read struct fields before populating them.
	type deps struct {
		listObjects         http.HandlerFunc
		getBucketVersioning http.HandlerFunc
		getObject           http.HandlerFunc
	}
	var d deps // every field is nil

	t.Run("direct – single nil handler, not promoted", func(t *testing.T) {
		// Only one PUT on /{bucket}; no query variants exist so the route
		// stays as an optimizedHandler and hits slowPath at dispatch time.
		r := teapot.New()
		r.PUT("/{bucket}", d.listObjects).Name("create").Action("create")

		assert.Panics(t, func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/b", nil))
		})
	})

	t.Run("query – nil dispatched route, real fallback and peers work", func(t *testing.T) {
		// Dispatcher has: real fallback, real ?location, nil ?versioning.
		// The nil route is route-specific: the others must still succeed.
		r := teapot.New()
		r.GET("/{bucket}", contextHandler).Name("list").Action("list")
		r.QueryGET("/{bucket}", contextHandler).Query("location").Name("get-location").Action("get-location")
		r.QueryGET("/{bucket}", d.getBucketVersioning).Query("versioning").Name("get-versioning").Action("get-versioning")

		// Fallback → 200
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/b", nil))
		assert.Equal(t, 200, w.Code)

		// Real query route → 200
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/b?location", nil))
		assert.Equal(t, 200, w.Code)

		// Nil query route → panic (does NOT fall through to fallback)
		assert.Panics(t, func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/b?versioning", nil))
		})
	})

	t.Run("nil fallback – real query route still dispatches", func(t *testing.T) {
		// The direct GET is nil.  When the QueryGET is registered, auto-
		// promotion moves the nil route into the dispatcher as the fallback.
		// The query route works; hitting the fallback (no query param) panics.
		r := teapot.New()
		r.GET("/{bucket}", d.listObjects).Name("list").Action("list")
		r.QueryGET("/{bucket}", contextHandler).Query("versioning").Name("get-versioning").Action("get-versioning")

		// Query route → 200
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/b?versioning", nil))
		assert.Equal(t, 200, w.Code)

		// Fallback → panic
		assert.Panics(t, func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/b", nil))
		})
	})

	t.Run("wildcard – nil query route on {key:.*}", func(t *testing.T) {
		// Same as "query" above but on /{bucket}/{key:.*}.  The pattern
		// translates to /* in chi; auto-promotion calls mux.Method() a second
		// time on that translated pattern.  Nested keys are used so the
		// wildcard capture is exercised too.
		r := teapot.New()
		r.GET("/{bucket}/{key:.*}", contextHandler).Name("get-object").Action("get-object")
		r.QueryGET("/{bucket}/{key:.*}", d.getObject).Query("tagging").Name("get-tagging").Action("get-tagging")
		r.QueryGET("/{bucket}/{key:.*}", contextHandler).Query("versioning").Name("get-versioning").Action("get-versioning")

		// Fallback with nested key → 200
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/b/a/b/c", nil))
		assert.Equal(t, 200, w.Code)

		// Nil query route with nested key → panic
		assert.Panics(t, func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/b/a/b/c?tagging", nil))
		})

		versionW := httptest.NewRecorder()
		assert.NotPanics(t, func() {
			r.ServeHTTP(versionW, httptest.NewRequest("GET", "/b/a/b/c?versioning", nil))
		})
		assert.Equal(t, 200, versionW.Code)
		assert.Equal(t, "ROUTE:get-versioning|ACTION:get-versioning|BUCKET:b|KEY:a/b/c", versionW.Body.String())
	})

	t.Run("multi-matcher – nil handler gated by two query params", func(t *testing.T) {
		// The nil route requires both partNumber AND uploadId.  Sending only
		// one param fails the matcher → fallback is served (200, no panic).
		// Sending both satisfies the matcher → nil handler is reached → panic.
		// This confirms the panic is gated by the full matcher chain, not
		// triggered unconditionally when a nil route exists on the dispatcher.
		r := teapot.New()
		r.PUT("/{bucket}/{key:.*}", contextHandler).Name("put-object").Action("put-object")
		r.QueryPUT("/{bucket}/{key:.*}", d.getObject).
			Query("partNumber").Query("uploadId").
			Name("upload-part").Action("upload-part")

		// No query params → fallback
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("PUT", "/b/k", nil))
		assert.Equal(t, 200, w.Code)

		// Only partNumber → matcher not fully satisfied → fallback
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("PUT", "/b/k?partNumber=1", nil))
		assert.Equal(t, 200, w.Code)

		// Both params → matcher satisfied → nil handler → panic
		assert.Panics(t, func() {
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/b/k?partNumber=1&uploadId=abc", nil))
		})
	})
}
