package cache

import (
	"context"
	"encoding/json"
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

const (
	VirtualLabelAnnotationDomainKey = "virtual.konflux-ci.dev/"

	VirtualLabelKeyAccess     = VirtualLabelAnnotationDomainKey + "access"
	VirtualLabelKeyVisibility = VirtualLabelAnnotationDomainKey + "visibility"

	VirtualLabelValueVisibilityAuthenticated = "authenticated"
	VirtualLabelValueVisibilityPrivate       = "private"

	VirtualAnnotationKeySubjectName      = VirtualLabelAnnotationDomainKey + "subject-name"
	VirtualAnnotationKeySubjectNamespace = VirtualLabelAnnotationDomainKey + "subject-namespace"
)

var ErrSynchAlreadyRunning error = errors.New("Synch operation already running")

func isSynchAlreadyRunningErr(err error) bool {
	return err != nil && errors.Is(err, ErrSynchAlreadyRunning)
}

var _ AccessCache = &SynchronizedAccessCache{}

// SynchronizedAccessCache wraps an AccessCache adding logic for synchronizing its data.
type SynchronizedAccessCache struct {
	AccessCache

	requested     chan struct{}
	synchronizing atomic.Bool
	once          sync.Once

	subjectLocator  rbac.SubjectLocator
	namespaceLister client.Reader

	logger           *slog.Logger
	syncErrorHandler func(context.Context, error, *SynchronizedAccessCache)
	resyncPeriod     time.Duration
	synchTimeout     time.Duration

	metrics AccessCacheMetrics
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
		requested:   make(chan struct{}, 1),

		subjectLocator:  subjectLocator,
		namespaceLister: namespaceLister,
	})
}

// Synch recalculates the data to be stored in the cache and applies
func (s *SynchronizedAccessCache) Synch(ctx context.Context) error {
	if !s.synchronizing.CompareAndSwap(false, true) {
		// already running a synch operation
		return ErrSynchAlreadyRunning
	}
	defer s.synchronizing.Store(false)

	// add timeout for the synch operation
	sctx, cancel := context.WithTimeout(ctx, s.synchTimeout)
	defer cancel()

	// execute synch operation
	cacheData, err := s.synch(sctx)

	// collect metrics wrt to synch operation result
	s.metrics.CollectSynchMetrics(cacheData, err)

	return err
}

func (s *SynchronizedAccessCache) synch(ctx context.Context) (AccessData, error) {
	s.logger.Debug("start synchronization")
	nn := corev1.NamespaceList{}
	if err := s.namespaceLister.List(ctx, &nn); err != nil {
		return nil, err
	}

	c := AccessData{}

	// get subjects for each namespace
	for _, ns := range nn.Items {
		// interrupt if context elapsed
		if err := ctx.Err(); err != nil {
			s.logger.Warn("cache restocking: could not complete calculate access data process", "error", err)
			return AccessData{}, ctx.Err()
		}

		ar := authorizer.AttributesRecord{
			Verb:            "get",
			Resource:        "namespaces",
			APIGroup:        corev1.GroupName,
			APIVersion:      corev1.SchemeGroupVersion.Version,
			Name:            ns.GetName(),
			Namespace:       ns.GetName(),
			ResourceRequest: true,
		}

		ss, err := s.subjectLocator.AllowedSubjects(ctx, ar)
		if err != nil {
			// do not forward the error as it should be due
			// to cache evicted (cluster)roles
			s.logger.Debug("cache restocking: error caculating allowed subjects", "namespace", ns.GetName(), "error", err)
		}

		// remove duplicates from allowed subjects
		ss = s.removeDuplicateSubjects(ss)

		// enforce visibility label
		s.setVisibilityVirtualLabel(&ns, ss)

		// store in temp cache
		for _, sub := range ss {
			lns := s.withVirtualLabelsAndAnnotationsForAccess(ns, sub)

			c[sub] = append(c[sub], lns)
		}
	}

	// restock the cache
	s.Restock(&c)

	s.logDumpCacheData(ctx, slog.LevelDebug, c)
	return c, nil
}

func (s *SynchronizedAccessCache) setVisibilityVirtualLabel(ns *corev1.Namespace, subs []rbacv1.Subject) {
	// system:authenticated matcher function
	isSystemAuthenticatedGroup := func(sub rbacv1.Subject) bool {
		return sub.APIGroup == rbacv1.GroupName &&
			sub.Kind == rbacv1.GroupKind &&
			sub.Name == "system:authenticated"
	}

	// retrieve Labels
	ll := ns.GetLabels()
	if ll == nil {
		ll = map[string]string{}
	}

	// if namespace is shared with `system:authenticated` group,
	// then the visibility virtual label is set to `authenticated`
	if slices.ContainsFunc(subs, isSystemAuthenticatedGroup) {
		ll[VirtualLabelKeyVisibility] = VirtualLabelValueVisibilityAuthenticated
		return
	}

	// otherwise visibility virtual label is set to `private`
	ll[VirtualLabelKeyVisibility] = VirtualLabelValueVisibilityPrivate
}

func (s *SynchronizedAccessCache) withVirtualLabelsAndAnnotationsForAccess(ns corev1.Namespace, sub rbacv1.Subject) corev1.Namespace {
	// we need to deepcopy otherwise we'll have side effects
	lns := ns.DeepCopy()

	// calculate data
	lkind := strings.ToLower(sub.Kind)

	// add labels
	ll := lns.GetLabels()
	if ll == nil {
		ll = map[string]string{}
	}
	ll[VirtualLabelKeyAccess] = lkind
	lns.Labels = ll

	// add annotations
	aa := lns.GetAnnotations()
	if aa == nil {
		aa = map[string]string{}
	}
	aa[VirtualAnnotationKeySubjectName] = sub.Name
	if sub.Namespace != "" {
		aa[VirtualAnnotationKeySubjectNamespace] = sub.Namespace
	}
	lns.Annotations = aa

	// return copy
	return *lns
}

func (s *SynchronizedAccessCache) logDumpCacheData(ctx context.Context, level slog.Level, c AccessData) {
	if !s.logger.Enabled(ctx, level) {
		return
	}

	// calculate subject-namespace pairs dump
	snp, snt := 0, make(map[string]int, len(c))
	for k, v := range c {
		snp += len(v)
		snt[k.String()] = len(v)
	}

	// log the line
	args := []slog.Attr{
		slog.Int("subjects", len(c)),
		slog.Int("subject-namespace pairs", snp),
	}
	if jsnt, err := json.Marshal(snt); err == nil {
		args = append(args, slog.String("dump-json", string(jsnt)))
	} else {
		args = append(args, slog.Any("dump", snt))
	}
	s.logger.LogAttrs(ctx, level, "cache restocked", args...)
}

func (s *SynchronizedAccessCache) removeDuplicateSubjects(ss []rbacv1.Subject) []rbacv1.Subject {
	// sort the list of subjects
	slices.SortFunc(ss, func(a, b rbacv1.Subject) int {
		if c := strings.Compare(a.APIGroup, b.APIGroup); c != 0 {
			return c
		}
		if c := strings.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		if c := strings.Compare(a.Namespace, b.Namespace); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
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
func (s *SynchronizedAccessCache) Request(event Event) bool {
	// request to synchronize the cache
	queued := s.request()

	// collect metrics on event and request
	s.metrics.CollectRequestMetrics(event, queued)

	return queued
}

func (s *SynchronizedAccessCache) request() bool {
	select {
	case s.requested <- struct{}{}:
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
					queued := s.Request(timeTriggeredEvent)
					s.logger.Debug("time-based cache synchronization request made", "queued", queued)
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

				case <-s.requested:
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
