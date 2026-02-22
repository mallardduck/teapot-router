package tests

import (
	"testing"

	"github.com/mallardduck/teapot-router/internal/testutil"
	"github.com/mallardduck/teapot-router/pkg/teapot"
)

func TestMountWithNamePrefix(t *testing.T) {
	r := teapot.New()
	sub := teapot.New()

	// Handler for testing
	handler := testutil.OKResponseHandler

	// Add a route to sub
	sub.GET("/users", handler).Name("list")

	// Mount sub into r with a name prefix... but how?
	// Currently Mount only takes pattern and handler.
	r.Mount("/api", sub)

	// Check if the route is registered as "api.list" (it won't be, it'll be just "list" unless r has a namePrefix)
	_, err := r.URL("api.list")
	if err != nil {
		t.Logf("Expected api.list but got error: %v", err)
	} else {
		t.Log("Successfully found api.list")
	}

	// Try with NamedGroup
	r.NamedGroup("/admin", "admin", func(r *teapot.Router) {
		r.Mount("/dashboard", sub)
	})

	_, err = r.URL("admin.list")
	if err != nil {
		t.Logf("Expected admin.list but got error: %v", err)
	} else {
		t.Log("Successfully found admin.list")
	}

	// Test MountNamed
	r.MountNamed("/v1", "api.v1", sub)

	_, err = r.URL("api.v1.list")
	if err != nil {
		t.Logf("Expected api.v1.list but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found api.v1.list")
	}

	// Test Homing with MountNamed
	sub.GET("/profile", handler).Name("profile")

	_, err = r.URL("api.v1.profile")
	if err != nil {
		t.Logf("Expected api.v1.profile but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found api.v1.profile")
	}

	// Test Mount with .Name() builder
	sub2 := teapot.New()
	sub2.GET("/items", handler).Name("list")

	r.Mount("/v2", sub2).Name("api.v2")

	_, err = r.URL("api.v2.list")
	if err != nil {
		t.Logf("Expected api.v2.list but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found api.v2.list")
	}

	// Test Homing with Mount builder
	sub2.GET("/detail", handler).Name("detail")
	_, err = r.URL("api.v2.detail")
	if err != nil {
		t.Logf("Expected api.v2.detail but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found api.v2.detail")
	}

	// Test Hierarchical Mount builder
	r0 := teapot.New()
	r1 := teapot.New()
	r2 := teapot.New()

	r0.Mount("/r1", r1).Name("p0")
	r1.Mount("/r2", r2).Name("p1")
	r2.GET("/r3", handler).Name("list")

	// Current name should be p0.p1.list
	_, err = r0.URL("p0.p1.list")
	if err != nil {
		t.Logf("Expected p0.p1.list but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found p0.p1.list")
	}

	// Now RENAME the mount in r1
	// Find the mount builder? Wait, Mount returns a new one each time.
	// But r1.Mount("/r2", r2) is what we want to rename.
	r1.Mount("/r2", r2).Name("p1_new")

	// Check r0 again. Should have p0.p1_new.list
	_, err = r0.URL("p0.p1_new.list")
	if err != nil {
		t.Logf("Expected p0.p1_new.list but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found p0.p1_new.list")
	}

	// What happens to the old name? It should be gone.
	_, err = r0.URL("p0.p1.list")
	if err == nil {
		t.Log("Expected p0.p1.list to be gone, but it's still there")
		t.Fail()
	}

	// Test deep homing with prefix change
	r2.GET("/r4", handler).Name("new_route")
	_, err = r0.URL("p0.p1_new.new_route")
	if err != nil {
		t.Logf("Expected p0.p1_new.new_route but got error: %v", err)
		t.Fail()
	} else {
		t.Log("Successfully found p0.p1_new.new_route")
	}
}
