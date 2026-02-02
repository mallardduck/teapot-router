package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestFinalizeOptimization demonstrates the Finalize() optimization
func TestFinalizeOptimization(t *testing.T) {
	r := teapot.New()

	// Register routes
	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("SIMPLE"))
	})

	r.GET("/with-meta", func(w http.ResponseWriter, r *http.Request) {
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
	if w.Body.String() != "SIMPLE" {
		t.Errorf("expected SIMPLE, got %s", w.Body.String())
	}

	req = httptest.NewRequest("GET", "/with-meta", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != "s3:Test:test" {
		t.Errorf("expected s3:Test:test, got %s", w.Body.String())
	}

	// Verify router is finalized
	if !r.IsFinalized() {
		t.Error("router should be finalized")
	}
}

// TestFinalizeIdempotent ensures calling Finalize multiple times is safe
func TestFinalizeIdempotent(t *testing.T) {
	r := teapot.New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	r.Finalize()
	r.Finalize() // Should be safe to call again
	r.Finalize() // And again

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "OK" {
		t.Errorf("expected OK, got %s", w.Body.String())
	}
}

// TestS3WithFinalize demonstrates S3 API with Finalize
func TestS3WithFinalize(t *testing.T) {
	r := teapot.New()

	// Service endpoint
	r.GET("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("LIST_BUCKETS"))
	}).Name("service.list").Action("s3:ListAllMyBuckets")

	// Bucket operations
	r.NamedGroup("/{bucket}", "bucket", func(r *teapot.Router) {
		r.PUT("", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("CREATE"))
		}).Name("create").Action("s3:CreateBucket")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("LIST"))
		}).Name("list").Action("s3:ListBucket")

		r.QueryGET("", func(w http.ResponseWriter, r *http.Request) {
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

		if w.Body.String() != tt.expected {
			t.Errorf("%s %s: got %q, want %q", tt.method, tt.path, w.Body.String(), tt.expected)
		}
	}
}
