package teapot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/internal/testutil"
)

// --- matchPattern boundaries (wildcard and segment counts) ---

func TestMatchPatternWildcardBoundaries(t *testing.T) {
	r := New()
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"wildcard with exact prefix length", "/a/*", "/a", true},
		{"wildcard with one extra segment", "/a/*", "/a/b", true},
		{"wildcard with deep path", "/a/*", "/a/b/c/d", true},
		{"root wildcard matches root", "/*", "/", true},
		{"root wildcard matches deep path", "/*", "/x/y", true},
		{"two-segment prefix too short", "/a/b/*", "/a", false},
		{"two-segment prefix exact", "/a/b/*", "/a/b", true},
		{"wildcard literal mismatch", "/a/*", "/b/c", false},
		{"param segment matches any value", "/users/{id}/posts", "/users/42/posts", true},
		{"param segment count mismatch", "/users/{id}", "/users/42/extra", false},
		{"literal without params is exact only", "/a/b", "/a/c", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.matchPattern(tt.pattern, tt.path))
		})
	}
}

// --- findMatchingRoute scoring: literal beats parameterised, more literals beat fewer ---

func TestFindMatchingRoutePrefersLiteralPattern(t *testing.T) {
	r := New()
	noop := http.HandlerFunc(testutil.NoopResponse)
	r.GET("/a/{x}", noop).Name("param")
	r.GET("/a/c", noop).Name("literal")

	rt := r.findMatchingRoute("GET", "/a/c")
	require.NotNil(t, rt)
	assert.Equal(t, "literal", rt.Name)
}

func TestFindMatchingRoutePrefersMoreLiteralSegments(t *testing.T) {
	r := New()
	noop := http.HandlerFunc(testutil.NoopResponse)
	r.GET("/a/{x}/{y}", noop).Name("two-params")
	r.GET("/a/{x}/c", noop).Name("one-literal-tail")

	// Both patterns match /a/1/c; the one with more literal segments wins.
	rt := r.findMatchingRoute("GET", "/a/1/c")
	require.NotNil(t, rt)
	assert.Equal(t, "one-literal-tail", rt.Name)
}

func TestFindMatchingRouteSingleMatchReturned(t *testing.T) {
	r := New()
	noop := http.HandlerFunc(testutil.NoopResponse)
	r.GET("/only/{x}", noop).Name("only")

	rt := r.findMatchingRoute("GET", "/only/7")
	require.NotNil(t, rt)
	assert.Equal(t, "only", rt.Name)
	assert.Nil(t, r.findMatchingRoute("POST", "/only/7"))
}

func TestFindMatchingRouteDispatcherFallbackWithoutUnconditionalRoute(t *testing.T) {
	r := New()
	noop := http.HandlerFunc(testutil.NoopResponse)
	r.QueryGET("/q/{id}", noop).Query("acl").Name("acl")
	r.QueryGET("/q/{id}", noop).Query("policy").Name("policy")

	rt := r.findMatchingRoute("GET", "/q/9")
	require.NotNil(t, rt, "dispatcher with only query-matched routes should still resolve a fallback")
	assert.Contains(t, []string{"acl", "policy"}, rt.Name)
}

// --- Route name suffix handling on Mount ---

func TestMountNameAddsSeparatorOnce(t *testing.T) {
	for _, name := range []string{"api", "api."} {
		t.Run(name, func(t *testing.T) {
			sub := New()
			sub.GET("/items", http.HandlerFunc(testutil.NoopResponse)).Name("list")

			r := New()
			r.Mount("/api", sub).Name(name)

			assert.Equal(t, "/api/items", r.MustURL("api.list"))
		})
	}
}

func TestMountBuilderNameWithoutSubRouterIsNoop(t *testing.T) {
	r := New()
	mb := &MountBuilder{router: r}
	assert.Same(t, mb, mb.Name("anything"))
}

// --- RegisterExternal ---

func TestRegisterExternalWithoutNameIsListedButNotIndexed(t *testing.T) {
	r := New()
	r.RegisterExternal("GET", "/ext", "", "")

	found := false
	for _, rt := range r.Routes() {
		if rt.Pattern == "/ext" && rt.Method == "GET" {
			found = true
			assert.Empty(t, rt.Name)
		}
	}
	assert.True(t, found, "phantom route without a name must still appear in Routes()")
	_, err := r.URL("")
	assert.Error(t, err, "unnamed phantom route must not be reachable by name")
}

func TestRegisterExternalWithNameIsIndexed(t *testing.T) {
	r := New()
	r.RegisterExternal("GET", "/ext/{id}", "ext.show", "s3:GetObject")

	url, err := r.URL("ext.show", "id", "5")
	require.NoError(t, err)
	assert.Equal(t, "/ext/5", url)
}

// --- Resource validation panics and option handling ---

type gapController struct{}

func (gapController) Index(w http.ResponseWriter, r *http.Request) {}

func TestResourceValidationPanics(t *testing.T) {
	tests := []struct {
		name       string
		controller any
		opts       []ResourceOption
		wantMsg    string
	}{
		{"path without leading slash", gapController{}, []ResourceOption{Path("photos")}, "Path must start"},
		{"store verb GET rejected", gapController{}, []ResourceOption{StoreVerb(HTTPMethod(http.MethodGet))}, "StoreVerb must be POST or PUT"},
		{"only and except combined", gapController{}, []ResourceOption{Only("index"), Except("show")}, "cannot be combined"},
		{"unknown action in Only", gapController{}, []ResourceOption{Only("bogus")}, "unknown action"},
		{"unknown action in Except", gapController{}, []ResourceOption{Except("bogus")}, "unknown action"},
		{"unknown action in Names", gapController{}, []ResourceOption{Names(map[string]string{"bogus": "x"})}, "Names refers to unknown action"},
		{"nil controller", nil, nil, "controller must not be nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			msg := testutil.CapturePanic(func() {
				r.Resource("photos", tt.controller, tt.opts...)
			})
			require.NotEmpty(t, msg, "expected a panic")
			assert.Contains(t, msg, tt.wantMsg)
		})
	}
}

func TestResourceAcceptsValidOptionCombinations(t *testing.T) {
	r := New()
	r.Resource("photos", gapController{}, Only("index"), Path("/gallery"), Param("photo"))
	r.Resource("buckets", gapController{}, Except("index"), StoreVerb(PUT))

	routes := r.Routes()
	var patterns []string
	for _, rt := range routes {
		patterns = append(patterns, rt.Method+" "+rt.Pattern)
	}
	assert.Contains(t, patterns, "GET /gallery")
}

func TestIsResourceActionKnownAndUnknown(t *testing.T) {
	for _, a := range []string{"index", "create", "store", "show", "edit", "update", "destroy"} {
		assert.True(t, isResourceAction(a), a)
	}
	assert.False(t, isResourceAction("bogus"))
	assert.False(t, isResourceAction(""))
}

// --- Optimized handler: each combination of action, name, wildcard, middleware ---

func TestFinalizedRouteCombinations(t *testing.T) {
	var mwCalls int
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mwCalls++
			next.ServeHTTP(w, req)
		})
	}

	r := New()
	r.GET("/action-only", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Action", GetAction(req))
		w.Header().Set("X-Name", GetRouteName(req))
	})).Action("s3:Only")
	r.GET("/name-only", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Name", GetRouteName(req))
	})).Name("named")
	r.GET("/mw-only", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).With(mw)
	r.GET("/files/{path:.*}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Path", URLParam(req, "path"))
	})).Name("files")
	r.GET("/bare", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	r.Finalize()

	serve := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}

	w := serve("/action-only")
	assert.Equal(t, "s3:Only", w.Header().Get("X-Action"))
	assert.Empty(t, w.Header().Get("X-Name"))

	w = serve("/name-only")
	assert.Equal(t, "named", w.Header().Get("X-Name"))

	w = serve("/mw-only")
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, 1, mwCalls, "route middleware must run exactly once per request")

	w = serve("/files/a/b/c.txt")
	assert.Equal(t, "a/b/c.txt", w.Header().Get("X-Path"))

	w = serve("/bare")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Route listing output ---

func TestRoutesJSONResponseShape(t *testing.T) {
	r := New()
	r.GET("/x", http.HandlerFunc(testutil.NoopResponse)).Name("x")
	r.GET("/y", http.HandlerFunc(testutil.NoopResponse))

	h := NewListRoutesHandler(r, nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var body struct {
		Count  int         `json:"count"`
		Routes []RouteInfo `json:"routes"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Count)
	assert.Len(t, body.Routes, 2)
}

func TestRoutesHTMLPlaceholdersAndLinks(t *testing.T) {
	r := New()
	r.GET("/plain", http.HandlerFunc(testutil.NoopResponse))
	r.GET("/param/{id}", http.HandlerFunc(testutil.NoopResponse))
	r.QueryGET("/q", http.HandlerFunc(testutil.NoopResponse)).Query("acl")

	h := NewListRoutesHandler(r, &ListRoutesOptions{BaseURL: "https://example.test/"})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/", nil))
	html := w.Body.String()

	// Literal patterns link to the base URL; parameterised patterns do not.
	assert.Contains(t, html, `href="https://example.test/plain"`)
	assert.NotContains(t, html, `href="https://example.test/param/{id}"`)
	// Unnamed, unactioned routes render "-" placeholders.
	assert.Contains(t, html, `<td class="name">-</td>`)
	assert.Contains(t, html, `<td class="action">-</td>`)
	// Query column appears because one route uses query matching.
	assert.Contains(t, html, `<td class="query">acl</td>`)
}

func TestRoutesHTMLOmitsQueryAndHeaderColumnsWhenUnused(t *testing.T) {
	r := New()
	r.GET("/plain", http.HandlerFunc(testutil.NoopResponse))

	w := httptest.NewRecorder()
	NewListRoutesHandler(r, nil)(w, httptest.NewRequest("GET", "/", nil))
	html := w.Body.String()

	assert.NotContains(t, html, "<th>Query</th>")
	assert.NotContains(t, html, "<th>Headers</th>")
}

func TestRoutesHTMLShowsHeaderColumnWhenUsed(t *testing.T) {
	r := New()
	r.GET("/plain", http.HandlerFunc(testutil.NoopResponse))
	r.Dispatch("GET", "/h", func(d *DispatchBuilder, m Matchers) {
		d.When(m.HeaderEquals("X-Kind", "a")).DoFunc(testutil.NoopResponse).Name("h.a")
		d.DefaultFunc(testutil.NoopResponse).Name("h.default")
	})

	w := httptest.NewRecorder()
	NewListRoutesHandler(r, nil)(w, httptest.NewRequest("GET", "/", nil))
	html := w.Body.String()

	assert.Contains(t, html, "<th>Headers</th>")
	assert.NotContains(t, html, "<th>Query</th>")
	assert.Contains(t, html, "X-Kind: a")
}

func TestFormatRoutesCompactAndTableFormatting(t *testing.T) {
	routes := []RouteInfo{
		{
			Method:       "GET",
			Pattern:      "/b",
			QueryParams:  []QueryParam{{Key: "acl"}, {Key: "versioning", Value: "on"}},
			HeaderParams: []HeaderParam{{Key: "X-A"}, {Key: "X-B", Value: "2"}},
			Name:         "b.get",
			Action:       "s3:Get",
		},
		{Method: "GET", Pattern: "/a"},
	}

	var compact bytes.Buffer
	require.NoError(t, FormatRoutesCompact(&compact, routes))
	lines := strings.Split(strings.TrimSpace(compact.String()), "\n")
	require.Len(t, lines, 2)
	assert.Regexp(t, `^GET\s+/a\s+-$`, lines[0])
	assert.Regexp(t, `^GET\s+/b\?acl&versioning=on \[X-A, X-B: 2\]\s+b\.get$`, lines[1])

	var table bytes.Buffer
	require.NoError(t, FormatRoutesTable(&table, routes))
	out := table.String()
	assert.Contains(t, out, "acl&versioning=on")
	assert.Contains(t, out, "s3:Get")
	// Route /a has no name or action; both render as "-".
	assert.Regexp(t, `GET\s+/a\s+-\s+-\s+-`, out)
}

// --- Formatter ordering: pattern first, then method ---

func TestFormatRoutesSortOrderPatternThenMethod(t *testing.T) {
	routes := []RouteInfo{
		{Method: "POST", Pattern: "/b"},
		{Method: "PUT", Pattern: "/a"},
		{Method: "GET", Pattern: "/b"},
		{Method: "DELETE", Pattern: "/a"},
	}
	want := []string{"DELETE /a", "PUT /a", "GET /b", "POST /b"}

	t.Run("compact", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, FormatRoutesCompact(&buf, routes))
		var got []string
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			fields := strings.Fields(line)
			got = append(got, fields[0]+" "+fields[1])
		}
		assert.Equal(t, want, got)
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, FormatRoutesTable(&buf, routes))
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		var got []string
		for _, line := range lines[2:] { // skip header and rule
			fields := strings.Fields(line)
			got = append(got, fields[0]+" "+fields[1])
		}
		assert.Equal(t, want, got)
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, FormatRoutesJSON(&buf, routes))
		var body struct {
			Routes []RouteInfo `json:"routes"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &body))
		var names []string
		for _, rt := range body.Routes {
			names = append(names, rt.Method+" "+rt.Pattern)
		}
		assert.Equal(t, want, names)
	})

	// Input slice must not be reordered by the formatters.
	assert.Equal(t, "POST", routes[0].Method)
}
