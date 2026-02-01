package core

import "net/http"

// QueryMatcher defines how to match query parameters
type QueryMatcher interface {
	Matches(r *http.Request) bool
	Specificity() int // Higher = more specific
}

// QueryExistsMatcher matches if a query parameter exists (any value)
type QueryExistsMatcher struct {
	Key string
}

func (m QueryExistsMatcher) Matches(r *http.Request) bool {
	return r.URL.Query().Has(m.Key)
}

func (m QueryExistsMatcher) Specificity() int {
	return 1
}

// QueryValueMatcher matches if a query parameter has a specific value
type QueryValueMatcher struct {
	Key   string
	Value string
}

func (m QueryValueMatcher) Matches(r *http.Request) bool {
	return r.URL.Query().Get(m.Key) == m.Value
}

func (m QueryValueMatcher) Specificity() int {
	return 2 // Value matching is more specific than existence
}
