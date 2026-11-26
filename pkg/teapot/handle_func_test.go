package teapot_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	teapot "github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestHandleFuncServesFunction(t *testing.T) {
	r := teapot.New()
	r.HandleFunc("GET", "/hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hi"))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/hello", nil))

	if rec.Body.String() != "hi" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "hi")
	}
}

func TestHandleFuncReturnsRouteBuilderForNaming(t *testing.T) {
	r := teapot.New()
	r.HandleFunc("GET", "/users/{id}", func(http.ResponseWriter, *http.Request) {}).Name("users.show")

	if got := r.MustURL("users.show", "id", "42"); got != "/users/42" {
		t.Fatalf("MustURL = %q, want %q", got, "/users/42")
	}
}

func TestHandleFuncMatchesHandleWithHandlerFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("same")) }

	viaFunc := teapot.New()
	viaFunc.HandleFunc("PUT", "/x", fn)

	viaHandle := teapot.New()
	viaHandle.Handle("PUT", "/x", http.HandlerFunc(fn))

	for name, r := range map[string]*teapot.Router{"HandleFunc": viaFunc, "Handle": viaHandle} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("PUT", "/x", nil))
		if rec.Body.String() != "same" {
			t.Fatalf("%s: body = %q, want %q", name, rec.Body.String(), "same")
		}
	}
}
