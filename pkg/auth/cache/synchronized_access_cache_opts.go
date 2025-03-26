package cache

import (
	"cmp"
	"context"
	"log/slog"
	"time"
)

// CacheSynchronizerOptions allows tune SynchronizedAccessCache's behavior
type CacheSynchronizerOptions struct {
	Logger           *slog.Logger
	ResyncPeriod     time.Duration
	SynchTimeout     time.Duration
	SyncErrorHandler func(context.Context, error, *SynchronizedAccessCache)
	Metrics          AccessCacheMetrics
}

var defaultCacheSynchronizerOptions = CacheSynchronizerOptions{
	Logger:       slog.Default(),
	ResyncPeriod: 10 * time.Minute,
	SynchTimeout: 1 * time.Minute,
	SyncErrorHandler: func(ctx context.Context, err error, s *SynchronizedAccessCache) {
		s.logger.Error("error synchronizing cache", "error", err)
	},
	Metrics: &NoOpAccessCacheMetrics{},
}

// Apply applies the provided options to the SynchronizedAccessCache.
// It enforces defaults where values were not provided.
func (opts *CacheSynchronizerOptions) Apply(s *SynchronizedAccessCache) *SynchronizedAccessCache {
	// add resync period
	s.resyncPeriod = cmp.Or(opts.ResyncPeriod, defaultCacheSynchronizerOptions.ResyncPeriod)

	// add synch Timeout
	s.synchTimeout = cmp.Or(opts.SynchTimeout, max(defaultCacheSynchronizerOptions.SynchTimeout, s.resyncPeriod-time.Minute))

	// add synch error handler
	s.syncErrorHandler = opts.SyncErrorHandler
	if s.syncErrorHandler == nil {
		s.syncErrorHandler = defaultCacheSynchronizerOptions.SyncErrorHandler
	}

	// add logger
	s.logger = cmp.Or(opts.Logger, defaultCacheSynchronizerOptions.Logger)

	// add metricsRegistry
	s.metrics = cmp.Or(opts.Metrics, defaultCacheSynchronizerOptions.Metrics)

	return s
}
