package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// tagMiddleware marks responses with a header so tests can see it ran.
func tagMiddleware(tag string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Tag", tag)
			next.ServeHTTP(w, r)
		})
	}
}

// TestResourceRoutesInheritGroupMiddleware pins the current way to put
// middleware on resource routes: a group with Use. It must keep working.
func TestResourceRoutesInheritGroupMiddleware(t *testing.T) {
	r := teapot.New()
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	r.Group("/admin", func(r *teapot.Router) {
		r.Use(tagMiddleware("admin"))
		r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{Index: h})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/photos", nil))
	assert.Equal(t, "admin", rec.Header().Get("X-Tag"))
}
