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

func TestDispatcherSingleRoute(t *testing.T) {
	route := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: testutil.OKResponseHandler,
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestDispatcherQueryMatching(t *testing.T) {
	route1 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       http.HandlerFunc(testutil.StringResponseWriterBuilder("HANDLER1")),
		QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "foo"}},
	}
	route2 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: http.HandlerFunc(testutil.StringResponseWriterBuilder("HANDLER2")),
	}

	dispatcher := &Dispatcher{}
	dispatcher.AddRoute(route1)
	dispatcher.AddRoute(route2)

	// Request with ?foo should match route1
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()
	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, "HANDLER1", w.Body.String())

	// Request without ?foo should match route2
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, "HANDLER2", w.Body.String())
}

func TestDispatcherNoMatch(t *testing.T) {
	route := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       testutil.NoopResponseHandler,
		QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "required"}},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	// Request without required query param
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestDispatcherContextInjection(t *testing.T) {
	var capturedAction, capturedName string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAction = GetAction(r.Context())
		capturedName = GetRouteName(r.Context())
		testutil.OKResponse(w, r)
	})

	route := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
		Action:  "s3:GetBucket",
		Name:    "bucket.get",
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, "s3:GetBucket", capturedAction)
	assert.Equal(t, "bucket.get", capturedName)
}

func TestDispatcherWildcardParams(t *testing.T) {
	var capturedKey string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = chi.URLParam(r, "key")
		_, _ = w.Write([]byte("OK"))
	})

	route := &Route{
		Method:         "GET",
		Pattern:        "/{bucket}/{key:.*}",
		Handler:        handler,
		WildcardParams: map[string]bool{"key": true},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	// Create chi router context
	req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", "mybucket")
	rctx.URLParams.Add("*", "path/to/file.txt")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	dispatcher.ServeHTTP(w, req)

	assert.Equal(t, "path/to/file.txt", capturedKey)
}

func TestDispatcherMiddleware(t *testing.T) {
	middlewareCalled := false

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	route := &Route{
		Method:      "GET",
		Pattern:     "/test",
		Handler:     handler,
		Middlewares: []func(http.Handler) http.Handler{middleware},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	assert.True(t, middlewareCalled, "expected middleware to be called")
}

func TestDispatcherAddRouteSpecificity(t *testing.T) {
	asserts := assert.New(t)
	handler := http.HandlerFunc(testutil.NoopResponse)

	route1 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
	}
	route2 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []dispatch.Matcher{dispatch.QueryExistsMatcher{Key: "foo"}},
	}
	route3 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []dispatch.Matcher{dispatch.QueryValueMatcher{Key: "type", Value: "full"}},
	}

	dispatcher := &Dispatcher{}
	dispatcher.AddRoute(route1)
	dispatcher.AddRoute(route2)
	dispatcher.AddRoute(route3)

	// Routes should be sorted by specificity: route3 (2), route2 (1), route1 (0)
	asserts.Len(dispatcher.Routes, 3)
	asserts.Equal(2, dispatcher.routeSpecificity(dispatcher.Routes[0]))
	asserts.Equal(1, dispatcher.routeSpecificity(dispatcher.Routes[1]))
	asserts.Equal(0, dispatcher.routeSpecificity(dispatcher.Routes[2]))
}

func TestDispatcherUpdateSpecificity(t *testing.T) {
	handler := http.HandlerFunc(testutil.NoopResponse)

	route1 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
	}
	route2 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []dispatch.Matcher{},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route1, route2},
	}

	// Add query matcher to route2
	route2.QueryMatchers = append(route2.QueryMatchers, dispatch.QueryExistsMatcher{Key: "foo"})

	// Update specificity
	dispatcher.UpdateSpecificity()

	// route2 should now be first due to higher specificity
	assert.Equal(t, 1, dispatcher.routeSpecificity(dispatcher.Routes[0]))
}

func TestFindBestDispatcherRoute(t *testing.T) {
	handler := http.HandlerFunc(testutil.NoopResponse)

	t.Run("returns fallback route when available", func(t *testing.T) {
		dispatcher := &Dispatcher{
			Routes: []*Route{
				{Method: "GET", Pattern: "/bucket", Handler: handler, Action: "s3:GetBucketAcl", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "acl"}}},
				{Method: "GET", Pattern: "/bucket", Handler: handler, Action: "s3:ListBucket", QueryMatchers: []dispatch.Matcher{}}, // fallback
				{Method: "GET", Pattern: "/bucket", Handler: handler, Action: "s3:GetBucketVersioning", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "versioning"}}},
			},
		}

		route := FindBestDispatcherRoute(dispatcher)
		assert.NotNil(t, route)
		assert.Equal(t, "s3:ListBucket", route.Action)
		assert.Empty(t, route.QueryMatchers)
	})

	t.Run("returns first route when no fallback", func(t *testing.T) {
		dispatcher := &Dispatcher{
			Routes: []*Route{
				{Method: "GET", Pattern: "/bucket", Handler: handler, Action: "s3:GetBucketAcl", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "acl"}}},
				{Method: "GET", Pattern: "/bucket", Handler: handler, Action: "s3:GetBucketVersioning", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "versioning"}}},
			},
		}

		route := FindBestDispatcherRoute(dispatcher)
		assert.NotNil(t, route)
		assert.Equal(t, "s3:GetBucketAcl", route.Action)
	})

	t.Run("returns nil for empty dispatcher", func(t *testing.T) {
		dispatcher := &Dispatcher{
			Routes: []*Route{},
		}

		route := FindBestDispatcherRoute(dispatcher)
		assert.Nil(t, route)
	})

	t.Run("prefers fallback over first when both exist", func(t *testing.T) {
		dispatcher := &Dispatcher{
			Routes: []*Route{
				{Method: "GET", Pattern: "/test", Handler: handler, Action: "first:WithQuery", QueryMatchers: []dispatch.Matcher{&dispatch.QueryExistsMatcher{Key: "q"}}},
				{Method: "GET", Pattern: "/test", Handler: handler, Action: "second:Fallback", QueryMatchers: []dispatch.Matcher{}},
			},
		}

		route := FindBestDispatcherRoute(dispatcher)
		assert.NotNil(t, route)
		assert.Equal(t, "second:Fallback", route.Action)
	})
}

// TestDispatcherBuildEager exercises Build() directly (currently 0% because
// existing tests only trigger the lazy path through ServeHTTP).
func TestDispatcherBuildEager(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("EAGER"))
	})

	d := &Dispatcher{
		Routes: []*Route{{Method: "GET", Pattern: "/test", Handler: handler}},
	}

	// Eager build — should be safe to call before any request
	d.Build()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	d.ServeHTTP(w, req)
	assert.Equal(t, "EAGER", w.Body.String())

	// Second Build() is a no-op (sync.Once) — must not panic
	d.Build()
}
