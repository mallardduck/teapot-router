package core

import (
	"net/http/httptest"
	"testing"
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
			if got := matcher.Matches(req); got != tt.expected {
				t.Errorf("Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestQueryExistsMatcherSpecificity(t *testing.T) {
	matcher := QueryExistsMatcher{Key: "test"}
	if got := matcher.Specificity(); got != 1 {
		t.Errorf("Specificity() = %d, want 1", got)
	}
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
			if got := matcher.Matches(req); got != tt.expected {
				t.Errorf("Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestQueryValueMatcherSpecificity(t *testing.T) {
	matcher := QueryValueMatcher{Key: "type", Value: "test"}
	if got := matcher.Specificity(); got != 2 {
		t.Errorf("Specificity() = %d, want 2", got)
	}
}
