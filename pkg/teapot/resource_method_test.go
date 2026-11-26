package teapot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mallardduck/teapot-router/pkg/teapot"
)

// StoreVerb accepts only POST or PUT. Any other non-empty value must fail
// loudly at registration, not silently drop the route.
func TestResourceRejectsInvalidStoreVerb(t *testing.T) {
	r := teapot.New()

	assert.Panics(t, func() {
		r.Resource("photos", &fullController{}, teapot.StoreVerb(teapot.HTTPMethod("GET")))
	})
}

// A lowercase value is a typo, not a valid method. It must also be rejected.
func TestResourceRejectsLowercaseStoreVerb(t *testing.T) {
	r := teapot.New()

	assert.Panics(t, func() {
		r.Resource("photos", &fullController{}, teapot.StoreVerb(teapot.HTTPMethod("put")))
	})
}

// The default (no StoreVerb) must not panic.
func TestResourceDefaultStoreVerbDoesNotPanic(t *testing.T) {
	r := teapot.New()

	assert.NotPanics(t, func() {
		r.Resource("photos", &fullController{})
	})
}
