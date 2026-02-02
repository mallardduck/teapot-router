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
