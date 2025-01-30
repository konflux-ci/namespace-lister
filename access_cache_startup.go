package main

import (
	"context"
	"os"
	"time"

	authcache "github.com/konflux-ci/namespace-lister/pkg/auth/cache"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func buildAndStartAccessCache(ctx context.Context, resourceCache crcache.Cache) (*authcache.SynchronizedAccessCache, error) {
	aur := &CRAuthRetriever{resourceCache, ctx, getLoggerFromContext(ctx)}
	sae := rbac.NewSubjectAccessEvaluator(aur, aur, aur, aur, "")
	synchCache := authcache.NewSynchronizedAccessCache(
		sae,
		resourceCache, authcache.CacheSynchronizerOptions{
			Logger:       getLoggerFromContext(ctx),
			ResyncPeriod: getResyncPeriodFromEnvOrZero(ctx),
		},
	)

	// register event handlers on resource cache
	oo := []client.Object{
		&corev1.Namespace{},
		&rbacv1.RoleBinding{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&rbacv1.Role{},
	}
	for _, o := range oo {
		i, err := resourceCache.GetInformer(ctx, o)
		if err != nil {
			return nil, err
		}

		if _, err := i.AddEventHandler(
			toolscache.ResourceEventHandlerFuncs{
				AddFunc:    func(obj interface{}) { synchCache.Request() },
				UpdateFunc: func(oldObj, newObj interface{}) { synchCache.Request() },
				DeleteFunc: func(obj interface{}) { synchCache.Request() },
			}); err != nil {
			return nil, err
		}
	}
	synchCache.Start(ctx)

	if err := synchCache.Synch(ctx); err != nil {
		return nil, err
	}
	return synchCache, nil
}

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
