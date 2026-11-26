package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuilderAcceptsTypedHandlerFunc pins that explicitly typed
// http.HandlerFunc values route correctly. This holds both before and after the
// move to http.Handler, so it guards the conversion.
func TestBuilderAcceptsTypedHandlerFunc(t *testing.T) {
	var def http.HandlerFunc = func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("default")) }
	var special http.HandlerFunc = func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("special")) }

	d := New(func(b *Builder) {
		b.Default(def)
		b.When(QueryEquals("type", "special")).Do(special)
	})

	for url, want := range map[string]string{"/": "default", "/?type=special": "special"} {
		req := httptest.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		d.ServeHTTP(rec, req)
		assert.Equal(t, want, rec.Body.String(), url)
	}
}
