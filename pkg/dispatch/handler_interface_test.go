package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// bodyHandler is a non-function http.Handler, like a router or a
// handler struct, that has to work wherever a handler is accepted.
type bodyHandler string

func (h bodyHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(h))
}

func serveBody(d *Dispatcher, url string) string {
	rec := httptest.NewRecorder()
	d.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec.Body.String()
}

// FuncDefault and FuncDo should be equivalent to passing an http.HandlerFunc
// to Default and Do.
func TestFuncVariantsRegisterHandlers(t *testing.T) {
	d := New(func(b *Builder) {
		b.FuncDefault(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("default")) })
		b.When(QueryExists("special")).FuncDo(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("special"))
		})
	})

	assert.Equal(t, "default", serveBody(d, "/"))
	assert.Equal(t, "special", serveBody(d, "/?special"))
}

// Default and Do accept any http.Handler, including non-function handlers.
func TestHandlerInterfaceAcceptedByDefaultAndDo(t *testing.T) {
	d := New(func(b *Builder) {
		b.Default(bodyHandler("default"))
		b.When(QueryExists("special")).Do(bodyHandler("special"))
	})

	assert.Equal(t, "default", serveBody(d, "/"))
	assert.Equal(t, "special", serveBody(d, "/?special"))
}
