package teapot

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// --- SetDebugLog / debugLogf (0% / 50%) ---

func TestSetDebugLog(t *testing.T) {
	r := New()
	assert.False(t, r.debugLog)

	ret := r.SetDebugLog(true)
	assert.True(t, r.debugLog)
	assert.Same(t, r, ret) // fluent return

	r.SetDebugLog(false)
	assert.False(t, r.debugLog)
}

func TestDebugLogfEnabled(t *testing.T) {
	r := New()
	r.SetDebugLog(true)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	r.debugLogf("test %s", "message")
	assert.Contains(t, buf.String(), "[teapot-debug] test message")
}

// --- Query / QueryValue panic on isDirect (80% each) ---

func TestQueryPanicOnDirectRoute(t *testing.T) {
	r := New()
	rb := r.GET("/direct-q-panic", func(_ http.ResponseWriter, _ *http.Request) {})
	assert.Panics(t, func() { rb.Query("foo") })
}

func TestQueryValuePanicOnDirectRoute(t *testing.T) {
	r := New()
	rb := r.GET("/direct-qv-panic", func(_ http.ResponseWriter, _ *http.Request) {})
	assert.Panics(t, func() { rb.QueryValue("foo", "bar") })
}

// --- handleDirect: dispatcher-already-exists branch (87.5%) ---
// Exercises the path where QueryGET creates a dispatcher first, then a plain
// GET on the same pattern is added to it as a fallback.

func TestHandleDirectAddedToExistingDispatcher(t *testing.T) {
	r := New()
	r.SetDebugLog(true)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// QueryGET creates the dispatcher
	r.QueryGET("/disp-first", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("query"))
	}).Query("q")

	// Plain GET on the same pattern → added to existing dispatcher as fallback
	r.GET("/disp-first", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("direct"))
	})

	assert.Contains(t, buf.String(), "Adding direct route to existing dispatcher")

	r.Finalize()

	req := httptest.NewRequest("GET", "/disp-first", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "direct", w.Body.String())

	req = httptest.NewRequest("GET", "/disp-first?q=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "query", w.Body.String())
}

// --- handleQuery: promote-direct branch (77.8%) ---
// Exercises the path where a direct GET exists first, then QueryGET on the
// same pattern auto-promotes the direct route into a new dispatcher.

func TestHandleQueryPromotesDirect(t *testing.T) {
	r := New()
	r.SetDebugLog(true)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Direct GET first
	r.GET("/promote-me", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("original-direct"))
	})

	// QueryGET on same pattern → promotes the direct route to dispatcher fallback
	r.QueryGET("/promote-me", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("query-route"))
	}).Query("special")

	assert.Contains(t, buf.String(), "Auto-promoting direct route to dispatcher")

	r.Finalize()

	req := httptest.NewRequest("GET", "/promote-me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "original-direct", w.Body.String())

	req = httptest.NewRequest("GET", "/promote-me?special=yes", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "query-route", w.Body.String())
}

// --- Router.Route() — Chi sub-router wrapper (0%) ---

func TestRouteMethod(t *testing.T) {
	r := New()

	r.Route("/api", func(sub *Router) {
		sub.GET("/hello", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("hello-from-route"))
		}).Name("api.hello")
	})

	r.Finalize()

	req := httptest.NewRequest("GET", "/api/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "hello-from-route", w.Body.String())

	// Route() lets Chi handle the prefix; stored pattern is the inner path
	url, err := r.URL("api.hello")
	require.NoError(t, err)
	assert.Equal(t, "/hello", url)
}

// --- findMatchingRoute branches (46.7%) ---

func TestFindMatchingRouteDirectHit(t *testing.T) {
	r := New()
	r.GET("/find-direct", func(_ http.ResponseWriter, _ *http.Request) {}).Action("direct-action")

	route := r.findMatchingRoute("GET", "/find-direct")
	require.NotNil(t, route)
	assert.Equal(t, "direct-action", route.Action)
}

func TestFindMatchingRouteDispatcherWithFallback(t *testing.T) {
	r := New()
	// First QueryGET has no .Query() → no matchers → acts as fallback
	r.QueryGET("/find-disp-fb", func(_ http.ResponseWriter, _ *http.Request) {}).Action("fallback-action")
	r.QueryGET("/find-disp-fb", func(_ http.ResponseWriter, _ *http.Request) {}).Query("x").Action("query-action")

	route := r.findMatchingRoute("GET", "/find-disp-fb")
	require.NotNil(t, route)
	assert.Equal(t, "fallback-action", route.Action)
}

func TestFindMatchingRouteDispatcherNoFallback(t *testing.T) {
	r := New()
	// Only routes with matchers — no fallback route exists
	r.QueryGET("/find-disp-nofb", func(_ http.ResponseWriter, _ *http.Request) {}).Query("a").Action("first-action")
	r.QueryGET("/find-disp-nofb", func(_ http.ResponseWriter, _ *http.Request) {}).Query("b").Action("second-action")

	route := r.findMatchingRoute("GET", "/find-disp-nofb")
	require.NotNil(t, route)
	// Returns first route in the slice (both have specificity 1)
	assert.NotEmpty(t, route.Action)
}

func TestFindMatchingRouteNoMatch(t *testing.T) {
	r := New()
	r.GET("/only-this", func(_ http.ResponseWriter, _ *http.Request) {})

	assert.Nil(t, r.findMatchingRoute("GET", "/not-registered"))
	assert.Nil(t, r.findMatchingRoute("POST", "/only-this"))
}

// --- matchPattern uncovered branches (81%) ---

func TestMatchPatternWildcardTooFewSegments(t *testing.T) {
	r := New()
	// /a/b/* requires ≥2 segments before wildcard; /a has only 1
	assert.False(t, r.matchPattern("/a/b/*", "/a"))
}

func TestMatchPatternLiteralMismatch(t *testing.T) {
	r := New()
	assert.False(t, r.matchPattern("/users/posts", "/users/comments"))
}

// --- URLParams with no chi RouteContext (87.5%) ---

func TestURLParamsNilRouteContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/anything", nil)
	assert.Nil(t, URLParams(req))
}

// --- Routes() HeaderExistsMatcher / HeaderValueMatcher cases (83.3%) ---

func TestRoutesPopulatesHeaderParams(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/header-info", func(d *DispatchBuilder) {
		d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("hi.default")
		d.When(dispatch.HeaderExists("X-Custom")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("hi.exists")
		d.When(dispatch.HeaderEquals("Content-Type", "application/json")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("hi.value")
	})

	routes := r.Routes()
	require.Len(t, routes, 3)

	byName := make(map[string]RouteInfo)
	for _, rt := range routes {
		byName[rt.Name] = rt
	}

	assert.Empty(t, byName["hi.default"].HeaderParams)

	exists := byName["hi.exists"]
	require.Len(t, exists.HeaderParams, 1)
	assert.Equal(t, "X-Custom", exists.HeaderParams[0].Key)
	assert.Equal(t, "", exists.HeaderParams[0].Value)

	value := byName["hi.value"]
	require.Len(t, value.HeaderParams, 1)
	assert.Equal(t, "Content-Type", value.HeaderParams[0].Key)
	assert.Equal(t, "application/json", value.HeaderParams[0].Value)
}

// --- formatHeaderParams all branches (25%) ---

func TestFormatHeaderParamsEmpty(t *testing.T) {
	assert.Equal(t, "-", formatHeaderParams(nil))
	assert.Equal(t, "-", formatHeaderParams([]HeaderParam{}))
}

func TestFormatHeaderParamsExistenceOnly(t *testing.T) {
	assert.Equal(t, "X-Trace", formatHeaderParams([]HeaderParam{{Key: "X-Trace"}}))
}

func TestFormatHeaderParamsWithValue(t *testing.T) {
	assert.Equal(t, "Content-Type: application/json",
		formatHeaderParams([]HeaderParam{{Key: "Content-Type", Value: "application/json"}}))
}

func TestFormatHeaderParamsMultiple(t *testing.T) {
	params := []HeaderParam{
		{Key: "X-Trace"},
		{Key: "Content-Type", Value: "application/json"},
	}
	assert.Equal(t, "X-Trace, Content-Type: application/json", formatHeaderParams(params))
}

// --- FormatRoutesCompact header branch (94.1%) ---

func TestFormatRoutesCompactWithHeaders(t *testing.T) {
	routes := []RouteInfo{{
		Method:  "GET",
		Pattern: "/test",
		Name:    "test.route",
		HeaderParams: []HeaderParam{
			{Key: "X-Custom", Value: "val"},
		},
	}}

	var buf bytes.Buffer
	err := FormatRoutesCompact(&buf, routes)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "[X-Custom: val]")
}

func TestFormatRoutesCompactWithQueryAndHeaders(t *testing.T) {
	routes := []RouteInfo{{
		Method:  "GET",
		Pattern: "/test",
		Name:    "test.route",
		QueryParams: []QueryParam{
			{Key: "action", Value: "copy"},
		},
		HeaderParams: []HeaderParam{
			{Key: "X-Amz-Copy-Source", Value: "src"},
		},
	}}

	var buf bytes.Buffer
	err := FormatRoutesCompact(&buf, routes)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "/test?action=copy")
	assert.Contains(t, output, "[X-Amz-Copy-Source: src]")
}

// --- Dispatch route without a name (93.3%) ---
// Exercises the "fullName == empty string" guard that skips nameIndex registration.

func TestDispatchRouteUnnamed(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/unnamed", func(d *DispatchBuilder) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("unnamed-ok"))
		})
		// Intentionally no .Name() call
	})

	req := httptest.NewRequest("GET", "/unnamed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "unnamed-ok", w.Body.String())
}

// --- NewListRoutesHandler HTML path with header params ---
// Exercises formatHeaderParams through the HTML template rendering.

func TestNewListRoutesHandlerHTMLWithHeaders(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/hht", func(d *DispatchBuilder) {
		d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("hht.default")
		d.When(dispatch.HeaderEquals("X-Copy", "source")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("hht.copy")
	})

	handler := NewListRoutesHandler(r, nil)

	req := httptest.NewRequest("GET", "/.internal/routes", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "<th>Headers</th>")
	assert.Contains(t, body, "X-Copy: source")
}
