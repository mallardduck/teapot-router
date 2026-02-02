package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Benchmark truly minimal routes
func BenchmarkMinimal_Chi(b *testing.B) {
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

func BenchmarkMinimal_TeapotDirect(b *testing.B) {
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

func BenchmarkMinimal_TeapotQuery(b *testing.B) {
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

// Benchmark with Name/Action (common case)
func BenchmarkWithMeta_Chi(b *testing.B) {
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

func BenchmarkWithMeta_TeapotDirect(b *testing.B) {
	r := teapot.New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
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

func BenchmarkWithMeta_TeapotQuery(b *testing.B) {
	r := teapot.New()
	r.QueryGET("/test", func(w http.ResponseWriter, r *http.Request) {
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
