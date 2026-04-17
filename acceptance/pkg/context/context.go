package context

import (
	"context"
	"crypto/tls"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ContextKey string

const (
	ContextKeyNamespaces             ContextKey = "namespaces"
	ContextKeyRunId                  ContextKey = "run-id"
	ContextKeyUserInfo               ContextKey = "user-info"
	ContextKeyBuildUserClient        ContextKey = "build-user-client"
	ContextKeyNamespaceListerAddress ContextKey = "namespace-lister-address"
	ContextKeyTLSConfig              ContextKey = "tls-config"
	ContextKeyMetricsAddress         ContextKey = "metrics-address"
	ContextKeyHTTPResponse           ContextKey = "http-response"
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

func WithNamespaceListerAddress(ctx context.Context, address string) context.Context {
	return into(ctx, ContextKeyNamespaceListerAddress, address)
}

func NamespaceListerAddress(ctx context.Context) string {
	return get[string](ctx, ContextKeyNamespaceListerAddress)
}

func WithTLSConfig(ctx context.Context, cfg *tls.Config) context.Context {
	return into(ctx, ContextKeyTLSConfig, cfg)
}

func TLSConfig(ctx context.Context) *tls.Config {
	return get[*tls.Config](ctx, ContextKeyTLSConfig)
}

func WithMetricsAddress(ctx context.Context, address string) context.Context {
	return into(ctx, ContextKeyMetricsAddress, address)
}

func MetricsAddress(ctx context.Context) string {
	return get[string](ctx, ContextKeyMetricsAddress)
}

func WithHTTPResponse(ctx context.Context, statusCode int, body []byte) context.Context {
	b := make([]byte, len(body))
	copy(b, body)
	resp := HTTPResponse{StatusCode: statusCode, Body: b}
	return into(ctx, ContextKeyHTTPResponse, resp)
}

func GetHTTPResponse(ctx context.Context) HTTPResponse {
	return get[HTTPResponse](ctx, ContextKeyHTTPResponse)
}

type HTTPResponse struct {
	StatusCode int
	Body       []byte
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
