package teapot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchBasicRouting(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/items", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("list-all"))
		})
		d.When(m.QueryEquals("type", "active")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("list-active"))
		})
		d.When(m.QueryExists("search")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("search"))
		})
	})

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"no params hits default", "/items", "list-all"},
		{"type=active hits specific route", "/items?type=active", "list-active"},
		{"search param hits search route", "/items?search=foo", "search"},
		{"type=inactive hits default", "/items?type=inactive", "list-all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

func TestDispatchNameAndAction(t *testing.T) {
	r := New()

	var capturedName, capturedAction string

	r.Dispatch("GET", "/things", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(_ http.ResponseWriter, req *http.Request) {
			capturedName = GetRouteName(req)
			capturedAction = GetAction(req)
		}).Name("things.list").Action("api:things:List")

		d.When(m.QueryEquals("view", "detail")).Do(func(_ http.ResponseWriter, req *http.Request) {
			capturedName = GetRouteName(req)
			capturedAction = GetAction(req)
		}).Name("things.detail").Action("api:things:Detail")
	})

	t.Run("default route injects name and action", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/things", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "things.list", capturedName)
		assert.Equal(t, "api:things:List", capturedAction)
	})

	t.Run("conditional route injects name and action", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/things?view=detail", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "things.detail", capturedName)
		assert.Equal(t, "api:things:Detail", capturedAction)
	})
}

func TestDispatchURLGeneration(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/users/{id}", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("user.show")
		d.When(m.QueryExists("edit")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("user.edit")
	})

	url, err := r.URL("user.show", "id", "42")
	require.NoError(t, err)
	assert.Equal(t, "/users/42", url)

	url, err = r.URL("user.edit", "id", "42")
	require.NoError(t, err)
	assert.Equal(t, "/users/42", url)
}

func TestDispatchRouteListing(t *testing.T) {
	r := New()

	r.Dispatch("GET", "/api/items", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("api.items.list").Action("api:Items:List")
		d.When(m.QueryEquals("status", "active")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("api.items.active").Action("api:Items:ListActive")
		d.When(m.QueryExists("search")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("api.items.search").Action("api:Items:Search")
	})

	routes := r.Routes()
	require.Len(t, routes, 3)

	// Index by name for assertions
	byName := make(map[string]RouteInfo)
	for _, rt := range routes {
		byName[rt.Name] = rt
	}

	list := byName["api.items.list"]
	assert.Equal(t, "GET", list.Method)
	assert.Equal(t, "/api/items", list.Pattern)
	assert.Equal(t, "api:Items:List", list.Action)
	assert.Empty(t, list.QueryParams)

	active := byName["api.items.active"]
	assert.Equal(t, "api:Items:ListActive", active.Action)
	require.Len(t, active.QueryParams, 1)
	assert.Equal(t, "status", active.QueryParams[0].Key)
	assert.Equal(t, "active", active.QueryParams[0].Value)

	search := byName["api.items.search"]
	assert.Equal(t, "api:Items:Search", search.Action)
	require.Len(t, search.QueryParams, 1)
	assert.Equal(t, "search", search.QueryParams[0].Key)
	assert.Equal(t, "", search.QueryParams[0].Value) // existence check, no value
}

func TestDispatchWildcardParams(t *testing.T) {
	r := New()

	var capturedBucket, capturedKey string

	r.Dispatch("GET", "/{bucket}/{key:.*}", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(w http.ResponseWriter, req *http.Request) {
			capturedBucket = URLParam(req, "bucket")
			capturedKey = URLParam(req, "key")
			_, _ = w.Write([]byte("object"))
		})
		d.When(m.QueryExists("acl")).Do(func(w http.ResponseWriter, req *http.Request) {
			capturedBucket = URLParam(req, "bucket")
			capturedKey = URLParam(req, "key")
			_, _ = w.Write([]byte("acl"))
		})
	})

	t.Run("default route with wildcard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "mybucket", capturedBucket)
		assert.Equal(t, "path/to/file.txt", capturedKey)
		assert.Equal(t, "object", w.Body.String())
	})

	t.Run("conditional route with wildcard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mybucket/path/to/file.txt?acl", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "mybucket", capturedBucket)
		assert.Equal(t, "path/to/file.txt", capturedKey)
		assert.Equal(t, "acl", w.Body.String())
	})
}

func TestDispatchMiddleware(t *testing.T) {
	r := New()

	middlewareCalled := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, req)
		})
	}

	r.Dispatch("GET", "/mw-test", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}).With(mw)
	})

	req := httptest.NewRequest("GET", "/mw-test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, "ok", w.Body.String())
}

func TestDispatchConflictWithDirect(t *testing.T) {
	r := New()
	r.GET("/conflict", func(_ http.ResponseWriter, _ *http.Request) {})

	assert.Panics(t, func() {
		r.Dispatch("GET", "/conflict", func(d *DispatchBuilder, m Matchers) {
			d.Default(func(_ http.ResponseWriter, _ *http.Request) {})
		})
	})
}

func TestDispatchConflictWithExistingDispatcher(t *testing.T) {
	r := New()
	r.QueryGET("/conflict", func(_ http.ResponseWriter, _ *http.Request) {}).Query("foo")

	assert.Panics(t, func() {
		r.Dispatch("GET", "/conflict", func(d *DispatchBuilder, m Matchers) {
			d.Default(func(_ http.ResponseWriter, _ *http.Request) {})
		})
	})
}

func TestDispatchMissingHandler(t *testing.T) {
	r := New()

	assert.Panics(t, func() {
		r.Dispatch("GET", "/missing", func(d *DispatchBuilder, m Matchers) {
			d.When(m.QueryExists("foo")) // No .Do() call — handler is nil
		})
	})
}

func TestDispatchDuplicateName(t *testing.T) {
	r := New()

	assert.Panics(t, func() {
		r.Dispatch("GET", "/dup-name", func(d *DispatchBuilder, m Matchers) {
			d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("same-name")
			d.When(m.QueryExists("x")).Do(func(_ http.ResponseWriter, _ *http.Request) {}).Name("same-name")
		})
	})
}

func TestDispatchNameConflictWithFluentAPI(t *testing.T) {
	r := New()
	r.GET("/other", func(_ http.ResponseWriter, _ *http.Request) {}).Name("taken")

	assert.Panics(t, func() {
		r.Dispatch("GET", "/different-path", func(d *DispatchBuilder, m Matchers) {
			d.Default(func(_ http.ResponseWriter, _ *http.Request) {}).Name("taken")
		})
	})
}

func TestDispatchCoexistsWithFluentAPI(t *testing.T) {
	r := New()

	// Fluent-style route on one path
	r.GET("/fluent", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fluent"))
	}).Name("fluent.route")

	// Dispatch-style route on a different path
	r.Dispatch("GET", "/grouped", func(d *DispatchBuilder, m Matchers) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("grouped-default"))
		}).Name("grouped.default")
		d.When(m.QueryExists("special")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("grouped-special"))
		}).Name("grouped.special")
	})

	t.Run("fluent route works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fluent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "fluent", w.Body.String())
	})

	t.Run("dispatch default works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/grouped", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "grouped-default", w.Body.String())
	})

	t.Run("dispatch conditional works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/grouped?special", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "grouped-special", w.Body.String())
	})

	// All routes visible in listing
	routes := r.Routes()
	assert.Len(t, routes, 3)
}

func TestDispatchWithNamedGroup(t *testing.T) {
	r := New()

	r.NamedGroup("/api", "api", func(sub *Router) {
		sub.Dispatch("GET", "/things", func(d *DispatchBuilder, m Matchers) {
			d.Default(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("things"))
			}).Name("list").Action("api:Things:List")
		})
	})

	// Route should be reachable at /api/things
	req := httptest.NewRequest("GET", "/api/things", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "things", w.Body.String())

	// Name should be prefixed
	url, err := r.URL("api.list")
	require.NoError(t, err)
	assert.Equal(t, "/api/things", url)
}
