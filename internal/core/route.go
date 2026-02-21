package core

import (
	"net/http"
	"strings"

	"github.com/mallardduck/teapot-router/pkg/dispatch"
)

// Route represents a single route with all its metadata
type Route struct {
	Method         string
	Pattern        string // Original pattern (e.g., "/{bucket}/{key:.*}")
	ChiPattern     string // Chi-compatible pattern (e.g., "/{bucket}/*")
	Handler        http.Handler
	Name           string
	Action         string
	QueryMatchers  []dispatch.Matcher
	Middlewares    []func(http.Handler) http.Handler
	WildcardParams map[string]bool // Track which params are wildcards (e.g., "key" -> true)
}

// TranslatePattern converts our {key:.*} syntax to Chi's wildcard syntax
// Returns the Chi pattern and a map of wildcard parameter names
func TranslatePattern(pattern string) (string, map[string]bool) {
	wildcardParams := make(map[string]bool)

	// Find all {key:.*} patterns and extract the parameter names
	result := pattern
	start := 0
	for {
		idx := strings.Index(result[start:], "{")
		if idx == -1 {
			break
		}
		idx += start

		endIdx := strings.Index(result[idx:], "}")
		if endIdx == -1 {
			break
		}
		endIdx += idx

		// Extract the parameter definition
		paramDef := result[idx+1 : endIdx]

		// Check if it's a wildcard pattern {key:.*}
		if strings.HasSuffix(paramDef, ":.*") {
			paramName := strings.TrimSuffix(paramDef, ":.*")
			wildcardParams[paramName] = true

			// Replace {key:.*} with /* (Chi wildcard syntax)
			// We need to keep everything before {key:.*} and replace the rest
			prefix := result[:idx]
			// Remove trailing slash if present before adding /*
			prefix = strings.TrimSuffix(prefix, "/")
			result = prefix + "/*"
			break // Only one wildcard per pattern is supported by Chi
		}

		// Check for Go 1.22+ exact match operator {$}
		if paramDef == "$" {
			// Chi handles exact matches with the standard trailing slash / pattern.
			// We translate /{ $ } to / so that Chi routes correctly.
			// Example: /users/{$} becomes /users/
			prefix := result[:idx]
			suffix := result[endIdx+1:]

			// If we have something like /admin/{$}, we want it to be /admin/
			// If it's just /{$}, it becomes /
			result = prefix + suffix

			// Don't advance start, re-scan from where suffix begins
			start = idx
			continue
		}

		start = endIdx + 1
	}

	return result, wildcardParams
}
