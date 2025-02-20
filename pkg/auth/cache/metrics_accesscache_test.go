package cache_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache/mocks"
)

const (
	resourcesRequestsMetricFullname     = "namespace_lister_accesscache_resource_requests_total"
	timeRequestsMetricFullname          = "namespace_lister_accesscache_time_requests_total"
	syncMetricFullname                  = "namespace_lister_accesscache_synch_op_total"
	subjectNamespacePairsMetricFullname = "namespace_lister_accesscache_subject_namespace_pairs"
	subjectsMetricFullname              = "namespace_lister_accesscache_subjects"

	metadataSynchOp = `# HELP namespace_lister_accesscache_synch_op_total synchronization operations
# TYPE namespace_lister_accesscache_synch_op_total counter`
	metadataSubjects = `# HELP namespace_lister_accesscache_subjects Subjects in the cache
# TYPE namespace_lister_accesscache_subjects gauge`
	metadataSubjectNamespacePairs = `# HELP namespace_lister_accesscache_subject_namespace_pairs (Subject, Namespace) pairs in the cache
# TYPE namespace_lister_accesscache_subject_namespace_pairs gauge`

	entriesSynchOpCompleted = `
namespace_lister_accesscache_synch_op_total{error="",status="completed"} 1
`
	entriesSynchOpFailed = `
namespace_lister_accesscache_synch_op_total{error="err",status="failed"} 1
`
	entriesSubjectsFmt = `
namespace_lister_accesscache_subjects %d
`
	entriesSubjectNamespacePairsFmt = `
namespace_lister_accesscache_subject_namespace_pairs %d
`
)

var _ = Describe("MetricsAccessCache/FailedSynch", func() {
	var metrics cache.AccessCacheMetrics

	BeforeEach(func() {
		metrics = cache.NewAccessCacheMetrics()
	})

	It("collects failed synch metrics for empty access data", func(ctx context.Context) {
		// when
		metrics.CollectSynchMetrics(cache.AccessData{}, errors.New("err"))

		// then
		Expect(
			testutil.CollectAndCompare(metrics, strings.NewReader(metadataSynchOp+entriesSynchOpFailed), syncMetricFullname)).
			To(Succeed())
	})
})

func metricsFmt(metadata, entriesFmt string, entriesArgs ...interface{}) *strings.Reader {
	entries := fmt.Sprintf(entriesFmt, entriesArgs...)
	all := fmt.Sprintf("%s\n%s\n", metadata, entries)
	return strings.NewReader(all)
}

var _ = DescribeTable("MetricsAccessCache/SuccessfulSynch", func(data cache.AccessData, err error, subs, subNsPairs int) {
	// given
	metrics := cache.NewAccessCacheMetrics()

	// when
	metrics.CollectSynchMetrics(data, err)

	// then
	Expect(
		testutil.CollectAndCompare(metrics, metricsFmt(metadataSynchOp, entriesSynchOpCompleted), syncMetricFullname)).
		To(Succeed())
	Expect(
		testutil.CollectAndCompare(
			metrics,
			metricsFmt(metadataSubjectNamespacePairs, entriesSubjectNamespacePairsFmt, subNsPairs),
			subjectNamespacePairsMetricFullname)).
		To(Succeed())
	Expect(
		testutil.CollectAndCompare(
			metrics,
			metricsFmt(metadataSubjects, entriesSubjectsFmt, subs),
			subjectsMetricFullname)).
		To(Succeed())
},
	Entry("nil data", nil, nil, 0, 0),
	Entry("empty data", cache.AccessData{}, nil, 0, 0),
	Entry("1 subject", cache.AccessData{
		rbacv1.Subject{}: []corev1.Namespace{},
	}, nil, 1, 0),
	Entry("2 subjects - 10 Namespaces", cache.AccessData{
		rbacv1.Subject{Name: "1"}: []corev1.Namespace{{}, {}, {}, {}, {}},
		rbacv1.Subject{Name: "2"}: []corev1.Namespace{{}, {}, {}, {}, {}},
	}, nil, 2, 10),
)

var _ = Describe("MetricsAccessCache/TimeRequests", func() {
	var metrics cache.AccessCacheMetrics
	var ctrl *gomock.Controller

	BeforeEach(func() {
		metrics = cache.NewAccessCacheMetrics()
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("collects time-triggered request metrics", func(ctx context.Context) {
		metadata := `# HELP namespace_lister_accesscache_time_requests_total synchronization requests triggered when resync period elapses
# TYPE namespace_lister_accesscache_time_requests_total counter`
		entries := `
namespace_lister_accesscache_time_requests_total{status="queued"} 1
`
		nl := mocks.NewMockClientReader(ctrl)
		nl.EXPECT().List(ctx, gomock.Any()).Return(nil).Times(1)

		// when
		cache.NewSynchronizedAccessCache(nil, nl, cache.CacheSynchronizerOptions{
			ResyncPeriod: 100 * time.Millisecond,
			Metrics:      metrics,
		}).Start(ctx)

		time.Sleep(150 * time.Millisecond)

		// then
		Expect(
			testutil.CollectAndCompare(metrics, strings.NewReader(metadata+entries), timeRequestsMetricFullname)).
			To(Succeed())
	})
})

var _ = DescribeTableSubtree("MetricsAccessCache/ResourceRequests",
	func(actualEvent cache.Event, expectedApiVersion, expectedKind, expectedName, expectedNamespace string) {
		metadata := `# HELP namespace_lister_accesscache_resource_requests_total synchronization requests triggered by events on watched resources
# TYPE namespace_lister_accesscache_resource_requests_total counter`
		entriesFmt := `namespace_lister_accesscache_resource_requests_total{event_type="%s",resource_apiversion="%s",resource_kind="%s",resource_name="%s",resource_namespace="%s",status="%s"} 1`

		var metrics cache.AccessCacheMetrics

		BeforeEach(func() {
			metrics = cache.NewAccessCacheMetrics()
		})

		It("collects request metrics for queued request", func() {
			// when
			metrics.CollectRequestMetrics(actualEvent, true)

			// then
			Expect(
				testutil.CollectAndCompare(metrics,
					metricsFmt(metadata,
						entriesFmt, actualEvent.Type, expectedApiVersion, expectedKind, expectedName, expectedNamespace, cache.StatusQueuedLabel),
					resourcesRequestsMetricFullname)).
				To(Succeed())
		})

		It("collects request metrics for skipped request", func() {
			// when
			metrics.CollectRequestMetrics(actualEvent, false)

			// then
			Expect(
				testutil.CollectAndCompare(metrics,
					metricsFmt(metadata,
						entriesFmt, actualEvent.Type, expectedApiVersion, expectedKind, expectedName, expectedNamespace, cache.StatusSkippedLabel),
					resourcesRequestsMetricFullname)).
				To(Succeed())
		})
	},
	Entry("nil object event", cache.Event{Type: cache.ResourceAddedEventType}, "", "", "", ""),
	Entry("namespace add event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceAddedEventType,
	}, "v1", "Namespace", "myns", ""),
	Entry("namespace update event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceUpdatedEventType,
	}, "v1", "Namespace", "myns", ""),
	Entry("namespace deleted event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceDeletedEventType,
	}, "v1", "Namespace", "myns", ""),
	Entry("role add event", cache.Event{
		Object: &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "myrole", Namespace: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "RoleBinding", APIVersion: rbacv1.SchemeGroupVersion.String()},
		},
		Type: cache.ResourceAddedEventType,
	}, rbacv1.SchemeGroupVersion.String(), "RoleBinding", "myrole", "myns"),
	Entry("role update event", cache.Event{
		Object: &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "myrole", Namespace: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "RoleBinding", APIVersion: rbacv1.SchemeGroupVersion.String()},
		},
		Type: cache.ResourceUpdatedEventType,
	}, rbacv1.SchemeGroupVersion.String(), "RoleBinding", "myrole", "myns"),
	Entry("role deleted event", cache.Event{
		Object: &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "myrole", Namespace: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "RoleBinding", APIVersion: rbacv1.SchemeGroupVersion.String()},
		},
		Type: cache.ResourceDeletedEventType,
	}, rbacv1.SchemeGroupVersion.String(), "RoleBinding", "myrole", "myns"),
)
