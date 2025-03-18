package main

import (
	"errors"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
)

const CacheNamespaceLabelSelectorEnv string = "CACHE_NAMESPACE_LABELSELECTOR"

var ResourceCacheConfigErr error = errors.New("error building resource cache configuration")

func NewResourceCacheConfigFromEnv(cfg *rest.Config) (*cacheConfig, error) {
	// get namespaces labelSelector
	cacheCfg := &cacheConfig{restConfig: cfg}
	if err := getNamespacesLabelSelectors(cacheCfg); err != nil {
		return nil, err
	}

	return cacheCfg, nil
}

func getNamespacesLabelSelectors(cfg *cacheConfig) error {
	ls, err := labels.Parse(os.Getenv(CacheNamespaceLabelSelectorEnv))
	if err != nil {
		return fmt.Errorf("%w for namespaces: %w", ResourceCacheConfigErr, err)
	}

	cfg.namespacesLabelSector = ls
	return nil
}
