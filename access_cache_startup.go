package main

import (
	"context"

	"github.com/konflux-ci/namespace-lister/internal/log"
	"github.com/konflux-ci/namespace-lister/internal/resourcecache"
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
	acm, err := resourcecache.BuildAndRegisterAccessCacheMetrics(registry)
	if err != nil {
		return nil, err
	}

	aur := &CRAuthRetriever{resourceCache}
	sae := rbac.NewSubjectAccessEvaluator(aur, aur, aur, aur, "")
	synchCache := cache.NewSynchronizedAccessCache(
		sae,
		resourceCache, cache.CacheSynchronizerOptions{
			Logger:       log.GetLoggerFromContext(ctx),
			ResyncPeriod: resourcecache.GetResyncPeriodFromEnvOrZero(ctx),
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

