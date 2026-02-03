package dispatch

import "net/http"

// Matcher defines a condition for dispatching to a handler.
// All matchers must match for a route to be selected (AND semantics when
// combined via Builder.When). Implementations must be safe for concurrent use.
type Matcher interface {
	Matches(r *http.Request) bool
	Specificity() int // Higher values are checked first
}

// QueryExistsMatcher matches when a query parameter is present, regardless of its value.
type QueryExistsMatcher struct {
	Key string
}

// Matches reports whether the query parameter Key is present in the request URL.
func (m QueryExistsMatcher) Matches(r *http.Request) bool {
	return r.URL.Query().Has(m.Key)
}

// Specificity returns 1 (existence check).
func (m QueryExistsMatcher) Specificity() int {
	return 1
}

// QueryExists returns a Matcher that matches when the given query parameter is present.
func QueryExists(key string) Matcher {
	return QueryExistsMatcher{Key: key}
}

// QueryValueMatcher matches when a query parameter equals a specific value.
type QueryValueMatcher struct {
	Key   string
	Value string
}

// Matches reports whether the query parameter Key equals Value.
func (m QueryValueMatcher) Matches(r *http.Request) bool {
	return r.URL.Query().Get(m.Key) == m.Value
}

// Specificity returns 2 (value match is more specific than existence).
func (m QueryValueMatcher) Specificity() int {
	return 2
}

// QueryEquals returns a Matcher that matches when the given query parameter equals value.
func QueryEquals(key, value string) Matcher {
	return QueryValueMatcher{Key: key, Value: value}
}
