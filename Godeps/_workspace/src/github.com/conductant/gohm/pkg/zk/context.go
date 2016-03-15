package zk

import (
	"golang.org/x/net/context"
	"time"
)

type timeoutContextKey int

const (
	TimeoutContextKey timeoutContextKey = 1
)

func ContextGetTimeout(ctx context.Context) time.Duration {
	if v, ok := (ctx.Value(TimeoutContextKey)).(time.Duration); ok {
		return v
	}
	return DefaultTimeout
}

func ContextPutTimeout(ctx context.Context, t time.Duration) context.Context {
	return context.WithValue(ctx, TimeoutContextKey, t)
}
