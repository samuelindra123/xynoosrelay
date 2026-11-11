package authctx

import (
	"context"
)

type actorContextKey struct{}

// Actor represents the authenticated identity in the request context.
type Actor struct {
	UserID         string
	SessionID      string
	ManagedSession bool
}

// WithActor injects an Actor into context.
func WithActor(ctx context.Context, actor *Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// GetActor extracts the Actor from request context if present.
func GetActor(ctx context.Context) (*Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(*Actor)
	return actor, ok && actor != nil
}
