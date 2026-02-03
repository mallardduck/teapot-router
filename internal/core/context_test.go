package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetAction(t *testing.T) {
	ctx := context.Background()

	// Test setting action
	ctx = SetAction(ctx, "s3:GetObject")

	// Test getting action
	assert.Equal(t, "s3:GetObject", GetAction(ctx))
}

func TestGetActionEmpty(t *testing.T) {
	ctx := context.Background()

	// Test getting action from empty context
	assert.Equal(t, "", GetAction(ctx))
}

func TestSetAndGetRouteName(t *testing.T) {
	ctx := context.Background()

	// Test setting route name
	ctx = SetRouteName(ctx, "bucket.list")

	// Test getting route name
	assert.Equal(t, "bucket.list", GetRouteName(ctx))
}

func TestGetRouteNameEmpty(t *testing.T) {
	ctx := context.Background()

	// Test getting route name from empty context
	assert.Equal(t, "", GetRouteName(ctx))
}

func TestMultipleContextValues(t *testing.T) {
	asserts := assert.New(t)
	ctx := context.Background()

	// Set both action and route name
	ctx = SetAction(ctx, "s3:PutObject")
	ctx = SetRouteName(ctx, "object.put")

	// Verify both values are preserved
	asserts.Equal("s3:PutObject", GetAction(ctx))
	asserts.Equal("object.put", GetRouteName(ctx))
}

func TestInjectRouteMetadata(t *testing.T) {
	t.Run("injects both action and name", func(t *testing.T) {
		ctx := context.Background()
		route := &Route{
			Action: "test:Action",
			Name:   "test.name",
		}

		ctx = InjectRouteMetadata(ctx, route)

		assert.Equal(t, "test:Action", GetAction(ctx))
		assert.Equal(t, "test.name", GetRouteName(ctx))
	})

	t.Run("injects only action when name is empty", func(t *testing.T) {
		ctx := context.Background()
		route := &Route{
			Action: "test:Action",
			Name:   "",
		}

		ctx = InjectRouteMetadata(ctx, route)

		assert.Equal(t, "test:Action", GetAction(ctx))
		assert.Equal(t, "", GetRouteName(ctx))
	})

	t.Run("injects only name when action is empty", func(t *testing.T) {
		ctx := context.Background()
		route := &Route{
			Action: "",
			Name:   "test.name",
		}

		ctx = InjectRouteMetadata(ctx, route)

		assert.Equal(t, "", GetAction(ctx))
		assert.Equal(t, "test.name", GetRouteName(ctx))
	})

	t.Run("handles empty route gracefully", func(t *testing.T) {
		ctx := context.Background()
		route := &Route{
			Action: "",
			Name:   "",
		}

		ctx = InjectRouteMetadata(ctx, route)

		assert.Equal(t, "", GetAction(ctx))
		assert.Equal(t, "", GetRouteName(ctx))
	})

	t.Run("overwrites existing context values", func(t *testing.T) {
		ctx := context.Background()
		ctx = SetAction(ctx, "old:Action")
		ctx = SetRouteName(ctx, "old.name")

		route := &Route{
			Action: "new:Action",
			Name:   "new.name",
		}

		ctx = InjectRouteMetadata(ctx, route)

		assert.Equal(t, "new:Action", GetAction(ctx))
		assert.Equal(t, "new.name", GetRouteName(ctx))
	})
}
