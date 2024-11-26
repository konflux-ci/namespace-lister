package context

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ContextKey string

const (
	ContextKeyNamespaces      ContextKey = "namespaces"
	ContextKeyRunId           ContextKey = "run-id"
	ContextKeyUserInfo        ContextKey = "user-info"
	ContextKeyBuildUserClient ContextKey = "build-user-client"
)

type BuildUserClientFunc func(context.Context) (client.Client, error)

func WithBuildUserClientFunc(ctx context.Context, builder BuildUserClientFunc) context.Context {
	return into(ctx, ContextKeyBuildUserClient, builder)
}

func InvokeBuildUserClientFunc(ctx context.Context) (client.Client, error) {
	return get[BuildUserClientFunc](ctx, ContextKeyBuildUserClient)(ctx)
}

func WithUser(ctx context.Context, userInfo UserInfo) context.Context {
	return into(ctx, ContextKeyUserInfo, userInfo)
}

func User(ctx context.Context) UserInfo {
	return get[UserInfo](ctx, ContextKeyUserInfo)
}

func WithNamespaces(ctx context.Context, namespaces []corev1.Namespace) context.Context {
	return into(ctx, ContextKeyNamespaces, namespaces)
}

func Namespaces(ctx context.Context) []corev1.Namespace {
	return get[[]corev1.Namespace](ctx, ContextKeyNamespaces)
}

func WithRunId(ctx context.Context, runId string) context.Context {
	return into(ctx, ContextKeyRunId, runId)
}

func RunId(ctx context.Context) string {
	return get[string](ctx, ContextKeyRunId)
}

// aux
func into[T any](ctx context.Context, key ContextKey, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

func get[T any](ctx context.Context, key ContextKey) T {
	if v, ok := ctx.Value(key).(T); ok {
		return v
	}

	var t T
	return t
}
