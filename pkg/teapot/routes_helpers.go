package teapot

import (
	"encoding/json"
	"fmt"
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

// NewListRoutesHandler returns an HTTP handler that displays registered routes.
// The handler responds with JSON or HTML based on the Accept header.
// If filter is non-nil, only routes for which filter returns true are included.
// Pass nil to show all routes.
//
// Example:
//
//	// Show all routes
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, nil))
//
//	// Exclude internal routes
//	r.GET("/.internal/routes", teapot.NewListRoutesHandler(router, func(route teapot.RouteInfo) bool {
//	    return !strings.HasPrefix(route.Pattern, "/.internal/")
//	}))
func NewListRoutesHandler(router *Router, filter RouteFilter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		routes := FilterRoutes(router.Routes(), filter)

		// Check Accept header for JSON vs HTML
		accept := req.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"count":  len(routes),
				"routes": routes,
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
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Routes</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; margin: 2rem; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%%; margin-top: 1rem; }
        th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; font-weight: 600; }
        tr:hover { background-color: #f9f9f9; }
        .method { font-family: monospace; font-weight: 600; }
        .pattern { font-family: monospace; color: #0066cc; }
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
    <p class="count">Total: %d routes</p>
    <table>
        <thead>
            <tr>
                <th>Method</th>
                <th>Pattern</th>
                <th>Query</th>
                <th>Headers</th>
                <th>Name</th>
                <th>Action</th>
            </tr>
        </thead>
        <tbody>
`, len(routes))

		for _, route := range routes {
			methodClass := strings.ToLower(route.Method)
			name := route.Name
			if name == "" {
				name = "-"
			}
			action := route.Action
			if action == "" {
				action = "-"
			}
			query := formatQueryParams(route.QueryParams)
			headers := formatHeaderParams(route.HeaderParams)

			_, _ = fmt.Fprintf(w, `            <tr>
                <td class="method %s">%s</td>
                <td class="pattern">%s</td>
                <td class="query">%s</td>
                <td class="query">%s</td>
                <td class="name">%s</td>
                <td class="action">%s</td>
            </tr>
`, methodClass, route.Method, route.Pattern, query, headers, name, action)
		}

		_, _ = fmt.Fprintf(w, `        </tbody>
    </table>
</body>
</html>`)
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
