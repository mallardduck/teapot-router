package teapot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHeadPinExplicitRouteAndGet pins behavior that must not change when HEAD
// falls back to GET: GET still works, and an explicit HEAD route still wins.
func TestHeadPinExplicitRouteAndGet(t *testing.T) {
	r := New()
	r.GET("/items/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ran", "get")
		_, _ = w.Write([]byte("body"))
	}))
	r.HEAD("/things/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ran", "head")
	}))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/items/1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "body", rec.Body.String())

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("HEAD", "/things/1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "head", rec.Header().Get("X-Ran"))
}

// HEAD on a GET-only route should reach the GET handler, as HTTP and Laravel
// expect. Intentionally red until the fallback is implemented.
func TestHeadFallsBackToGetRoute(t *testing.T) {
	r := New()
	r.GET("/items/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ran", "get")
	}))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("HEAD", "/items/1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "get", rec.Header().Get("X-Ran"))
}

// An explicit HEAD route, registered after GET on the same pattern, must still
// win over the fallback.
func TestExplicitHeadWinsOverGetFallback(t *testing.T) {
	r := New()
	r.GET("/things/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ran", "get")
	}))
	r.HEAD("/things/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ran", "head")
	}))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("HEAD", "/things/1", nil))
	assert.Equal(t, "head", rec.Header().Get("X-Ran"))
}
