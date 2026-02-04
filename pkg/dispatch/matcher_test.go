package dispatch

import (
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
