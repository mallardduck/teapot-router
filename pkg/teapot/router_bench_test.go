package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Benchmark teapot router vs raw Chi for simple routes (no query multiplexing)
func BenchmarkSimpleRoute_Teapot(b *testing.B) {
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

func BenchmarkSimpleRoute_Chi(b *testing.B) {
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

// Benchmark with URL parameters
func BenchmarkURLParams_Teapot(b *testing.B) {
	r := teapot.New()
	r.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = teapot.URLParam(r, "id")
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkURLParams_Chi(b *testing.B) {
	r := chi.NewRouter()
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = chi.URLParam(r, "id")
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark query multiplexing (unique to teapot)
func BenchmarkQueryMultiplexing_SingleRoute(b *testing.B) {
	r := teapot.New()
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/bucket", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkQueryMultiplexing_ThreeRoutes(b *testing.B) {
	r := teapot.New()
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Query("acl")
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Query("versioning")

	req := httptest.NewRequest("GET", "/bucket", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkQueryMultiplexing_ThreeRoutes_MatchLast(b *testing.B) {
	r := teapot.New()
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Query("acl")
	r.GET("/bucket", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Query("versioning")

	req := httptest.NewRequest("GET", "/bucket?versioning", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// Benchmark context injection overhead
func BenchmarkContextInjection_Teapot(b *testing.B) {
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

func BenchmarkContextInjection_Chi(b *testing.B) {
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
