// Package dispatch provides a router-agnostic HTTP request dispatcher that
// routes to handlers based on Matcher conditions (query parameters, and
// eventually other request attributes like headers).
//
// A Dispatcher implements http.Handler and can be used with any Go HTTP router.
// Routes are evaluated in specificity order — the first route whose matchers
// all match wins. A route with no matchers acts as the fallback.
//
// Basic usage:
//
//	d := dispatch.New(func(b *dispatch.Builder) {
//	    b.Default(listV1)                                          // fallback
//	    b.When(dispatch.QueryEquals("list-type", "2")).Do(listV2)  // specific match
//	    b.When(dispatch.QueryExists("location")).Do(getLocation)   // existence match
//	})
//
//	http.Handle("/bucket", d)  // works with any router
package dispatch

import (
	"net/http"
	"sort"
)

// Dispatcher routes requests to handlers based on Matcher conditions.
// It implements http.Handler and is completely router-agnostic — no dependency
// on chi, gorilla, or any other router package.
type Dispatcher struct {
	routes []*route
}

type route struct {
	handler  http.HandlerFunc
	matchers []Matcher
}

// New creates a Dispatcher by calling fn with a Builder to configure routes.
// The returned Dispatcher is ready to serve requests immediately.
func New(fn func(b *Builder)) *Dispatcher {
	b := &Builder{}
	fn(b)
	d := &Dispatcher{routes: b.routes}
	d.sortRoutes()
	return d
}

// ServeHTTP dispatches the request to the first handler whose matchers all match.
// Routes are evaluated in specificity order (highest first). If no route matches,
// responds with 404 Not Found.
func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rt := range d.routes {
		if matchesAll(r, rt.matchers) {
			rt.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func matchesAll(r *http.Request, matchers []Matcher) bool {
	for _, m := range matchers {
		if !m.Matches(r) {
			return false
		}
	}
	return true // empty matchers = always matches (fallback)
}

func (d *Dispatcher) sortRoutes() {
	sort.SliceStable(d.routes, func(i, j int) bool {
		return specificity(d.routes[i]) > specificity(d.routes[j])
	})
}

func specificity(rt *route) int {
	s := 0
	for _, m := range rt.matchers {
		s += m.Specificity()
	}
	return s
}

// Builder configures routes for a Dispatcher.
type Builder struct {
	routes []*route
}

// Default sets the fallback handler — it matches when no other route's matchers match.
// Only one Default should be set per Dispatcher; the last one wins if multiple are added.
func (b *Builder) Default(h http.HandlerFunc) {
	b.routes = append(b.routes, &route{handler: h})
}

// When starts a conditional route with the given matchers.
// All matchers must match for the route to be selected (AND semantics).
// Call [Route.Do] on the returned Route to set the handler.
func (b *Builder) When(matchers ...Matcher) *Route {
	rt := &route{matchers: matchers}
	b.routes = append(b.routes, rt)
	return &Route{route: rt}
}

// Route is a conditional route being configured via When.
type Route struct {
	route *route
}

// Do will set the handler for this conditional route.
func (rt *Route) Do(h http.HandlerFunc) {
	rt.route.handler = h
}
