package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDispatcherDefault(t *testing.T) {
	d := New(func(b *Builder) {
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("default"))
		})
	})

	req := httptest.NewRequest("GET", "/anything", nil)
	w := httptest.NewRecorder()
	d.ServeHTTP(w, req)

	assert.Equal(t, "default", w.Body.String())
}

func TestDispatcherQueryRouting(t *testing.T) {
	d := New(func(b *Builder) {
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("fallback"))
		})
		b.When(QueryExists("action")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("action"))
		})
		b.When(QueryEquals("type", "v2")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("v2"))
		})
	})

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"no query params hits fallback", "/test", "fallback"},
		{"type=v2 hits v2 route", "/test?type=v2", "v2"},
		{"action param hits action route", "/test?action=foo", "action"},
		{"type=v1 hits fallback", "/test?type=v1", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			d.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

func TestDispatcherSpecificityOrder(t *testing.T) {
	// QueryEquals (specificity 2) should be checked before QueryExists (specificity 1)
	// regardless of registration order.
	d := New(func(b *Builder) {
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("fallback"))
		})
		// Register Exists first — sort must still put Equals ahead
		b.When(QueryExists("type")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("exists"))
		})
		b.When(QueryEquals("type", "special")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("special"))
		})
	})

	// type=special should match the Equals route, not the Exists route
	req := httptest.NewRequest("GET", "/test?type=special", nil)
	w := httptest.NewRecorder()
	d.ServeHTTP(w, req)
	assert.Equal(t, "special", w.Body.String())

	// type=other should match the Exists route
	req = httptest.NewRequest("GET", "/test?type=other", nil)
	w = httptest.NewRecorder()
	d.ServeHTTP(w, req)
	assert.Equal(t, "exists", w.Body.String())
}

func TestDispatcherAndSemantics(t *testing.T) {
	d := New(func(b *Builder) {
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("fallback"))
		})
		// Both a AND b required (specificity 2: 1+1)
		b.When(QueryExists("a"), QueryExists("b")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("both"))
		})
		// Only a required (specificity 1)
		b.When(QueryExists("a")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("a-only"))
		})
	})

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"both a and b", "/test?a=1&b=2", "both"},
		{"only a", "/test?a=1", "a-only"},
		{"only b hits fallback", "/test?b=2", "fallback"},
		{"neither hits fallback", "/test", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			d.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

func TestDispatcherNoRoutes(t *testing.T) {
	d := New(func(b *Builder) {})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	d.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestDispatcherNoDefaultNoMatch(t *testing.T) {
	d := New(func(b *Builder) {
		b.When(QueryExists("required")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("matched"))
		})
	})

	// Without required param, no route matches → 404
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	d.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)

	// With required param → matched
	req = httptest.NewRequest("GET", "/test?required=yes", nil)
	w = httptest.NewRecorder()
	d.ServeHTTP(w, req)
	assert.Equal(t, "matched", w.Body.String())
}

func TestDispatcherMixedSpecificity(t *testing.T) {
	// Exists("a") + Equals("b","1") = specificity 3
	// Equals("a","x")              = specificity 2
	// Exists("a")                  = specificity 1
	// Default                      = specificity 0
	d := New(func(b *Builder) {
		b.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("default"))
		})
		b.When(QueryExists("a")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("a-exists"))
		})
		b.When(QueryEquals("a", "x")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("a-equals-x"))
		})
		b.When(QueryExists("a"), QueryEquals("b", "1")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("a-and-b1"))
		})
	})

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"a present + b=1", "/test?a=anything&b=1", "a-and-b1"},
		{"a=x only", "/test?a=x", "a-equals-x"},
		{"a=other only", "/test?a=other", "a-exists"},
		{"nothing", "/test", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			d.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}
