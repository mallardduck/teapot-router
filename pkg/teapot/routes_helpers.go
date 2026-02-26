package teapot

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"text/tabwriter"
)

// RouteFilter is a predicate function that determines whether a route
// should be included in output. Return true to include the route.
type RouteFilter func(RouteInfo) bool

// FilterRoutes applies a filter to a slice of routes, returning only
// those for which the filter returns true. If filter is nil, all routes
// are returned unchanged.
func FilterRoutes(routes []RouteInfo, filter RouteFilter) []RouteInfo {
	if filter == nil {
		return routes
	}
	filtered := make([]RouteInfo, 0, len(routes))
	for _, route := range routes {
		if filter(route) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// ListRoutesOptions controls the behaviour of [NewListRoutesHandler] and
// [NewListRoutesHandlerWithRoutes].
type ListRoutesOptions struct {
	// Filter, when non-nil, is applied to each route; only routes for which it
	// returns true are included in the output.
	Filter RouteFilter

	// BaseURL, when non-empty, is prepended to each route pattern in the HTML
	// output to generate a clickable link.  Only patterns without path
	// parameters (i.e. no "{…}" segments) are linked, since those are directly
	// navigable in a browser.
	//
	// Example: "https://myapp.example.com" or "http://localhost:8080"
	//
	// If BaseURLFunc is also set, BaseURLFunc takes precedence.
	BaseURL string

	// BaseURLFunc, when non-nil, is called on each request to determine the
	// base URL used for clickable links in the HTML output.  It takes
	// precedence over the static BaseURL field, and is useful when the scheme
	// or host must be derived from the incoming request (e.g. via the Host
	// header or X-Forwarded-Host).
	//
	// Example – derive base URL from the request host:
	//
	//	BaseURLFunc: func(r *http.Request) string {
	//	    scheme := "https"
	//	    if r.TLS == nil {
	//	        scheme = "http"
	//	    }
	//	    return scheme + "://" + r.Host
	//	}
	BaseURLFunc func(*http.Request) string
}

// resolveBaseURL returns the effective base URL for the given request.
// BaseURLFunc takes precedence over the static BaseURL string.
func (o ListRoutesOptions) resolveBaseURL(req *http.Request) string {
	if o.BaseURLFunc != nil {
		return o.BaseURLFunc(req)
	}
	return o.BaseURL
}

// NewListRoutesHandler returns an HTTP handler that displays registered routes.
// The handler responds with JSON or HTML based on the Accept header.
//
// Example:
//
//	// Show all routes
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, teapot.ListRoutesOptions{}))
//
//	// Exclude internal routes
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, teapot.ListRoutesOptions{
//	    Filter: func(route teapot.RouteInfo) bool {
//	        return !strings.HasPrefix(route.Pattern, "/.internal/")
//	    },
//	}))
//
//	// With a static base URL for clickable links in the HTML output
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, teapot.ListRoutesOptions{
//	    BaseURL: "http://localhost:8080",
//	}))
//
//	// With a dynamic base URL derived from each incoming request
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, teapot.ListRoutesOptions{
//	    BaseURLFunc: func(r *http.Request) string {
//	        scheme := "https"
//	        if r.TLS == nil {
//	            scheme = "http"
//	        }
//	        return scheme + "://" + r.Host
//	    },
//	}))
//
// If you want to merge routes from multiple routers, use [AggregateRoutes] and
// then [NewListRoutesHandlerWithRoutes].
func NewListRoutesHandler(router *Router, opts ListRoutesOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		NewListRoutesHandlerWithRoutes(router.Routes(), opts)(w, req)
	}
}

// NewListRoutesHandlerWithRoutes returns an HTTP handler that displays the
// provided routes slice.  This is useful when combining routes from multiple
// routers via [AggregateRoutes].
func NewListRoutesHandlerWithRoutes(routes []RouteInfo, opts ListRoutesOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		filteredRoutes := FilterRoutes(routes, opts.Filter)
		renderListRoutes(w, req, filteredRoutes, opts.resolveBaseURL(req))
	}
}

// routesPageData is the template data for the routes HTML page.
type routesPageData struct {
	Count      int
	HasQuery   bool
	HasHeaders bool
	Routes     []routeRowData
}

// routeRowData is the per-route data passed to the routes HTML template.
type routeRowData struct {
	MethodClass  string
	Method       string
	Pattern      string
	PatternURL   string // non-empty only when the pattern has no path parameters
	QueryParams  string
	HeaderParams string
	Name         string
	Action       string
}

var routesPageTmpl = template.Must(template.New("routes").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Routes</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; margin: 2rem; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
        th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; font-weight: 600; }
        tr:hover { background-color: #f9f9f9; }
        .method { font-family: monospace; font-weight: 600; }
        .pattern { font-family: monospace; color: #0066cc; }
        .pattern a { color: inherit; text-decoration: none; }
        .pattern a:hover { text-decoration: underline; }
        .query { font-family: monospace; color: #6c757d; }
        .name { color: #666; }
        .action { color: #888; font-size: 0.9em; }
        .count { color: #666; font-size: 0.9em; }
        .get { color: #28a745; }
        .post { color: #007bff; }
        .put { color: #ffc107; }
        .delete { color: #dc3545; }
        .head { color: #6c757d; }
        .patch { color: #17a2b8; }
        .options { color: #6610f2; }
    </style>
</head>
<body>
    <h1>Registered Routes</h1>
    <p class="count">Total: {{.Count}} routes</p>
    <table>
        <thead>
            <tr>
                <th>Method</th>
                <th>Pattern</th>
                {{if .HasQuery}}<th>Query</th>{{end}}
                {{if .HasHeaders}}<th>Headers</th>{{end}}
                <th>Name</th>
                <th>Action</th>
            </tr>
        </thead>
        <tbody>
            {{range .Routes}}<tr>
                <td class="method {{.MethodClass}}">{{.Method}}</td>
                <td class="pattern">{{if .PatternURL}}<a href="{{.PatternURL}}">{{.Pattern}}</a>{{else}}{{.Pattern}}{{end}}</td>
                {{if $.HasQuery}}<td class="query">{{.QueryParams}}</td>{{end}}
                {{if $.HasHeaders}}<td class="query">{{.HeaderParams}}</td>{{end}}
                <td class="name">{{.Name}}</td>
                <td class="action">{{.Action}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>`))

func renderListRoutes(w http.ResponseWriter, req *http.Request, filteredRoutes []RouteInfo, baseURL string) {
	// Check Accept header for JSON vs HTML
	accept := req.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]any{
			"count":  len(filteredRoutes),
			"routes": filteredRoutes,
		})
		if err != nil {
			log.Printf("[teapot-router] failed to encode routes as JSON: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Default to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Only show Query/Headers columns when at least one route uses them.
	hasQuery := false
	hasHeaders := false
	for _, rt := range filteredRoutes {
		if len(rt.QueryParams) > 0 {
			hasQuery = true
		}
		if len(rt.HeaderParams) > 0 {
			hasHeaders = true
		}
		if hasQuery && hasHeaders {
			break
		}
	}

	base := strings.TrimRight(baseURL, "/")

	data := routesPageData{
		Count:      len(filteredRoutes),
		HasQuery:   hasQuery,
		HasHeaders: hasHeaders,
		Routes:     make([]routeRowData, len(filteredRoutes)),
	}

	for i, route := range filteredRoutes {
		name := route.Name
		if name == "" {
			name = "-"
		}
		action := route.Action
		if action == "" {
			action = "-"
		}

		var patternURL string
		if base != "" && !strings.Contains(route.Pattern, "{") {
			patternURL = base + route.Pattern
		}

		data.Routes[i] = routeRowData{
			MethodClass:  strings.ToLower(route.Method),
			Method:       route.Method,
			Pattern:      route.Pattern,
			PatternURL:   patternURL,
			QueryParams:  formatQueryParams(route.QueryParams),
			HeaderParams: formatHeaderParams(route.HeaderParams),
			Name:         name,
			Action:       action,
		}
	}

	if err := routesPageTmpl.Execute(w, data); err != nil {
		log.Printf("[teapot-router] failed to render routes HTML: %v", err)
	}
}

// FormatRoutesJSON writes routes as JSON to the writer.
// This is useful for CLI commands with --json flag.
//
// Example:
//
//	routes := router.Routes()
//	teapot.FormatRoutesJSON(os.Stdout, routes)
func FormatRoutesJSON(w io.Writer, routes []RouteInfo) error {
	// Sort by pattern, then method
	sorted := make([]RouteInfo, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Pattern != sorted[j].Pattern {
			return sorted[i].Pattern < sorted[j].Pattern
		}
		return sorted[i].Method < sorted[j].Method
	})

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]any{
		"count":  len(sorted),
		"routes": sorted,
	})
}

// FormatRoutesTable writes routes as a formatted table to the writer.
// This is useful for CLI commands with human-readable output.
//
// Example:
//
//	routes := router.Routes()
//	teapot.FormatRoutesTable(os.Stdout, routes)
func FormatRoutesTable(w io.Writer, routes []RouteInfo) error {
	// Sort by pattern, then method
	sorted := make([]RouteInfo, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Pattern != sorted[j].Pattern {
			return sorted[i].Pattern < sorted[j].Pattern
		}
		return sorted[i].Method < sorted[j].Method
	})

	// Create tabwriter
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintln(tw, "METHOD\tPATTERN\tQUERY PARAMS\tNAME\tACTION")
	_, _ = fmt.Fprintln(tw, "------\t-------\t------------\t----\t------")

	// Rows
	for _, route := range sorted {
		name := route.Name
		if name == "" {
			name = "-"
		}
		action := route.Action
		if action == "" {
			action = "-"
		}
		queryParams := formatQueryParams(route.QueryParams)

		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			route.Method,
			route.Pattern,
			queryParams,
			name,
			action,
		)
	}

	return tw.Flush()
}

// formatQueryParams formats query parameters for display
func formatQueryParams(params []QueryParam) string {
	if len(params) == 0 {
		return "-"
	}

	var parts []string
	for _, p := range params {
		if p.Value == "" {
			parts = append(parts, p.Key)
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", p.Key, p.Value))
		}
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "&"
		}
		result += part
	}
	return result
}

// formatHeaderParams formats header parameters for display
func formatHeaderParams(params []HeaderParam) string {
	if len(params) == 0 {
		return "-"
	}

	var parts []string
	for _, p := range params {
		if p.Value == "" {
			parts = append(parts, p.Key)
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", p.Key, p.Value))
		}
	}
	return strings.Join(parts, ", ")
}

// FormatRoutesCompact writes routes in a compact format (METHOD PATH?QUERY NAME).
// This is useful for quick scanning of routes.
//
// Example:
//
//	routes := router.Routes()
//	teapot.FormatRoutesCompact(os.Stdout, routes)
func FormatRoutesCompact(w io.Writer, routes []RouteInfo) error {
	// Sort by pattern, then method
	sorted := make([]RouteInfo, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Pattern != sorted[j].Pattern {
			return sorted[i].Pattern < sorted[j].Pattern
		}
		return sorted[i].Method < sorted[j].Method
	})

	for _, route := range sorted {
		name := route.Name
		if name == "" {
			name = "-"
		}

		// Build path with query params
		pathWithQuery := route.Pattern
		if len(route.QueryParams) > 0 {
			pathWithQuery += "?" + formatQueryParams(route.QueryParams)
		}
		if len(route.HeaderParams) > 0 {
			pathWithQuery += " [" + formatHeaderParams(route.HeaderParams) + "]"
		}

		_, _ = fmt.Fprintf(w, "%-7s %-50s %s\n",
			route.Method,
			pathWithQuery,
			name,
		)
	}

	return nil
}
