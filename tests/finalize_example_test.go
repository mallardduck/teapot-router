package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestFinalizeOptimization demonstrates the Finalize() optimization
func TestFinalizeOptimization(t *testing.T) {
	r := teapot.New()

	// Register routes
	r.Func().GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("SIMPLE"))
	})

	r.Func().GET("/with-meta", func(w http.ResponseWriter, r *http.Request) {
		action := teapot.GetAction(r)
		name := teapot.GetRouteName(r)
		_, _ = w.Write([]byte(action + ":" + name))
	}).Name("test").Action("s3:Test")

	// Optimize before serving
	r.Finalize()

	// Test routes work correctly
	req := httptest.NewRequest("GET", "/simple", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "SIMPLE", w.Body.String())

	req = httptest.NewRequest("GET", "/with-meta", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "s3:Test:test", w.Body.String())

	// Verify router is finalized
	assert.True(t, r.IsFinalized(), "router should be finalized")
}

// TestFinalizeIdempotent ensures calling Finalize multiple times is safe
func TestFinalizeIdempotent(t *testing.T) {
	r := teapot.New()
	r.Func().GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	r.Finalize()
	r.Finalize() // Should be safe to call again
	r.Finalize() // And again

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "OK", w.Body.String())
}

// TestS3WithFinalize demonstrates S3 API with Finalize
func TestS3WithFinalize(t *testing.T) {
	r := teapot.New()

	// Service endpoint
	r.Func().GET("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("LIST_BUCKETS"))
	}).Name("service.list").Action("s3:ListAllMyBuckets")

	// Bucket operations
	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.Func().PUT("", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("CREATE"))
		}).Name("create").Action("s3:CreateBucket")

		r.Func().QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("LIST"))
		}).Name("list").Action("s3:ListBucket")

		r.Func().QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ACL"))
		}).Name("acl").Action("s3:GetBucketAcl").Query("acl")
	})

	// Optimize!
	r.Finalize()

	// Test optimized routes
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/", "LIST_BUCKETS"},
		{"PUT", "/mybucket", "CREATE"},
		{"GET", "/mybucket", "LIST"},
		{"GET", "/mybucket?acl", "ACL"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, tt.expected, w.Body.String(), "%s %s", tt.method, tt.path)
	}
}
