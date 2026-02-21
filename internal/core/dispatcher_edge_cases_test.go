package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// TestDispatcherFastPathEdgeCases tests the fast path condition that gremlins
// identified as LIVED (line 19:19 - CONDITIONALS_NEGATION)
func TestDispatcherFastPathEdgeCases(t *testing.T) {
	t.Run("exactly one route with exactly zero query matchers", func(t *testing.T) {
		// This should trigger fast path
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("FAST"))
		})

		route := &Route{
			Method:         "GET",
			Pattern:        "/test",
			Handler:        handler,
			QueryMatchers:  []dispatch.Matcher{}, // Explicitly empty
			WildcardParams: map[string]bool{},
		}

		dispatcher := &Dispatcher{
			Routes: []*Route{route},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, "FAST", w.Body.String())
	})

	t.Run("exactly one route with one query matcher", func(t *testing.T) {
		// This should NOT trigger fast path (tests the negation)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("SLOW"))
		})

		route := &Route{
			Method:        "GET",
			Pattern:       "/test",
			Handler:       handler,
			QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "foo"}},
		}

		dispatcher := &Dispatcher{
			Routes: []*Route{route},
		}

		req := httptest.NewRequest("GET", "/test?foo=bar", nil)
		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, "SLOW", w.Body.String())
	})

	t.Run("zero routes", func(t *testing.T) {
		// Edge case: empty dispatcher
		dispatcher := &Dispatcher{
			Routes: []*Route{},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
	})

	t.Run("two routes with no query matchers", func(t *testing.T) {
		// Should NOT trigger fast path (len != 1)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("HANDLER"))
		})

		route1 := &Route{Method: "GET", Pattern: "/test", Handler: handler}
		route2 := &Route{Method: "POST", Pattern: "/test", Handler: handler}

		dispatcher := &Dispatcher{
			Routes: []*Route{route1, route2},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, "HANDLER", w.Body.String())
	})
}

// TestDispatcherBoundaryConditions tests boundary conditions for CONDITIONALS_BOUNDARY
// mutations that lived (lines 64:28, 92:42, 108:42)
func TestDispatcherBoundaryConditions(t *testing.T) {
	t.Run("route with exactly one wildcard param", func(t *testing.T) {
		var capturedKey string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedKey = chi.URLParam(r, "key")
			_, _ = w.Write([]byte("OK"))
		})

		route := &Route{
			Method:         "GET",
			Pattern:        "/{key:.*}",
			Handler:        handler,
			WildcardParams: map[string]bool{"key": true}, // len = 1
		}

		dispatcher := &Dispatcher{Routes: []*Route{route}}

		req := httptest.NewRequest("GET", "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "test/path")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, "test/path", capturedKey)
	})

	t.Run("route with multiple wildcard params", func(t *testing.T) {
		// Edge case: multiple wildcard params (len > 1)
		var capturedKey1, capturedKey2 string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedKey1 = chi.URLParam(r, "key1")
			capturedKey2 = chi.URLParam(r, "key2")
			_, _ = w.Write([]byte("OK"))
		})

		route := &Route{
			Method:  "GET",
			Pattern: "/*",
			Handler: handler,
			WildcardParams: map[string]bool{
				"key1": true,
				"key2": true,
			}, // len = 2
		}

		dispatcher := &Dispatcher{Routes: []*Route{route}}

		req := httptest.NewRequest("GET", "/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "test/path")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, "test/path", capturedKey1)
		assert.Equal(t, "test/path", capturedKey2)
	})

	t.Run("route with exactly one middleware", func(t *testing.T) {
		// Tests boundary for len(rt.Middlewares) - 1; i >= 0 (line 79-80)
		callOrder := []string{}
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, "middleware")
				next.ServeHTTP(w, r)
			})
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "handler")
		})

		route := &Route{
			Method:      "GET",
			Pattern:     "/test",
			Handler:     handler,
			Middlewares: []func(http.Handler) http.Handler{mw}, // len = 1
		}

		dispatcher := &Dispatcher{Routes: []*Route{route}}
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		dispatcher.ServeHTTP(w, req)

		assert.Equal(t, []string{"middleware", "handler"}, callOrder)
	})

	t.Run("dispatcher with exactly one route", func(t *testing.T) {
		// Tests specificity sorting with single route
		route := &Route{
			Method:        "GET",
			Pattern:       "/test",
			Handler:       http.HandlerFunc(testutil.NoopResponse),
			QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "foo"}},
		}

		dispatcher := &Dispatcher{}
		dispatcher.AddRoute(route)

		assert.Len(t, dispatcher.Routes, 1)
		assert.Equal(t, 1, dispatcher.routeSpecificity(dispatcher.Routes[0]))
	})

	t.Run("routes with specificity values at boundaries", func(t *testing.T) {
		// Test sorting with specificity 0, 1, 2, etc.
		route0 := &Route{Method: "GET", Pattern: "/test", Handler: http.HandlerFunc(testutil.NoopResponse)}
		route1 := &Route{
			Method:        "GET",
			Pattern:       "/test",
			Handler:       http.HandlerFunc(testutil.NoopResponse),
			QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "a"}},
		}
		route2 := &Route{
			Method:  "GET",
			Pattern: "/test",
			Handler: http.HandlerFunc(testutil.NoopResponse),
			QueryMatchers: []dispatch.Matcher{
				dispatch.QueryExistsMatcher{Key: "a"},
				dispatch.QueryExistsMatcher{Key: "b"},
			},
		}

		dispatcher := &Dispatcher{}
		dispatcher.AddRoute(route0)
		dispatcher.AddRoute(route1)
		dispatcher.AddRoute(route2)

		// Should be sorted: route2 (2), route1 (1), route0 (0)
		assert.Equal(t, 2, dispatcher.routeSpecificity(dispatcher.Routes[0]))
		assert.Equal(t, 1, dispatcher.routeSpecificity(dispatcher.Routes[1]))
		assert.Equal(t, 0, dispatcher.routeSpecificity(dispatcher.Routes[2]))
	})
}

// TestDispatcherNilAndEmptyContexts tests nil chi.RouteContext edge cases
func TestDispatcherNilAndEmptyContexts(t *testing.T) {
	t.Run("nil chi route context", func(t *testing.T) {
		route := &Route{
			Method:         "GET",
			Pattern:        "/*",
			Handler:        http.HandlerFunc(testutil.StringResponseWriterBuilder("OK")),
			WildcardParams: map[string]bool{"key": true},
		}

		dispatcher := &Dispatcher{Routes: []*Route{route}}

		// Request without chi route context
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		dispatcher.ServeHTTP(w, req)

		// Should not crash, wildcard handling is skipped
		assert.Equal(t, 200, w.Code)
	})
}
