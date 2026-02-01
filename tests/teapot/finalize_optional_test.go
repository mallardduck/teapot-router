package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestWithoutFinalize verifies that routes work WITHOUT calling Finalize()
// Finalize is purely an optimization - routes must work without it
func TestWithoutFinalize(t *testing.T) {
	r := teapot.New()

	// Register various route types
	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SIMPLE"))
	})

	r.GET("/with-name", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		w.Write([]byte("NAME:" + name))
	}).Name("test.name")

	r.GET("/with-action", func(w http.ResponseWriter, r *http.Request) {
		action := teapot.GetAction(r)
		w.Write([]byte("ACTION:" + action))
	}).Action("s3:TestAction")

	r.GET("/with-both", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)
		w.Write([]byte(name + ":" + action))
	}).Name("test.both").Action("s3:Both")

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("MIDDLEWARE"))
	}).With(mw)

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BASE"))
	})

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PARAM"))
	}).Query("test")

	// NO Finalize() call!
	// Verify router is not finalized
	if r.IsFinalized() {
		t.Fatal("router should not be finalized")
	}

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

			if w.Body.String() != tt.expected {
				t.Errorf("got %q, want %q", w.Body.String(), tt.expected)
			}

			if tt.header != "" {
				if got := w.Header().Get("X-Middleware"); got != tt.header {
					t.Errorf("middleware header: got %q, want %q", got, tt.header)
				}
			}
		})
	}
}

// TestWithFinalize verifies the same routes work WITH Finalize()
func TestWithFinalize(t *testing.T) {
	r := teapot.New()

	// Same routes as above
	r.GET("/simple", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SIMPLE"))
	})

	r.GET("/with-name", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		w.Write([]byte("NAME:" + name))
	}).Name("test.name")

	r.GET("/with-action", func(w http.ResponseWriter, r *http.Request) {
		action := teapot.GetAction(r)
		w.Write([]byte("ACTION:" + action))
	}).Action("s3:TestAction")

	r.GET("/with-both", func(w http.ResponseWriter, r *http.Request) {
		name := teapot.GetRouteName(r)
		action := teapot.GetAction(r)
		w.Write([]byte(name + ":" + action))
	}).Name("test.both").Action("s3:Both")

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("MIDDLEWARE"))
	}).With(mw)

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BASE"))
	})

	r.QueryGET("/query", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PARAM"))
	}).Query("test")

	// Call Finalize()
	r.Finalize()

	// Verify router is finalized
	if !r.IsFinalized() {
		t.Fatal("router should be finalized")
	}

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

			if w.Body.String() != tt.expected {
				t.Errorf("got %q, want %q", w.Body.String(), tt.expected)
			}

			if tt.header != "" {
				if got := w.Header().Get("X-Middleware"); got != tt.header {
					t.Errorf("middleware header: got %q, want %q", got, tt.header)
				}
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
			w.Write([]byte(name + ":" + action))
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

	if w1.Body.String() != w2.Body.String() {
		t.Errorf("Results differ: without Finalize=%q, with Finalize=%q",
			w1.Body.String(), w2.Body.String())
	}

	expected := "test:s3:Test"
	if w1.Body.String() != expected {
		t.Errorf("Incorrect result: got %q, want %q", w1.Body.String(), expected)
	}
}
