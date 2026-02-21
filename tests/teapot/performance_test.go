package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestDirectVsQueryPerformance demonstrates the performance difference
func TestDirectVsQueryPerformance(t *testing.T) {
	t.Run("Direct route (GET)", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", testutil.OKResponseHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("Query route (QueryGET)", func(t *testing.T) {
		r := teapot.New()
		r.QueryGET("/test", testutil.OKResponseHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

// TestQueryMethodPanic verifies that .Query() panics on direct routes
func TestQueryMethodPanic(t *testing.T) {
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.Func().GET("/test", testutil.NoopResponse).Query("param")
	})

	if msg == "" {
		t.Error("expected panic when using .Query() with GET")
	}
}

// TestQueryValueMethodPanic verifies that .QueryValue() panics on direct routes
func TestQueryValueMethodPanic(t *testing.T) {
	msg := testutil.CapturePanic(func() {
		r := teapot.New()
		r.Func().GET("/test", testutil.NoopResponse).QueryValue("key", "value")
	})

	if msg == "" {
		t.Error("expected panic when using .QueryValue() with GET")
	}
}

// Benchmark direct route (new fast path)
func BenchmarkDirectRoute(b *testing.B) {
	r := teapot.New()
	r.GET("/test", testutil.OKResponseHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark query route (uses dispatcher)
func BenchmarkQueryRoute(b *testing.B) {
	r := teapot.New()
	r.QueryGET("/test", testutil.OKResponseHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark direct route with context injection
func BenchmarkDirectRouteWithContext(b *testing.B) {
	r := teapot.New()
	r.Func().GET("/test", func(w http.ResponseWriter, r *http.Request) {
		_ = teapot.GetAction(r)
		_ = teapot.GetRouteName(r)
		w.WriteHeader(200)
	}).Name("test").Action("s3:Test")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark query route with context injection
func BenchmarkQueryRouteWithContext(b *testing.B) {
	r := teapot.New()
	r.Func().QueryGET("/test", func(w http.ResponseWriter, r *http.Request) {
		_ = teapot.GetAction(r)
		_ = teapot.GetRouteName(r)
		w.WriteHeader(200)
	}).Name("test").Action("s3:Test")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}
