package cache_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache/mocks"
	"github.com/konflux-ci/namespace-lister/pkg/metricsutil"
)

var (
	systemAuthenticatedGroupSubject = rbacv1.Subject{
		Kind:     rbacv1.GroupKind,
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Name:     "system:authenticated",
	}
)

var _ = Describe("VisibilityVirtualLabel", func() {
	var ctrl *gomock.Controller
	var subjectLocator *mocks.MockSubjectLocator

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		subjectLocator = mocks.NewMockSubjectLocator(ctrl)
	})

	When("the namespace is not shared with system:authenticated group", func() {
		It("sets the visibility virtual label to private", func(ctx context.Context) {
			namespaceLister := mocks.NewMockClientReader(ctrl)
			namespaceLister.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
					(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
					return nil
				}).
				Times(1)
			subjectLocator.EXPECT().
				AllowedSubjects(gomock.Any(), gomock.Any()).
				Return([]rbacv1.Subject{userSubject}, nil).
				Times(1)

			nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

			Expect(nsc.Synch(ctx)).To(Succeed())
			Expect(nsc.AccessCache.List(userSubject)).To(ConsistOf(expectedNamespacesUserAccessPrivate))
		})
	})

	When("the namespace is shared with system:authenticated group", func() {
		It("sets the visibility virtual label to authenticated", func(ctx context.Context) {
			namespaceLister := mocks.NewMockClientReader(ctrl)
			namespaceLister.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
					(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
					return nil
				}).
				Times(1)
			subjectLocator.EXPECT().
				AllowedSubjects(gomock.Any(), gomock.Any()).
				Return([]rbacv1.Subject{userSubject, systemAuthenticatedGroupSubject}, nil).
				Times(1)

			nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

			Expect(nsc.Synch(ctx)).To(Succeed())
			Expect(nsc.AccessCache.List(userSubject)).To(ConsistOf(expectedNamespacesUserAccessAuthenticated))
		})
	})
})

var _ = Describe("SynchDuration", func() {
	var ctrl *gomock.Controller
	var subjectLocator *mocks.MockSubjectLocator
	var metricsCache cache.AccessCacheMetrics

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		subjectLocator = mocks.NewMockSubjectLocator(ctrl)
		metricsCache = cache.NewAccessCacheMetrics()
	})

	It("sets a positive duration", func(ctx context.Context) {
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				time.Sleep(100 * time.Millisecond)
				(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)
		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			Return([]rbacv1.Subject{userSubject, systemAuthenticatedGroupSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(
			subjectLocator,
			namespaceLister,
			cache.CacheSynchronizerOptions{Metrics: metricsCache},
		)

		Expect(nsc.Synch(ctx)).To(Succeed())

		// Check histogram count for completed status
		count, err := metricsutil.GetHistogramCount(metricsCache, metricsutil.SynchDurationFullname, map[string]string{"status": "completed"})
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1.0))

		// Check histogram sum is positive (duration was observed)
		sum, err := metricsutil.GetHistogramSum(metricsCache, metricsutil.SynchDurationFullname, map[string]string{"status": "completed"})
		Expect(err).NotTo(HaveOccurred())
		// The sum should be at least 100ms (the sleep duration) in milliseconds
		Expect(sum).To(BeNumerically(">=", 100.0))
	})
})
