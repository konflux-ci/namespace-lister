package main

import (
	"context"
	"os"
	"time"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/prometheus/client_golang/prometheus"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// buildAndStartSynchronizedAccessCache builds a SynchronizedAccessCache.
// It registers handlers on events on resources that will trigger an AccessCache synchronization.
func buildAndStartSynchronizedAccessCache(ctx context.Context, resourceCache crcache.Cache, registry prometheus.Registerer) (*cache.SynchronizedAccessCache, error) {
	acm, err := buildAndRegisterAccessCacheMetrics(registry)
	if err != nil {
		return nil, err
	}

	aur := &CRAuthRetriever{resourceCache}
	sae := rbac.NewSubjectAccessEvaluator(aur, aur, aur, aur, "")
	synchCache := cache.NewSynchronizedAccessCache(
		sae,
		resourceCache, cache.CacheSynchronizerOptions{
			Logger:       getLoggerFromContext(ctx),
			ResyncPeriod: getResyncPeriodFromEnvOrZero(ctx),
			Metrics:      acm,
		},
	)

	// register event handlers on resource cache
	oo := []client.Object{
		&corev1.Namespace{},
		&rbacv1.ClusterRoleBinding{},
		&rbacv1.RoleBinding{},
		&rbacv1.ClusterRole{},
		&rbacv1.Role{},
	}
	for _, o := range oo {
		i, err := resourceCache.GetInformer(ctx, o)
		if err != nil {
			return nil, err
		}

		if _, err := i.AddEventHandler(synchCache.EventHandlerFuncs()); err != nil {
			return nil, err
		}
	}
	synchCache.Start(ctx)

	if err := synchCache.Synch(ctx); err != nil {
		return nil, err
	}
	return synchCache, nil
}

func buildAndRegisterAccessCacheMetrics(registry prometheus.Registerer) (cache.AccessCacheMetrics, error) {
	// if a registry has not been provided, let's proceed without metrics
	if registry == nil {
		return nil, nil
	}

	// build and register the AccessCacheMetrics
	accessCacheMetrics := cache.NewAccessCacheMetrics()
	if err := registry.Register(accessCacheMetrics); err != nil {
		return nil, err
	}
	return accessCacheMetrics, nil
}

// getResyncPeriodFromEnvOrZero retrieves AccessCache's ResyncPeriod from environment variables.
// If the environment variable is not set it returns the zero value.
func getResyncPeriodFromEnvOrZero(ctx context.Context) time.Duration {
	var zero time.Duration
	rps, ok := os.LookupEnv(EnvCacheResyncPeriod)
	if !ok {
		return zero
	}
	rp, err := time.ParseDuration(rps)
	if err != nil {
		getLoggerFromContext(ctx).Warn("can not parse duration from environment variable", "error", err)
		return zero
	}
	return rp
}
