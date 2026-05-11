package resourcecache

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// BuildAndStart builds and starts a resource Cache.
func BuildAndStart(ctx context.Context, cfg *Config) (cache.Cache, error) {
	c, err := build(cfg)
	if err != nil {
		return nil, err
	}

	if err := start(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func start(ctx context.Context, c cache.Cache) error {
	// get informers
	for _, o := range cachedObjects {
		_, err := c.GetInformer(ctx, o)
		if err != nil {
			return fmt.Errorf("error starting cache: getting informer for %s: %w", o.GetObjectKind().GroupVersionKind().String(), err)
		}
	}

	// start cache
	go func() {
		if err := c.Start(ctx); err != nil {
			panic(err)
		}
	}()

	// wait for cache sync
	if !c.WaitForCacheSync(ctx) {
		return errors.New("error starting the cache")
	}

	return nil
}
