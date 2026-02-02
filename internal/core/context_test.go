package core

import (
	"context"
	"testing"
)

func TestSetAndGetAction(t *testing.T) {
	ctx := context.Background()

	// Test setting action
	ctx = SetAction(ctx, "s3:GetObject")

	// Test getting action
	if got := GetAction(ctx); got != "s3:GetObject" {
		t.Errorf("GetAction() = %q, want %q", got, "s3:GetObject")
	}
}

func TestGetActionEmpty(t *testing.T) {
	ctx := context.Background()

	// Test getting action from empty context
	if got := GetAction(ctx); got != "" {
		t.Errorf("GetAction() = %q, want empty string", got)
	}
}

func TestSetAndGetRouteName(t *testing.T) {
	ctx := context.Background()

	// Test setting route name
	ctx = SetRouteName(ctx, "bucket.list")

	// Test getting route name
	if got := GetRouteName(ctx); got != "bucket.list" {
		t.Errorf("GetRouteName() = %q, want %q", got, "bucket.list")
	}
}

func TestGetRouteNameEmpty(t *testing.T) {
	ctx := context.Background()

	// Test getting route name from empty context
	if got := GetRouteName(ctx); got != "" {
		t.Errorf("GetRouteName() = %q, want empty string", got)
	}
}

func TestMultipleContextValues(t *testing.T) {
	ctx := context.Background()

	// Set both action and route name
	ctx = SetAction(ctx, "s3:PutObject")
	ctx = SetRouteName(ctx, "object.put")

	// Verify both values are preserved
	if got := GetAction(ctx); got != "s3:PutObject" {
		t.Errorf("GetAction() = %q, want %q", got, "s3:PutObject")
	}
	if got := GetRouteName(ctx); got != "object.put" {
		t.Errorf("GetRouteName() = %q, want %q", got, "object.put")
	}
}
