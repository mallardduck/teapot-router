package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestDispatcherAutoPromotion tests automatic promotion from direct to dispatcher-based routing
// to kill mutants in router.go:266-309
func TestDispatcherAutoPromotion(t *testing.T) {
	t.Run("single GET route uses direct routing", func(t *testing.T) {
		r := teapot.New()

		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("direct"))
		})

		r.Finalize()

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "direct", w.Body.String())
	})

	t.Run("QueryGET creates dispatcher immediately", func(t *testing.T) {
		r := teapot.New()

		// Line 266: dispatcherKey := method + ":" + chiPattern
		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("query-full"))
		}).QueryValue("type", "full")

		r.Finalize()

		req := httptest.NewRequest("GET", "/test?type=full", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "query-full", w.Body.String())
	})

	t.Run("GET after QueryGET promotes to dispatcher", func(t *testing.T) {
		r := teapot.New()

		// First register a QueryGET (creates dispatcher)
		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("query-full"))
		}).QueryValue("type", "full")

		// Then add a regular GET (should be added to existing dispatcher)
		// Line 269: if disp, exists := r.dispatchers[dispatcherKey]
		r.GET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("no-query"))
		})

		r.Finalize()

		// Test query route
		req1 := httptest.NewRequest("GET", "/test?type=full", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "query-full", w1.Body.String())

		// Test non-query route (fallback)
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "no-query", w2.Body.String())
	})

	t.Run("multiple Query routes on same pattern use dispatcher", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/object", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("acl"))
		}).Query("acl")

		r.QueryGET("/object", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("tagging"))
		}).Query("tagging")

		r.GET("/object", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("get-object"))
		})

		r.Finalize()

		// Test each route
		tests := []struct {
			query    string
			expected string
		}{
			{"?acl", "acl"},
			{"?tagging", "tagging"},
			{"", "get-object"},
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", "/object"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String(), "query: %s", tt.query)
		}
	})

	t.Run("dispatcher key uses chi pattern not original pattern", func(t *testing.T) {
		r := teapot.New()

		// Pattern with wildcard parameter - should be translated to Chi pattern
		// Line 266: dispatcherKey := method + ":" + chiPattern
		r.QueryGET("/{bucket}/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("acl"))
		}).Query("acl")

		r.GET("/{bucket}/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("get"))
		})

		r.Finalize()

		req1 := httptest.NewRequest("GET", "/mybucket/mykey?acl", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "acl", w1.Body.String())

		req2 := httptest.NewRequest("GET", "/mybucket/mykey", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "get", w2.Body.String())
	})
}

// TestQueryRouteAutoPromotion tests Query() method on QueryGET routes
func TestQueryRouteAutoPromotion(t *testing.T) {
	t.Run("QueryGET with QueryValue matcher", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("base"))
		}).QueryValue("version", "v2")

		r.Finalize()

		// With query param version=v2, should match
		req1 := httptest.NewRequest("GET", "/test?version=v2", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "base", w1.Body.String())

		// Without query param, should not match
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, 404, w2.Code)

		// Verify query params are in route info
		routes := r.Routes()
		assert.Len(t, routes, 1)
		assert.Len(t, routes[0].QueryParams, 1)
	})

	t.Run("QueryGET with Query existence matcher", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}).Query("foo")

		routes := r.Routes()
		assert.Len(t, routes, 1)
		// Query matcher should be registered
		assert.Len(t, routes[0].QueryParams, 1)
	})

	t.Run("multiple Query calls accumulate", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}).QueryValue("foo", "bar").Query("baz")

		routes := r.Routes()
		assert.Len(t, routes, 1)
		assert.Len(t, routes[0].QueryParams, 2)
	})
}

// TestMixedDirectAndQueryRoutes tests mixing direct and query-based routes
func TestMixedDirectAndQueryRoutes(t *testing.T) {
	t.Run("direct route then query route on same pattern", func(t *testing.T) {
		r := teapot.New()

		r.GET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("list"))
		})

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("location"))
		}).Query("location")

		r.Finalize()

		req1 := httptest.NewRequest("GET", "/bucket", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "list", w1.Body.String())

		req2 := httptest.NewRequest("GET", "/bucket?location", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "location", w2.Body.String())
	})

	t.Run("query route then direct route on same pattern", func(t *testing.T) {
		r := teapot.New()

		r.QueryGET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("versioning"))
		}).Query("versioning")

		r.GET("/bucket", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("list"))
		})

		r.Finalize()

		req1 := httptest.NewRequest("GET", "/bucket?versioning", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "versioning", w1.Body.String())

		req2 := httptest.NewRequest("GET", "/bucket", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "list", w2.Body.String())
	})

	t.Run("different methods don't interfere", func(t *testing.T) {
		r := teapot.New()

		// GET with query param
		r.QueryGET("/resource", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("get-filtered"))
		}).QueryValue("filter", "active")

		// POST direct (different method, same pattern - should be separate dispatcher)
		r.POST("/resource", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("post"))
		})

		r.Finalize()

		req1 := httptest.NewRequest("GET", "/resource?filter=active", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "get-filtered", w1.Body.String())

		req2 := httptest.NewRequest("POST", "/resource", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "post", w2.Body.String())
	})
}

// TestDispatcherKeyGeneration tests dispatcher key generation to ensure proper routing
func TestDispatcherKeyGeneration(t *testing.T) {
	t.Run("dispatcher key uses method and chi pattern", func(t *testing.T) {
		r := teapot.New()

		// Different methods, same pattern - should create separate dispatchers
		// Test ARITHMETIC_BASE mutations at router.go:266 (dispatcherKey := method + ":" + chiPattern)
		r.QueryGET("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("get"))
		}).QueryValue("a", "1")

		r.QueryPOST("/test", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("post"))
		}).QueryValue("b", "2")

		r.Finalize()

		// Verify GET works
		req1 := httptest.NewRequest("GET", "/test?a=1", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)
		assert.Equal(t, "get", w1.Body.String())

		// Verify POST works (different dispatcher)
		req2 := httptest.NewRequest("POST", "/test?b=2", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, 200, w2.Code)
		assert.Equal(t, "post", w2.Body.String())

		// Verify wrong method doesn't work
		req3 := httptest.NewRequest("GET", "/test?b=2", nil)
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, req3)
		assert.Equal(t, 404, w3.Code) // GET with POST query param

		req4 := httptest.NewRequest("POST", "/test?a=1", nil)
		w4 := httptest.NewRecorder()
		r.ServeHTTP(w4, req4)
		assert.Equal(t, 404, w4.Code) // POST with GET query param
	})

	t.Run("wildcard patterns generate correct dispatcher key", func(t *testing.T) {
		r := teapot.New()

		// Pattern with wildcard should be translated to Chi's /* pattern
		r.QueryGET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("acl"))
		}).Query("acl")

		r.QueryGET("/{key:.*}", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("tagging"))
		}).Query("tagging")

		r.Finalize()

		req1 := httptest.NewRequest("GET", "/myfile?acl", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		assert.Equal(t, "acl", w1.Body.String())

		req2 := httptest.NewRequest("GET", "/myfile?tagging", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		assert.Equal(t, "tagging", w2.Body.String())
	})
}
