package teapot_test

import (
	"net/http"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Test: Finalize optimizes minimal route
func TestFinalizeMinimalRoute(t *testing.T) {
	r := teapot.New()

	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// Finalize to trigger optimization
	r.Finalize()

	// Route should still work after finalization
	assertEqual(t, request(t, r, "GET", "/test").Body.String(), "OK")
}

// Test: Finalize optimizes route with action and name
func TestFinalizeWithActionAndName(t *testing.T) {
	r := teapot.New()

	var capturedAction, capturedName string
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		capturedName = teapot.GetRouteName(r)
		_, _ = w.Write([]byte("OK"))
	}).Name("bucket.list").Action("s3:ListBucket")

	// Finalize to trigger optimization
	r.Finalize()

	// Test that action and name are still injected after finalization
	request(t, r, "GET", "/bucket")
	assertEqual(t, capturedAction, "s3:ListBucket")
	assertEqual(t, capturedName, "bucket.list")
}

// Test: Finalize optimizes route with middleware
func TestFinalizeWithMiddleware(t *testing.T) {
	r := teapot.New()

	middlewareCalled := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/protected", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}).With(mw)

	// Finalize to trigger optimization
	r.Finalize()

	// Test that middleware is still called after finalization
	middlewareCalled = false
	request(t, r, "GET", "/protected")
	if !middlewareCalled {
		t.Error("middleware was not called after finalization")
	}
}

// Test: Finalize optimizes route with wildcard params
func TestFinalizeWithWildcardParams(t *testing.T) {
	r := teapot.New()

	var capturedKey string
	r.GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
		capturedKey = teapot.URLParam(r, "key")
		_, _ = w.Write([]byte("OK"))
	})

	// Finalize to trigger optimization
	r.Finalize()

	// Test that wildcard params are still accessible after finalization
	request(t, r, "GET", "/mybucket/path/to/file.txt")
	assertEqual(t, capturedKey, "path/to/file.txt")
}

// Test: Finalize optimizes route with everything
func TestFinalizeWithEverything(t *testing.T) {
	r := teapot.New()

	var capturedAction, capturedName, capturedKey string
	middlewareCalled := false

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		capturedName = teapot.GetRouteName(r)
		capturedKey = teapot.URLParam(r, "key")
		_, _ = w.Write([]byte("OK"))
	}).Name("object.get").Action("s3:GetObject").With(mw)

	// Finalize to trigger optimization
	r.Finalize()

	// Test that everything works after finalization
	middlewareCalled = false
	request(t, r, "GET", "/mybucket/path/to/file.txt")

	if !middlewareCalled {
		t.Error("middleware was not called")
	}
	assertEqual(t, capturedAction, "s3:GetObject")
	assertEqual(t, capturedName, "object.get")
	assertEqual(t, capturedKey, "path/to/file.txt")
}

// Test: Routes work both before and after finalization
func TestRouteBeforeAndAfterFinalize(t *testing.T) {
	r := teapot.New()

	var capturedAction string
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		_, _ = w.Write([]byte("OK"))
	}).Action("test:Action")

	// Test before finalization (slow path)
	capturedAction = ""
	request(t, r, "GET", "/test")
	assertEqual(t, capturedAction, "test:Action")

	// Finalize
	r.Finalize()

	// Test after finalization (fast path)
	capturedAction = ""
	request(t, r, "GET", "/test")
	assertEqual(t, capturedAction, "test:Action")
}

// Test: Query routes are optimized after finalization
func TestFinalizeQueryRoutes(t *testing.T) {
	r := teapot.New()

	var capturedAction string

	r.QueryGET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		_, _ = w.Write([]byte("LIST"))
	}).Name("bucket.list").Action("s3:ListBucket")

	r.QueryGET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		_, _ = w.Write([]byte("ACL"))
	}).Name("bucket.acl").Action("s3:GetBucketAcl").Query("acl")

	// Finalize
	r.Finalize()

	// Test both query routes work after finalization
	capturedAction = ""
	assertEqual(t, request(t, r, "GET", "/bucket").Body.String(), "LIST")
	assertEqual(t, capturedAction, "s3:ListBucket")

	capturedAction = ""
	assertEqual(t, request(t, r, "GET", "/bucket?acl").Body.String(), "ACL")
	assertEqual(t, capturedAction, "s3:GetBucketAcl")
}

// Test: Multiple finalizations should be safe
func TestMultipleFinalize(t *testing.T) {
	r := teapot.New()

	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// Multiple calls to Finalize should be safe
	r.Finalize()
	r.Finalize()
	r.Finalize()

	// Route should still work
	assertEqual(t, request(t, r, "GET", "/test").Body.String(), "OK")
}

// Test: Finalize with groups
func TestFinalizeWithGroups(t *testing.T) {
	r := teapot.New()

	r.Group("/api", func(r *teapot.Router) {
		r.GET("/users", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("USERS"))
		}).Name("api.users")
	})

	// Finalize
	r.Finalize()

	// Routes in groups should work after finalization
	assertEqual(t, request(t, r, "GET", "/api/users").Body.String(), "USERS")

	// URL generation should still work
	url := r.MustURL("api.users")
	assertEqual(t, url, "/api/users")
}

// Test: Finalize with only middleware (no action/name)
func TestFinalizeMiddlewareOnly(t *testing.T) {
	r := teapot.New()

	callCount := 0
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}).With(mw)

	// Finalize
	r.Finalize()

	// Test middleware is called
	callCount = 0
	request(t, r, "GET", "/test")
	assertEqual(t, callCount, 1)
}

// Test: Finalize with only action (no name/middleware/wildcards)
func TestFinalizeActionOnly(t *testing.T) {
	r := teapot.New()

	var capturedAction string
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedAction = teapot.GetAction(r)
		_, _ = w.Write([]byte("OK"))
	}).Action("test:Action")

	// Finalize
	r.Finalize()

	// Test action is injected
	capturedAction = ""
	request(t, r, "GET", "/test")
	assertEqual(t, capturedAction, "test:Action")
}

// Test: Finalize with only name (no action/middleware/wildcards)
func TestFinalizeNameOnly(t *testing.T) {
	r := teapot.New()

	var capturedName string
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedName = teapot.GetRouteName(r)
		_, _ = w.Write([]byte("OK"))
	}).Name("test.route")

	// Finalize
	r.Finalize()

	// Test name is injected
	capturedName = ""
	request(t, r, "GET", "/test")
	assertEqual(t, capturedName, "test.route")
}
