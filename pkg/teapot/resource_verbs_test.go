package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// rec records which action ran. Each controller below embeds it and
// implements only the actions it needs, so interface detection is exact.
type rec struct{ calls []string }

func (c *rec) do(name string, w http.ResponseWriter) {
	c.calls = append(c.calls, name)
	w.WriteHeader(200)
}

type updateCtrl struct{ rec }

func (c *updateCtrl) Update(w http.ResponseWriter, _ *http.Request) { c.do("update", w) }

type splitCtrl struct{ rec }

func (c *splitCtrl) Replace(w http.ResponseWriter, _ *http.Request) { c.do("replace", w) }
func (c *splitCtrl) Patch(w http.ResponseWriter, _ *http.Request)   { c.do("patch", w) }

type splitUpdateCtrl struct{ rec }

func (c *splitUpdateCtrl) Replace(w http.ResponseWriter, _ *http.Request) { c.do("replace", w) }
func (c *splitUpdateCtrl) Patch(w http.ResponseWriter, _ *http.Request)   { c.do("patch", w) }
func (c *splitUpdateCtrl) Update(w http.ResponseWriter, _ *http.Request)  { c.do("update", w) }

type replaceUpdateCtrl struct{ rec }

func (c *replaceUpdateCtrl) Replace(w http.ResponseWriter, _ *http.Request) { c.do("replace", w) }
func (c *replaceUpdateCtrl) Update(w http.ResponseWriter, _ *http.Request)  { c.do("update", w) }

type headCtrl struct{ rec }

func (c *headCtrl) Show(w http.ResponseWriter, _ *http.Request) { c.do("show", w) }
func (c *headCtrl) Head(w http.ResponseWriter, _ *http.Request) { c.do("head", w) }

func serveCode(r *teapot.Router, method, path string) int {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w.Code
}

func TestResourceUpdateVerbModes(t *testing.T) {
	tests := []struct {
		name       string
		controller any
		opts       []teapot.ResourceOption
		// want maps "METHOD" to the action expected to run for it.
		want map[string]string
	}{
		{
			name:       "update alone handles PUT and PATCH",
			controller: &updateCtrl{},
			want:       map[string]string{"PUT": "update", "PATCH": "update"},
		},
		{
			name:       "replace and patch split PUT and PATCH",
			controller: &splitCtrl{},
			want:       map[string]string{"PUT": "replace", "PATCH": "patch"},
		},
		{
			name:       "replace and patch take precedence over update",
			controller: &splitUpdateCtrl{},
			want:       map[string]string{"PUT": "replace", "PATCH": "patch"},
		},
		{
			name:       "update covers the verb replace does not",
			controller: &replaceUpdateCtrl{},
			want:       map[string]string{"PUT": "replace", "PATCH": "update"},
		},
		{
			name:       "Except(replace) hands PUT back to update",
			controller: &replaceUpdateCtrl{},
			opts:       []teapot.ResourceOption{teapot.Except("replace")},
			want:       map[string]string{"PUT": "update", "PATCH": "update"},
		},
		{
			name:       "Except(patch) leaves PATCH to update",
			controller: &splitUpdateCtrl{},
			opts:       []teapot.ResourceOption{teapot.Except("patch")},
			want:       map[string]string{"PUT": "replace", "PATCH": "update"},
		},
		{
			name:       "Only(replace) registers PUT and no PATCH",
			controller: &splitCtrl{},
			opts:       []teapot.ResourceOption{teapot.Only("replace")},
			want:       map[string]string{"PUT": "replace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := teapot.New()
			r.Resource("photos", tt.controller, tt.opts...)

			for _, method := range []string{"PUT", "PATCH"} {
				want, expected := tt.want[method]
				calls := callsFor(tt.controller)
				*calls = nil

				code := serveCode(r, method, "/photos/1")
				if !expected {
					assert.NotEqual(t, 200, code, "%s should not be registered", method)
					assert.Empty(t, *calls, "%s should not reach any handler", method)
					continue
				}
				assert.Equal(t, 200, code, "%s /photos/1", method)
				assert.Equal(t, []string{want}, *calls, "%s /photos/1", method)
			}
		})
	}
}

// callsFor returns the recorded calls for a controller built in this file.
func callsFor(controller any) *[]string {
	switch c := controller.(type) {
	case *updateCtrl:
		return &c.calls
	case *splitCtrl:
		return &c.calls
	case *splitUpdateCtrl:
		return &c.calls
	case *replaceUpdateCtrl:
		return &c.calls
	case *headCtrl:
		return &c.calls
	}
	panic("unknown controller type")
}

func TestResourceHeadWhenImplemented(t *testing.T) {
	r := teapot.New()
	ctrl := &headCtrl{}
	r.Resource("photos", ctrl)

	code := serveCode(r, "HEAD", "/photos/1")
	assert.Equal(t, 200, code)
	assert.Equal(t, []string{"head"}, ctrl.calls)

	// GET is unaffected by the HEAD route.
	ctrl.calls = nil
	serveCode(r, "GET", "/photos/1")
	assert.Equal(t, []string{"show"}, ctrl.calls)
}

func TestResourceHeadFallsBackToShowWhenNotImplemented(t *testing.T) {
	r := teapot.New()
	ctrl := &showOnlyCtrl{}
	r.Resource("photos", ctrl)

	// Without a Head method, HEAD is served by the GET (show) route.
	assert.Equal(t, 200, serveCode(r, "HEAD", "/photos/1"))
	assert.Equal(t, []string{"show"}, ctrl.calls)
}

type showOnlyCtrl struct{ rec }

func (c *showOnlyCtrl) Show(w http.ResponseWriter, _ *http.Request) { c.do("show", w) }

func TestResourceHeadRouteName(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", &headCtrl{})

	require.Equal(t, "/photos/7", r.MustURL("photos.head", "photo", "7"))
}

func TestResourceReplaceAndPatchNamesAndNames(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", &splitCtrl{}, teapot.Names(map[string]string{
		"replace": "put",
		"patch":   "partial",
	}))

	assert.Equal(t, "/photos/1", r.MustURL("photos.put", "photo", "1"))
	assert.Equal(t, "/photos/1", r.MustURL("photos.partial", "photo", "1"))
}

func TestResourceUpdateNameWhenShared(t *testing.T) {
	r := teapot.New()
	r.Resource("photos", &updateCtrl{})

	assert.Equal(t, "/photos/1", r.MustURL("photos.update", "photo", "1"))
}
