package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestWithoutFinalize verifies that routes work WITHOUT calling Finalize()
// Finalize is purely an optimization - routes must work without it
func TestWithoutFinalize(t *testing.T) {
	r := teapot.New()

	// Register various route types
	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("SIMPLE"))
	})

	r.GET("/with-name", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		_, _ = w.Write([]byte("NAME:" + name)) //nolint:gosec // G705: name is internal router metadata, not user input
	}).Name("test.name")

	r.GET("/with-action", func(w http.ResponseWriter, r *http.Request) {
		action := teapot.GetAction(r)
		_, _ = w.Write([]byte("ACTION:" + action)) //nolint:gosec // G705: action is internal router metadata, not user input
	}).Action("s3:TestAction")

	r.GET("/with-both", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)
		_, _ = w.Write([]byte(name + ":" + action)) //nolint:gosec // G705: name/action are internal router metadata, not user input
	}).Name("test.both").Action("s3:Both")

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("MIDDLEWARE"))
	}).With(mw)

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("BASE"))
	})

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PARAM"))
	}).Query("test")

	// NO Finalize() call!
	// Verify router is not finalized
	assert.False(t, r.IsFinalized(), "router should not be finalized")

	// Test all routes work correctly
	tests := []struct {
		path     string
		expected string
		header   string
	}{
		{"/simple", "SIMPLE", ""},
		{"/with-name", "NAME:test.name", ""},
		{"/with-action", "ACTION:s3:TestAction", ""},
		{"/with-both", "test.both:s3:Both", ""},
		{"/with-middleware", "MIDDLEWARE", "applied"},
		{"/query", "BASE", ""},
		{"/query?test", "PARAM", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Body.String())

			if tt.header != "" {
				assert.Equal(t, tt.header, w.Header().Get("X-Middleware"), "middleware header")
			}
		})
	}
}

// TestWithFinalize verifies the same routes work WITH Finalize()
func TestWithFinalize(t *testing.T) {
	r := teapot.New()

	// Same routes as above
	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("SIMPLE"))
	})

	r.GET("/with-name", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		_, _ = w.Write([]byte("NAME:" + name)) //nolint:gosec // G705: name is internal router metadata, not user input
	}).Name("test.name")

	r.GET("/with-action", func(w http.ResponseWriter, r *http.Request) {
		action := teapot.GetAction(r)
		_, _ = w.Write([]byte("ACTION:" + action)) //nolint:gosec // G705: action is internal router metadata, not user input
	}).Action("s3:TestAction")

	r.GET("/with-both", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)
		_, _ = w.Write([]byte(name + ":" + action)) //nolint:gosec // G705: name/action are internal router metadata, not user input
	}).Name("test.both").Action("s3:Both")

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("MIDDLEWARE"))
	}).With(mw)

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("BASE"))
	})

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PARAM"))
	}).Query("test")

	// Call Finalize()
	r.Finalize()

	// Verify router is finalized
	assert.True(t, r.IsFinalized(), "router should be finalized")

	// Test all routes work correctly (same tests as without Finalize)
	tests := []struct {
		path     string
		expected string
		header   string
	}{
		{"/simple", "SIMPLE", ""},
		{"/with-name", "NAME:test.name", ""},
		{"/with-action", "ACTION:s3:TestAction", ""},
		{"/with-both", "test.both:s3:Both", ""},
		{"/with-middleware", "MIDDLEWARE", "applied"},
		{"/query", "BASE", ""},
		{"/query?test", "PARAM", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Body.String())

			if tt.header != "" {
				assert.Equal(t, tt.header, w.Header().Get("X-Middleware"), "middleware header")
			}
		})
	}
}

// TestFinalizeIsPureOptimization proves Finalize is purely a performance optimization
func TestFinalizeIsPureOptimization(t *testing.T) {
	// Create two identical routers
	r1 := teapot.New()
	r2 := teapot.New()

	// Register same routes on both
	setupRouter := func(r *teapot.Router) {
		r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
			name := teapot.GetRouteName(r)
			action := teapot.GetAction(r)
			_, _ = w.Write([]byte(name + ":" + action))
		}).Name("test").Action("s3:Test")
	}

	setupRouter(r1)
	setupRouter(r2)

	// Only finalize r2
	r2.Finalize()

	// Both should produce identical results
	req := httptest.NewRequest("GET", "/test", nil)

	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, req)

	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req)

	assert.Equal(t, w2.Body.String(), w1.Body.String(), "Results should match regardless of Finalize")

	expected := "test:s3:Test"
	assert.Equal(t, expected, w1.Body.String())
}
