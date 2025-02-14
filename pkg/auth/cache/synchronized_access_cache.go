package cache

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
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

var _ AccessCache = &SynchronizedAccessCache{}

// SynchronizedAccessCache wraps an AccessCache adding logic for synchronizing its data.
type SynchronizedAccessCache struct {
	AccessCache

	request       chan struct{}
	synchronizing atomic.Bool
	once          sync.Once

	subjectLocator  rbac.SubjectLocator
	namespaceLister client.Reader

	logger           *slog.Logger
	syncErrorHandler func(context.Context, error, *SynchronizedAccessCache)
	resyncPeriod     time.Duration
}

// NewSynchronizedAccessCache builds a SynchronizedAccessCache.
// The cache is meant to be started via the `Start` method.
func NewSynchronizedAccessCache(
	subjectLocator rbac.SubjectLocator,
	namespaceLister client.Reader,
	opts CacheSynchronizerOptions,
) *SynchronizedAccessCache {
	return opts.Apply(&SynchronizedAccessCache{
		AccessCache: NewAtomicListRestockAccessCache(),
		request:     make(chan struct{}, 1),

		subjectLocator:  subjectLocator,
		namespaceLister: namespaceLister,
	})
}

// Synch recalculates the data to be stored in the cache and applies
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

		// remove duplicates from allowed subjects
		ss = s.removeDuplicateSubjects(ss)

		// store in temp cache
		for _, s := range ss {
			c[s] = append(c[s], ns)
		}
	}

	// restock the cache
	s.AccessCache.Restock(&c)

	s.logger.Debug("cache restocked")
	return nil
}

func (s *SynchronizedAccessCache) removeDuplicateSubjects(ss []rbacv1.Subject) []rbacv1.Subject {
	// sort the list of subjects
	slices.SortFunc(ss, func(a, b rbacv1.Subject) int {
		switch {
		case a.APIGroup != b.APIGroup:
			return strings.Compare(a.APIGroup, b.APIGroup)
		case a.Kind != b.Kind:
			return strings.Compare(a.Kind, b.Kind)
		case a.Namespace != b.Namespace:
			return strings.Compare(a.Namespace, b.Namespace)
		case a.Name != b.Name:
			return strings.Compare(a.Name, b.Name)
		default:
			return 0
		}
	})

	// remove duplicates
	ss = slices.CompactFunc(ss, func(a, b rbacv1.Subject) bool {
		return a.APIGroup == b.APIGroup &&
			a.Kind == b.Kind &&
			a.Namespace == b.Namespace &&
			a.Name == b.Name
	})

	// reduce slice capacity to its length
	return slices.Clip(ss)
}

// Request allows events to request to run a Synch operation.
// Only one request is kept in memory. If a Synch operation has already been
// requested - but still not processed, and a new request comes it will be discarded.
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

// Start runs two goroutines to keep the cache up-to-date.
//
// The former will enqueue requests to synch the cache by intervals of `resyncPeriod`.
// The latter waits for requests to synch the cache and runs the Synch operation.
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
