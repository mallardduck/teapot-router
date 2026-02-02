package teapot

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

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
			// Existence check only
			parts = append(parts, p.Key)
		} else {
			// Value check
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

		_, _ = fmt.Fprintf(w, "%-7s %-50s %s\n",
			route.Method,
			pathWithQuery,
			name,
		)
	}

	return nil
}
