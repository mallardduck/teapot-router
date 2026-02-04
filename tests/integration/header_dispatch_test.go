package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// TestHeaderDispatchRouting verifies end-to-end header-based dispatch.
// Models S3's CopyObject pattern: PUT to the same path, dispatched by
// header presence (HeaderExists) and header value (HeaderEquals).
func TestHeaderDispatchRouting(t *testing.T) {
	r := teapot.New()

	r.Dispatch("PUT", "/{bucket}/{key:.*}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
		d.Default(func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "PUT_OBJECT|bucket=%s|key=%s",
				teapot.URLParam(req, "bucket"),
				teapot.URLParam(req, "key"),
			)
		}).Name("hdr.put.object").Action("s3:PutObject")

		// Copy: header must exist (specificity 1)
		d.When(m.HeaderExists("X-Amz-Copy-Source")).Do(func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "COPY_OBJECT|src=%s", req.Header.Get("X-Amz-Copy-Source"))
		}).Name("hdr.put.copy").Action("s3:CopyObject")

		// Copy + Replace: two exact header values (specificity 2+2=4)
		d.When(
			m.HeaderEquals("X-Amz-Copy-Source", "/src-bucket/src-key"),
			m.HeaderEquals("X-Amz-Metadata-Directive", "REPLACE"),
		).Do(func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "COPY_REPLACE|bucket=%s",
				teapot.URLParam(req, "bucket"),
			)
		}).Name("hdr.put.copy-replace").Action("s3:CopyObjectReplace")
	})

	r.Finalize()

	t.Run("simple PUT with no copy headers", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/mybucket/path/to/obj.txt", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "PUT_OBJECT|bucket=mybucket|key=path/to/obj.txt", w.Body.String())
	})

	t.Run("copy with header present but non-matching value", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/dest/newkey.txt", nil)
		req.Header.Set("X-Amz-Copy-Source", "/other-bucket/other-key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// HeaderEquals for copy-replace doesn't match (wrong source) → falls to HeaderExists
		assert.Equal(t, "COPY_OBJECT|src=/other-bucket/other-key", w.Body.String())
	})

	t.Run("copy-replace with exact header values wins on specificity", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/dest/replaced.txt", nil)
		req.Header.Set("X-Amz-Copy-Source", "/src-bucket/src-key")
		req.Header.Set("X-Amz-Metadata-Directive", "REPLACE")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "COPY_REPLACE|bucket=dest", w.Body.String())
	})

	t.Run("route listing exposes header params", func(t *testing.T) {
		routes := r.Routes()
		require.Len(t, routes, 3)

		byName := make(map[string]teapot.RouteInfo)
		for _, rt := range routes {
			byName[rt.Name] = rt
		}

		assert.Empty(t, byName["hdr.put.object"].HeaderParams)

		copyRoute := byName["hdr.put.copy"]
		require.Len(t, copyRoute.HeaderParams, 1)
		assert.Equal(t, "X-Amz-Copy-Source", copyRoute.HeaderParams[0].Key)
		assert.Equal(t, "", copyRoute.HeaderParams[0].Value) // existence check

		copyReplace := byName["hdr.put.copy-replace"]
		require.Len(t, copyReplace.HeaderParams, 2)
		assert.Equal(t, "X-Amz-Copy-Source", copyReplace.HeaderParams[0].Key)
		assert.Equal(t, "/src-bucket/src-key", copyReplace.HeaderParams[0].Value)
		assert.Equal(t, "X-Amz-Metadata-Directive", copyReplace.HeaderParams[1].Key)
		assert.Equal(t, "REPLACE", copyReplace.HeaderParams[1].Value)
	})
}

// TestHeaderQueryCombinedDispatch verifies AND composition across header and
// query matchers.  A route gated on both a header and a query param has
// specificity 2 and beats single-matcher routes.
func TestHeaderQueryCombinedDispatch(t *testing.T) {
	r := teapot.New()

	r.Dispatch("PUT", "/{bucket}/{key:.*}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("default"))
		})

		// Query-only: tagging (specificity 1)
		d.When(m.QueryExists("tagging")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("tagging"))
		})

		// Header-only: copy (specificity 1)
		d.When(m.HeaderExists("X-Amz-Copy-Source")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("copy"))
		})

		// Header AND query: copy + acl (specificity 1+1=2)
		d.When(
			m.HeaderExists("X-Amz-Copy-Source"),
			m.QueryExists("acl"),
		).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("copy-with-acl"))
		})
	})

	r.Finalize()

	tests := []struct {
		name     string
		url      string
		headers  map[string]string
		expected string
	}{
		{"no params → default", "/b/k", nil, "default"},
		{"tagging query only", "/b/k?tagging=t", nil, "tagging"},
		{"copy header only", "/b/k", map[string]string{"X-Amz-Copy-Source": "/src/k"}, "copy"},
		{"copy header + acl query → most specific", "/b/k?acl", map[string]string{"X-Amz-Copy-Source": "/src/k"}, "copy-with-acl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", tt.url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

// TestHeaderSpecificityOrdering verifies that HeaderEquals (specificity 2)
// beats HeaderExists (specificity 1) when both match the same request.
func TestHeaderSpecificityOrdering(t *testing.T) {
	r := teapot.New()

	r.Dispatch("GET", "/items", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
		d.Default(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("default"))
		})

		d.When(m.HeaderExists("X-Version")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("versioned"))
		})

		d.When(m.HeaderEquals("X-Version", "v2")).Do(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("v2"))
		})
	})

	r.Finalize()

	t.Run("no header → default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "default", w.Body.String())
	})

	t.Run("X-Version: v1 → exists match only", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items", nil)
		req.Header.Set("X-Version", "v1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "versioned", w.Body.String())
	})

	t.Run("X-Version: v2 → value match wins over exists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items", nil)
		req.Header.Set("X-Version", "v2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, "v2", w.Body.String())
	})
}
