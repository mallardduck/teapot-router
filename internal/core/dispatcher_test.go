package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDispatcherSingleRoute(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	route := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", w.Body.String())
	}
}

func TestDispatcherQueryMatching(t *testing.T) {
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HANDLER1"))
	})
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HANDLER2"))
	})

	route1 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler1,
		QueryMatchers: []QueryMatcher{QueryExistsMatcher{Key: "foo"}},
	}
	route2 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler2,
	}

	dispatcher := &Dispatcher{}
	dispatcher.AddRoute(route1)
	dispatcher.AddRoute(route2)

	// Request with ?foo should match route1
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()
	dispatcher.ServeHTTP(w, req)

	if w.Body.String() != "HANDLER1" {
		t.Errorf("expected HANDLER1, got %q", w.Body.String())
	}

	// Request without ?foo should match route2
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	dispatcher.ServeHTTP(w, req)

	if w.Body.String() != "HANDLER2" {
		t.Errorf("expected HANDLER2, got %q", w.Body.String())
	}
}

func TestDispatcherNoMatch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	route := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []QueryMatcher{QueryExistsMatcher{Key: "required"}},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route},
	}

	// Request without required query param
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	dispatcher.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDispatcherContextInjection(t *testing.T) {
	var capturedAction, capturedName string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAction = GetAction(r.Context())
		capturedName = GetRouteName(r.Context())
		w.Write([]byte("OK"))
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

	if capturedAction != "s3:GetBucket" {
		t.Errorf("expected action 's3:GetBucket', got %q", capturedAction)
	}
	if capturedName != "bucket.get" {
		t.Errorf("expected name 'bucket.get', got %q", capturedName)
	}
}

func TestDispatcherWildcardParams(t *testing.T) {
	var capturedKey string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = chi.URLParam(r, "key")
		w.Write([]byte("OK"))
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

	if capturedKey != "path/to/file.txt" {
		t.Errorf("expected key 'path/to/file.txt', got %q", capturedKey)
	}
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
		w.Write([]byte("OK"))
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

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
}

func TestDispatcherAddRouteSpecificity(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	route1 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
	}
	route2 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []QueryMatcher{QueryExistsMatcher{Key: "foo"}},
	}
	route3 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []QueryMatcher{QueryValueMatcher{Key: "type", Value: "full"}},
	}

	dispatcher := &Dispatcher{}
	dispatcher.AddRoute(route1)
	dispatcher.AddRoute(route2)
	dispatcher.AddRoute(route3)

	// Routes should be sorted by specificity: route3 (2), route2 (1), route1 (0)
	if len(dispatcher.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(dispatcher.Routes))
	}
	if dispatcher.routeSpecificity(dispatcher.Routes[0]) != 2 {
		t.Errorf("expected first route specificity 2, got %d", dispatcher.routeSpecificity(dispatcher.Routes[0]))
	}
	if dispatcher.routeSpecificity(dispatcher.Routes[1]) != 1 {
		t.Errorf("expected second route specificity 1, got %d", dispatcher.routeSpecificity(dispatcher.Routes[1]))
	}
	if dispatcher.routeSpecificity(dispatcher.Routes[2]) != 0 {
		t.Errorf("expected third route specificity 0, got %d", dispatcher.routeSpecificity(dispatcher.Routes[2]))
	}
}

func TestDispatcherUpdateSpecificity(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	route1 := &Route{
		Method:  "GET",
		Pattern: "/test",
		Handler: handler,
	}
	route2 := &Route{
		Method:        "GET",
		Pattern:       "/test",
		Handler:       handler,
		QueryMatchers: []QueryMatcher{},
	}

	dispatcher := &Dispatcher{
		Routes: []*Route{route1, route2},
	}

	// Add query matcher to route2
	route2.QueryMatchers = append(route2.QueryMatchers, QueryExistsMatcher{Key: "foo"})

	// Update specificity
	dispatcher.UpdateSpecificity()

	// route2 should now be first due to higher specificity
	if dispatcher.routeSpecificity(dispatcher.Routes[0]) != 1 {
		t.Errorf("expected first route specificity 1, got %d", dispatcher.routeSpecificity(dispatcher.Routes[0]))
	}
}
