package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestDirectVsQueryPerformance demonstrates the performance difference
func TestDirectVsQueryPerformance(t *testing.T) {
	t.Run("Direct route (GET)", func(t *testing.T) {
		r := teapot.New()
		r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Query route (QueryGET)", func(t *testing.T) {
		r := teapot.New()
		r.QueryGET("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

// TestQueryMethodPanic verifies that .Query() panics on direct routes
func TestQueryMethodPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when using .Query() with GET")
		}
	}()

	r := teapot.New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {}).Query("param")
}

// TestQueryValueMethodPanic verifies that .QueryValue() panics on direct routes
func TestQueryValueMethodPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when using .QueryValue() with GET")
		}
	}()

	r := teapot.New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {}).QueryValue("key", "value")
}

// Benchmark direct route (new fast path)
func BenchmarkDirectRoute(b *testing.B) {
	r := teapot.New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

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
	r.QueryGET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

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
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
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
	r.QueryGET("/test", func(w http.ResponseWriter, r *http.Request) {
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
