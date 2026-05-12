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

// GetValidResyncPeriodFromEnvOrZero retrieves AccessCache's ResyncPeriod from environment variables.
// If the environment variable is not set or the value is not a valid non-negative duration, it returns the zero value.
func GetValidResyncPeriodFromEnvOrZero(ctx context.Context) time.Duration {
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
	if rp < 0 {
		log.GetLoggerFromContext(ctx).Warn("negative resync period, using zero", "value", rps)
		return zero
	}
	return rp
}
