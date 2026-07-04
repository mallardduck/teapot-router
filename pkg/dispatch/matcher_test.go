package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryExistsMatcher(t *testing.T) {
	matcher := QueryExistsMatcher{Key: "foo"}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"query param exists with value", "http://example.com/?foo=bar", true},
		{"query param exists without value", "http://example.com/?foo", true},
		{"query param does not exist", "http://example.com/?bar=baz", false},
		{"no query params", "http://example.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			assert.Equal(t, tt.expected, matcher.Matches(req))
		})
	}
}

func TestQueryExistsMatcherSpecificity(t *testing.T) {
	matcher := QueryExistsMatcher{Key: "test"}
	assert.Equal(t, 1, matcher.Specificity())
}

func TestQueryValueMatcher(t *testing.T) {
	matcher := QueryValueMatcher{Key: "type", Value: "full"}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"exact match", "http://example.com/?type=full", true},
		{"different value", "http://example.com/?type=partial", false},
		{"param missing", "http://example.com/?other=value", false},
		{"empty value", "http://example.com/?type=", false},
		{"no query params", "http://example.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			assert.Equal(t, tt.expected, matcher.Matches(req))
		})
	}
}

func TestQueryValueMatcherSpecificity(t *testing.T) {
	matcher := QueryValueMatcher{Key: "type", Value: "test"}
	assert.Equal(t, 2, matcher.Specificity())
}

func TestQueryExistsConstructor(t *testing.T) {
	m := QueryExists("foo")
	assert.IsType(t, QueryExistsMatcher{}, m)
	assert.Equal(t, 1, m.Specificity())
}

func TestQueryEqualsConstructor(t *testing.T) {
	m := QueryEquals("foo", "bar")
	assert.IsType(t, QueryValueMatcher{}, m)
	assert.Equal(t, 2, m.Specificity())
}

func TestHeaderExistsMatcher(t *testing.T) {
	matcher := HeaderExistsMatcher{Key: "X-Custom"}

	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{"header present", map[string]string{"X-Custom": "value"}, true},
		{"header missing", map[string]string{"Other": "value"}, false},
		{"no headers", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			assert.Equal(t, tt.expected, matcher.Matches(req))
		})
	}
}

func TestHeaderExistsMatcherSpecificity(t *testing.T) {
	matcher := HeaderExistsMatcher{Key: "X-Custom"}
	assert.Equal(t, 1, matcher.Specificity())
}

func TestHeaderValueMatcher(t *testing.T) {
	matcher := HeaderValueMatcher{Key: "Content-Type", Value: "application/json"}

	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{"exact match", map[string]string{"Content-Type": "application/json"}, true},
		{"different value", map[string]string{"Content-Type": "text/plain"}, false},
		{"header missing", map[string]string{"Other": "value"}, false},
		{"no headers", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			assert.Equal(t, tt.expected, matcher.Matches(req))
		})
	}
}

func TestHeaderValueMatcherSpecificity(t *testing.T) {
	matcher := HeaderValueMatcher{Key: "Content-Type", Value: "application/json"}
	assert.Equal(t, 2, matcher.Specificity())
}

func TestHeaderExistsConstructor(t *testing.T) {
	m := HeaderExists("X-Custom")
	assert.IsType(t, HeaderExistsMatcher{}, m)
	assert.Equal(t, 1, m.Specificity())
}

func TestHeaderEqualsConstructor(t *testing.T) {
	m := HeaderEquals("Content-Type", "application/json")
	assert.IsType(t, HeaderValueMatcher{}, m)
	assert.Equal(t, 2, m.Specificity())
}

func TestHostSubdomainMatcher(t *testing.T) {
	matcher := HostSubdomainMatcher{CanonicalDomain: "s3.example.com"}

	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"subdomain of canonical domain", "mybucket.s3.example.com", true},
		{"subdomain with port", "mybucket.s3.example.com:9000", true},
		{"bare canonical domain", "s3.example.com", false},
		{"unrelated host", "example.org", false},
		{"IP host", "192.168.1.10", false},
		{"substring, not suffix", "s3.example.com.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://placeholder/", nil)
			req.Host = tt.host
			assert.Equal(t, tt.expected, matcher.Matches(req))
		})
	}
}

func TestHostSubdomainMatcherSpecificity(t *testing.T) {
	matcher := HostSubdomainMatcher{CanonicalDomain: "s3.example.com"}
	assert.Equal(t, 1, matcher.Specificity())
}

func TestHostHasSubdomainConstructor(t *testing.T) {
	m := HostHasSubdomain("s3.example.com")
	assert.IsType(t, HostSubdomainMatcher{}, m)
	assert.Equal(t, 1, m.Specificity())
}

// TestHostHasSubdomainWithDispatcher verifies the matcher composes with
// Dispatcher the same way the query/header matchers do, for the two-router
// (path-style vs. subdomain-style) use case it was built for.
func TestHostHasSubdomainWithDispatcher(t *testing.T) {
	d := New(func(b *Builder) {
		b.When(HostHasSubdomain("s3.example.com")).Do(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("subdomain"))
		})
		b.Default(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("path"))
		})
	})

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"subdomain host routes to subdomain handler", "mybucket.s3.example.com", "subdomain"},
		{"bare canonical domain falls through to default", "s3.example.com", "path"},
		{"unrelated host falls through to default", "example.org", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://placeholder/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			d.ServeHTTP(rec, req)
			assert.Equal(t, tt.expected, rec.Body.String())
		})
	}
}
