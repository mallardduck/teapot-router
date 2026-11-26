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

// TestResourceRoutesInheritGroupMiddleware pins the group pattern for putting
// middleware on resource routes. It must keep working alongside Middleware.
func TestResourceRoutesInheritGroupMiddleware(t *testing.T) {
	r := teapot.New()

	r.Group("/admin", func(r *teapot.Router) {
		r.Use(tagMiddleware("admin"))
		r.Resource("photos", readOnlyController{})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/photos", nil))
	assert.Equal(t, "admin", rec.Header().Get("X-Tag"))
}
