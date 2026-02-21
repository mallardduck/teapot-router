package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Benchmark: Minimal route WITHOUT Finalize (slow path)
func BenchmarkMinimalDirect_NoFinalize(b *testing.B) {
	r := teapot.New()
	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	// NO Finalize() call - uses slow path

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: Minimal route WITH Finalize (fast path)
func BenchmarkMinimalDirect_WithFinalize(b *testing.B) {
	r := teapot.New()
	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r.Finalize() // Optimize!

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: Chi baseline for comparison
func BenchmarkMinimalDirect_Chi(b *testing.B) {
	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
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

// Benchmark: With Name/Action WITHOUT Finalize
func BenchmarkWithMeta_NoFinalize(b *testing.B) {
	r := teapot.New()
	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).Name("test").Action("s3:Test")
	// NO Finalize() - runtime checks

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: With Name/Action WITH Finalize
func BenchmarkWithMeta_WithFinalize(b *testing.B) {
	r := teapot.New()
	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).Name("test").Action("s3:Test")
	r.Finalize() // Pre-compute!

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: With middleware WITHOUT Finalize
func BenchmarkWithMiddleware_NoFinalize(b *testing.B) {
	r := teapot.New()

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).With(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: With middleware WITH Finalize
func BenchmarkWithMiddleware_WithFinalize(b *testing.B) {
	r := teapot.New()

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).With(mw)

	r.Finalize() // Pre-apply middleware!

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark: Complete scenario (Name + Action + Middleware)
func BenchmarkComplete_NoFinalize(b *testing.B) {
	r := teapot.New()

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = teapot.GetAction(r)
		_ = teapot.GetRouteName(r)
		w.WriteHeader(200)
	})).Name("test").Action("s3:Test").With(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkComplete_WithFinalize(b *testing.B) {
	r := teapot.New()

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	r.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = teapot.GetAction(r)
		_ = teapot.GetRouteName(r)
		w.WriteHeader(200)
	})).Name("test").Action("s3:Test").With(mw)

	r.Finalize() // Optimize everything!

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}
