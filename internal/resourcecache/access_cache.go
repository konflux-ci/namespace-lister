package resourcecache

import (
	"context"
	"os"
	"time"

	"github.com/konflux-ci/namespace-lister/internal/constants"
	"github.com/konflux-ci/namespace-lister/internal/log"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/prometheus/client_golang/prometheus"
)

func BuildAndRegisterAccessCacheMetrics(registry prometheus.Registerer) (cache.AccessCacheMetrics, error) {
	if registry == nil {
		return nil, nil
	}

	accessCacheMetrics := cache.NewAccessCacheMetrics()
	if err := registry.Register(accessCacheMetrics); err != nil {
		return nil, err
	}
	return accessCacheMetrics, nil
}

// GetResyncPeriodFromEnvOrZero retrieves AccessCache's ResyncPeriod from environment variables.
// If the environment variable is not set it returns the zero value.
func GetResyncPeriodFromEnvOrZero(ctx context.Context) time.Duration {
	var zero time.Duration
	rps, ok := os.LookupEnv(constants.EnvCacheResyncPeriod)
	if !ok {
		return zero
	}
	rp, err := time.ParseDuration(rps)
	if err != nil {
		log.GetLoggerFromContext(ctx).Warn("can not parse duration from environment variable", "error", err)
		return zero
	}
	return rp
}
