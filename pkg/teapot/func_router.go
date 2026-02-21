package teapot

import "net/http"

// FuncRouter wraps Router and accepts plain function literals as handlers.
// Obtain via r.Func() — safe for all sub-router contexts (groups, With, Route blocks).
//
// Example:
//
//	r.Func().GET("/users", func(w http.ResponseWriter, r *http.Request) {
//	    _, _ = w.Write([]byte("list users"))
//	}).Name("users.index")
type FuncRouter struct{ r *Router }

// Func returns a FuncRouter that wraps this Router and accepts plain function
// literals as handlers, wrapping them in http.HandlerFunc automatically.
func (r *Router) Func() *FuncRouter { return &FuncRouter{r: r} }

// GET registers a GET route using a plain function literal.
func (fr *FuncRouter) GET(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.GET(pattern, http.HandlerFunc(h))
}

// POST registers a POST route using a plain function literal.
func (fr *FuncRouter) POST(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.POST(pattern, http.HandlerFunc(h))
}

// PUT registers a PUT route using a plain function literal.
func (fr *FuncRouter) PUT(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.PUT(pattern, http.HandlerFunc(h))
}

// DELETE registers a DELETE route using a plain function literal.
func (fr *FuncRouter) DELETE(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.DELETE(pattern, http.HandlerFunc(h))
}

// HEAD registers a HEAD route using a plain function literal.
func (fr *FuncRouter) HEAD(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.HEAD(pattern, http.HandlerFunc(h))
}

// PATCH registers a PATCH route using a plain function literal.
func (fr *FuncRouter) PATCH(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.PATCH(pattern, http.HandlerFunc(h))
}

// OPTIONS registers an OPTIONS route using a plain function literal.
func (fr *FuncRouter) OPTIONS(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.OPTIONS(pattern, http.HandlerFunc(h))
}

// QueryGET registers a GET route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryGET(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryGET(pattern, http.HandlerFunc(h))
}

// QueryPOST registers a POST route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryPOST(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryPOST(pattern, http.HandlerFunc(h))
}

// QueryPUT registers a PUT route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryPUT(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryPUT(pattern, http.HandlerFunc(h))
}

// QueryDELETE registers a DELETE route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryDELETE(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryDELETE(pattern, http.HandlerFunc(h))
}

// QueryHEAD registers a HEAD route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryHEAD(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryHEAD(pattern, http.HandlerFunc(h))
}

// QueryPATCH registers a PATCH route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryPATCH(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryPATCH(pattern, http.HandlerFunc(h))
}

// QueryOPTIONS registers an OPTIONS route with query multiplexing using a plain function literal.
func (fr *FuncRouter) QueryOPTIONS(pattern string, h func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	return fr.r.QueryOPTIONS(pattern, http.HandlerFunc(h))
}
