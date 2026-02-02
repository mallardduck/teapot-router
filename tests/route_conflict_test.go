package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestRouteConflict verifies auto-promotion when mixing direct and query routes
func TestRouteConflict(t *testing.T) {
	r := teapot.New()

	// Register direct route first
	r.GET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DIRECT"))
	}).Name("direct")

	// Then register query route on same path
	// This should trigger auto-promotion to dispatcher
	r.QueryGET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("QUERY"))
	}).Query("test").Name("query")

	// Test direct route (without query param) - should still work via dispatcher
	req := httptest.NewRequest("GET", "/mybucket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	t.Logf("GET /mybucket (no query): %s", w.Body.String())
	assert.Equal(t, "DIRECT", w.Body.String(), "auto-promotion should preserve direct route")

	// Test query route (with query param)
	req = httptest.NewRequest("GET", "/mybucket?test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	t.Logf("GET /mybucket?test: %s", w.Body.String())
	assert.Equal(t, "QUERY", w.Body.String())
}

// TestQueryRouteFallback verifies that QueryGET without .Query() acts as default
func TestQueryRouteFallback(t *testing.T) {
	r := teapot.New()

	// Register QueryGET without any query matchers (should be fallback)
	r.QueryGET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("FALLBACK"))
	}).Name("fallback")

	// Register QueryGET with query matcher
	r.QueryGET("/{bucket}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("WITH_QUERY"))
	}).Query("test").Name("with-query")

	// Test without query param (should hit fallback)
	req := httptest.NewRequest("GET", "/mybucket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	t.Logf("GET /mybucket (no query): %s", w.Body.String())
	assert.Equal(t, "FALLBACK", w.Body.String())

	// Test with query param
	req = httptest.NewRequest("GET", "/mybucket?test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	t.Logf("GET /mybucket?test: %s", w.Body.String())
	assert.Equal(t, "WITH_QUERY", w.Body.String())
}
