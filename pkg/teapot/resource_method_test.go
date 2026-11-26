package teapot_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// Resource only accepts POST or PUT for Store, and PUT or POST for Update.
// Any other non-empty value must fail loudly at registration, not silently drop
// the route.
func TestResourceRejectsInvalidStoreMethod(t *testing.T) {
	r := teapot.New()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	assert.Panics(t, func() {
		r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
			Store:       h,
			StoreMethod: teapot.HTTPMethod("GET"),
		})
	})
}

func TestResourceRejectsInvalidUpdateMethod(t *testing.T) {
	r := teapot.New()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	assert.Panics(t, func() {
		r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
			Update:       h,
			UpdateMethod: teapot.HTTPMethod("PATCH"),
		})
	})
}

// A lowercase value is a typo, not a valid method. It must also be rejected.
func TestResourceRejectsLowercaseMethod(t *testing.T) {
	r := teapot.New()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	assert.Panics(t, func() {
		r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{
			Store:       h,
			StoreMethod: teapot.HTTPMethod("put"),
		})
	})
}

// Empty means "use the default", which must keep working.
func TestResourceEmptyMethodUsesDefault(t *testing.T) {
	r := teapot.New()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	assert.NotPanics(t, func() {
		r.Resource("photos", "/photos", "photo", teapot.ResourceHandlers{Store: h, Update: h})
	})
}
