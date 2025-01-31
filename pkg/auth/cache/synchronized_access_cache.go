package cache

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SynchAlreadyRunningErr error = errors.New("Synch operation already running")

func isSynchAlreadyRunningErr(err error) bool {
	return err != nil && errors.Is(err, SynchAlreadyRunningErr)
}

// applies changes to cache async
type SynchronizedAccessCache struct {
	*AccessCache
	request       chan struct{}
	synchronizing atomic.Bool
	once          sync.Once

	subjectLocator  rbac.SubjectLocator
	namespaceLister client.Reader

	logger           *slog.Logger
	syncErrorHandler func(context.Context, error, *SynchronizedAccessCache)
	resyncPeriod     time.Duration
}

func NewSynchronizedAccessCache(
	subjectLocator rbac.SubjectLocator,
	namespaceLister client.Reader,
	opts CacheSynchronizerOptions,
) *SynchronizedAccessCache {
	return opts.Apply(&SynchronizedAccessCache{
		AccessCache: NewAccessCache(),
		request:     make(chan struct{}, 1),

		subjectLocator:  subjectLocator,
		namespaceLister: namespaceLister,
	})
}

func (s *SynchronizedAccessCache) Synch(ctx context.Context) error {
	if !s.synchronizing.CompareAndSwap(false, true) {
		// already running a synch operation
		return SynchAlreadyRunningErr
	}
	defer s.synchronizing.Store(false)

	s.logger.Debug("start synchronization")
	nn := corev1.NamespaceList{}
	if err := s.namespaceLister.List(ctx, &nn); err != nil {
		return err
	}

	c := map[rbacv1.Subject][]corev1.Namespace{}

	// get subjects for each namespace
	for _, ns := range nn.Items {
		ar := authorizer.AttributesRecord{
			Verb:            "get",
			Resource:        "namespaces",
			APIGroup:        corev1.GroupName,
			APIVersion:      corev1.SchemeGroupVersion.Version,
			Name:            ns.GetName(),
			Namespace:       ns.GetName(),
			ResourceRequest: true,
		}

		ss, err := s.subjectLocator.AllowedSubjects(ar)
		if err != nil {
			// do not forward the error as it should be due
			// to cache evicted (cluster)roles
			s.logger.Debug("cache restocking: error caculating allowed subjects", "error", err)
		}

		for _, s := range ss {
			c[s] = append(c[s], ns)
		}
	}

	// restock the cache
	s.AccessCache.Restock(&c)

	s.logger.Debug("cache restocked")
	return nil
}

func (s *SynchronizedAccessCache) Request() bool {
	select {
	case s.request <- struct{}{}:
		// requested correctly
		return true
	default:
		// a request is already present
		return false
	}
}

func (s *SynchronizedAccessCache) Start(ctx context.Context) {
	s.once.Do(func() {
		// run time based resync
		go func() {
			for {
				select {
				case <-ctx.Done():
					// termination
					s.logger.Info("terminating time-based cache synchronization: context done")
					return
				case <-time.After(s.resyncPeriod):
					ok := s.Request()
					s.logger.Debug("time-based cache synchronization request made", "queued", ok)
				}
			}
		}()

		// schedule requested synch
		go func() {
			for {
				select {
				case <-ctx.Done():
					// termination
					s.logger.Info("terminating cache synchronization goroutine: context done")
					return

				case <-s.request:
					// a new request is present
					s.logger.Debug("start requested cache synchronization")
					if err := s.Synch(ctx); isSynchAlreadyRunningErr(err) {
						s.syncErrorHandler(ctx, err, s)
					}
				}
			}
		}()
	})
}
