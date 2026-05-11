package resourcecache

import (
	"errors"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
)

const CacheNamespaceLabelSelectorEnv string = "CACHE_NAMESPACE_LABELSELECTOR"

var ErrResourceCacheConfig error = errors.New("error building resource cache configuration")

type Config struct {
	RestConfig              *rest.Config
	NamespacesLabelSelector labels.Selector
}

func NewConfigFromEnv(cfg *rest.Config) (*Config, error) {
	// get namespaces labelSelector
	cacheCfg := &Config{RestConfig: cfg}
	if err := getNamespacesLabelSelectorsFromEnv(cacheCfg); err != nil {
		return nil, err
	}

	return cacheCfg, nil
}

func getNamespacesLabelSelectorsFromEnv(cfg *Config) error {
	ls, err := labels.Parse(os.Getenv(CacheNamespaceLabelSelectorEnv))
	if err != nil {
		return fmt.Errorf("%w for namespaces: %w", ErrResourceCacheConfig, err)
	}

	cfg.NamespacesLabelSelector = ls
	return nil
}
