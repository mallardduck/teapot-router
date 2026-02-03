package core

import "context"

// Context keys for injecting values into request context
type contextKey int

const (
	// ActionKey is the context key for storing S3 action values
	ActionKey contextKey = iota
	// RouteNameKey is the context key for storing route name values
	RouteNameKey
)

// SetAction sets the S3 action in the context
func SetAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, ActionKey, action)
}

// GetAction retrieves the S3 action from the context
func GetAction(ctx context.Context) string {
	if action, ok := ctx.Value(ActionKey).(string); ok {
		return action
	}
	return ""
}

// SetRouteName sets the route name in the context
func SetRouteName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, RouteNameKey, name)
}

// GetRouteName retrieves the route name from the context
func GetRouteName(ctx context.Context) string {
	if name, ok := ctx.Value(RouteNameKey).(string); ok {
		return name
	}
	return ""
}

// InjectRouteMetadata adds route metadata (Action, Name) to the context if present.
// Returns the updated context.
func InjectRouteMetadata(ctx context.Context, route *Route) context.Context {
	if route.Action != "" {
		ctx = SetAction(ctx, route.Action)
	}
	if route.Name != "" {
		ctx = SetRouteName(ctx, route.Name)
	}
	return ctx
}
