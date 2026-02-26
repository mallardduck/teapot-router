package teapot_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestFormatRoutesHelpersBoundaryConditions tests boundary conditions
// identified by gremlins in routes_helpers.go (lines 24:29, 26:27, 50:29, 52:27, etc.)
func TestFormatRoutesHelpersBoundaryConditions(t *testing.T) {
	t.Run("format routes with exactly zero routes", func(t *testing.T) {
		var routes []teapot.RouteInfo

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		// Should still have header
		output := buf.String()
		assert.Contains(t, output, "METHOD")
		assert.Contains(t, output, "PATTERN")
	})

	t.Run("format routes with exactly one route", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test", Name: "test.route"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "/test")
		assert.Contains(t, output, "test.route")
	})

	t.Run("format routes with exactly two routes", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test1", Name: "route1"},
			{Method: "POST", Pattern: "/test2", Name: "route2"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "route1")
		assert.Contains(t, output, "route2")
	})

	t.Run("format routes with many routes", func(t *testing.T) {
		routes := make([]teapot.RouteInfo, 100)
		for i := 0; i < 100; i++ {
			routes[i] = teapot.RouteInfo{
				Method:  "GET",
				Pattern: "/test",
				Name:    "route",
			}
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		// Should not crash with many routes
		output := buf.String()
		assert.NotEmpty(t, output)
	})
}

// TestFormatRoutesTableEdgeCases tests edge cases for FormatRoutesTable
func TestFormatRoutesTableEdgeCases(t *testing.T) {
	t.Run("empty name and action", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test", Name: "", Action: ""},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// Empty name/action should show as "-"
		assert.Contains(t, output, "-")
	})

	t.Run("route with query params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "foo=bar")
	})

	t.Run("route with multiple query params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
					{Key: "baz", Value: "qux"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "foo=bar")
		assert.Contains(t, output, "&")
		assert.Contains(t, output, "baz=qux")
	})

	t.Run("route with existence-only query param", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "acl", Value: ""}, // Existence check
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "acl")
		// Should not contain "=" for existence-only param
	})

	t.Run("routes with identical patterns different methods", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test"},
			{Method: "POST", Pattern: "/test"},
			{Method: "PUT", Pattern: "/test"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "PUT")
	})
}

// TestFormatRoutesJSONEdgeCases tests edge cases for FormatRoutesJSON
func TestFormatRoutesJSONEdgeCases(t *testing.T) {
	t.Run("empty routes list", func(t *testing.T) {
		routes := []teapot.RouteInfo{}

		var buf bytes.Buffer
		err := teapot.FormatRoutesJSON(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"count": 0`)
		assert.Contains(t, output, `"routes": []`)
	})

	t.Run("single route", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test", Name: "test.route"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesJSON(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"count": 1`)
		assert.Contains(t, output, `"GET"`)
		assert.Contains(t, output, `"/test"`)
		assert.Contains(t, output, `"test.route"`)
	})

	t.Run("routes with special characters", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test/{id}", Name: "test\"quote"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesJSON(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// JSON encoder should escape special characters
		assert.Contains(t, output, `\"quote`)
	})

	t.Run("sorting by pattern then method", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "POST", Pattern: "/b"},
			{Method: "GET", Pattern: "/b"},
			{Method: "POST", Pattern: "/a"},
			{Method: "GET", Pattern: "/a"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesJSON(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// Should be sorted: /a GET, /a POST, /b GET, /b POST
		aIdx := strings.Index(output, `"/a"`)
		bIdx := strings.Index(output, `"/b"`)
		assert.Less(t, aIdx, bIdx, "pattern /a should come before /b")
	})
}

// TestFormatRoutesCompactEdgeCases tests edge cases for FormatRoutesCompact
func TestFormatRoutesCompactEdgeCases(t *testing.T) {
	t.Run("empty routes list", func(t *testing.T) {
		routes := []teapot.RouteInfo{}

		var buf bytes.Buffer
		err := teapot.FormatRoutesCompact(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("single route without name", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesCompact(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "/test")
		assert.Contains(t, output, "-") // No name shows as "-"
	})

	t.Run("route with query params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesCompact(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "/test?foo=bar")
	})

	t.Run("route without query params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test"},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesCompact(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// Should not have "?" when no query params
		assert.NotContains(t, output, "?")
	})

	t.Run("multiple routes with mixed query params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test1",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
				},
			},
			{
				Method:  "GET",
				Pattern: "/test2",
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesCompact(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "/test1?foo=bar")
		lines := strings.Split(output, "\n")
		// Should have at least 2 non-empty lines
		nonEmpty := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmpty++
			}
		}
		assert.GreaterOrEqual(t, nonEmpty, 2)
	})
}

// TestFormatQueryParamsEdgeCases tests the formatQueryParams helper edge cases
func TestFormatQueryParamsEdgeCases(t *testing.T) {
	t.Run("exactly zero params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{Method: "GET", Pattern: "/test", QueryParams: []teapot.QueryParam{}},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// Zero params should show as "-"
		assert.Contains(t, output, "-")
	})

	t.Run("exactly one param with value", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "foo=bar")
		// Should not have "&" with only one param
		queryParamSection := extractQueryParamSection(output)
		assert.NotContains(t, queryParamSection, "&")
	})

	t.Run("exactly one param without value", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "acl", Value: ""},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "acl")
	})

	t.Run("exactly two params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "foo", Value: "bar"},
					{Key: "baz", Value: "qux"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "foo=bar")
		assert.Contains(t, output, "baz=qux")
		// Should have exactly one "&"
		queryParamSection := extractQueryParamSection(output)
		assert.Equal(t, 1, strings.Count(queryParamSection, "&"))
	})

	t.Run("many params", func(t *testing.T) {
		routes := []teapot.RouteInfo{
			{
				Method:  "GET",
				Pattern: "/test",
				QueryParams: []teapot.QueryParam{
					{Key: "a", Value: "1"},
					{Key: "b", Value: "2"},
					{Key: "c", Value: "3"},
					{Key: "d", Value: "4"},
					{Key: "e", Value: "5"},
				},
			},
		}

		var buf bytes.Buffer
		err := teapot.FormatRoutesTable(&buf, routes)
		require.NoError(t, err)

		output := buf.String()
		// Should have all params
		assert.Contains(t, output, "a=1")
		assert.Contains(t, output, "e=5")
		// Should have correct number of "&" separators (n-1)
		queryParamSection := extractQueryParamSection(output)
		assert.Equal(t, 4, strings.Count(queryParamSection, "&"))
	})
}

// TestListRoutesOptionsBaseURL tests the BaseURL / BaseURLFunc precedence in
// ListRoutesOptions as surfaced by NewListRoutesHandlerWithRoutes.
func TestListRoutesOptionsBaseURL(t *testing.T) {
	routes := []teapot.RouteInfo{
		{Method: "GET", Pattern: "/ping", Name: "ping"},
	}

	htmlReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/routes", nil)
		r.Header.Set("Accept", "text/html")
		return r
	}

	t.Run("static BaseURL produces clickable link", func(t *testing.T) {
		handler := teapot.NewListRoutesHandlerWithRoutes(routes, &teapot.ListRoutesOptions{
			BaseURL: "http://localhost:9000",
		})
		w := httptest.NewRecorder()
		handler(w, htmlReq())

		body := w.Body.String()
		assert.Contains(t, body, `href="http://localhost:9000/ping"`)
	})

	t.Run("BaseURLFunc overrides static BaseURL", func(t *testing.T) {
		handler := teapot.NewListRoutesHandlerWithRoutes(routes, &teapot.ListRoutesOptions{
			BaseURL: "http://should-not-appear.example.com",
			BaseURLFunc: func(r *http.Request) string {
				return "https://dynamic.example.com"
			},
		})
		w := httptest.NewRecorder()
		handler(w, htmlReq())

		body := w.Body.String()
		assert.Contains(t, body, `href="https://dynamic.example.com/ping"`)
		assert.NotContains(t, body, "should-not-appear")
	})

	t.Run("BaseURLFunc receives the request", func(t *testing.T) {
		req := htmlReq()
		req.Host = "myapp.example.com"

		var capturedHost string
		handler := teapot.NewListRoutesHandlerWithRoutes(routes, &teapot.ListRoutesOptions{
			BaseURLFunc: func(r *http.Request) string {
				capturedHost = r.Host
				return "http://" + r.Host
			},
		})
		w := httptest.NewRecorder()
		handler(w, req)

		assert.Equal(t, "myapp.example.com", capturedHost)
		assert.Contains(t, w.Body.String(), `href="http://myapp.example.com/ping"`)
	})

	t.Run("neither BaseURL nor BaseURLFunc yields no links", func(t *testing.T) {
		handler := teapot.NewListRoutesHandlerWithRoutes(routes, nil)
		w := httptest.NewRecorder()
		handler(w, htmlReq())

		assert.NotContains(t, w.Body.String(), "<a href=")
	})
}

// Helper function to extract the query param section from table output
func extractQueryParamSection(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "GET") || strings.Contains(line, "POST") {
			// This is a data row
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2] // Query params column
			}
		}
	}
	return ""
}
