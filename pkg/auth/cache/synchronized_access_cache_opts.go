package cache

import (
	"cmp"
	"context"
	"log/slog"
	"time"
)

type CacheSynchronizerOptions struct {
	Logger           *slog.Logger
	ResyncPeriod     time.Duration
	SyncErrorHandler func(context.Context, error, *SynchronizedAccessCache)
}

var defaultCacheSynchronizerOptions = CacheSynchronizerOptions{
	Logger:       slog.Default(),
	ResyncPeriod: 10 * time.Minute,
	SyncErrorHandler: func(ctx context.Context, err error, s *SynchronizedAccessCache) {
		s.logger.Error("error synchronizing cache", "error", err)
	},
}

func (opts *CacheSynchronizerOptions) Apply(s *SynchronizedAccessCache) *SynchronizedAccessCache {
	s.resyncPeriod = cmp.Or(opts.ResyncPeriod, defaultCacheSynchronizerOptions.ResyncPeriod)

	s.syncErrorHandler = opts.SyncErrorHandler
	if s.syncErrorHandler == nil {
		s.syncErrorHandler = defaultCacheSynchronizerOptions.SyncErrorHandler
	}

	s.logger = cmp.Or(opts.Logger, defaultCacheSynchronizerOptions.Logger)
	return s
}
