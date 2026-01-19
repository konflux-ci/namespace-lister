package cache_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache/mocks"
	"github.com/konflux-ci/namespace-lister/pkg/metricsutil"
)

var _ = Describe("MetricsAccessCache/FailedSynch", func() {
	var metrics cache.AccessCacheMetrics

	BeforeEach(func() {
		metrics = cache.NewAccessCacheMetrics()
	})

	It("collects failed synch metrics for empty access data", func(ctx context.Context) {
		// when
		metrics.CollectSynchMetrics(.0, cache.AccessData{}, errors.New("err"))

		// then
		vec, err := metricsutil.GetVector(metrics, metricsutil.SyncMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
		Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue("failed")))
		Expect(vec[0].Metric["error"]).To(Equal(model.LabelValue("err")))
	})
})

var _ = DescribeTable("MetricsAccessCache/SuccessfulSynch", func(data cache.AccessData, err error, subNsPairs int) {
	// given
	metrics := cache.NewAccessCacheMetrics()

	// when
	metrics.CollectSynchMetrics(.0, data, err)

	// then
	// check that the synch operation has been executed
	{
		vec, err := metricsutil.GetVector(metrics, metricsutil.SyncMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
		Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue("completed")))
	}
	// check we have registered the correct amount of subjects
	{
		vec, err := metricsutil.GetVector(metrics, metricsutil.SubjectsMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(len(data))))
	}
	// check we have registered the correct amount of (subject,namespace) pairs
	if subNsPairs > 0 {
		vec, err := metricsutil.GetVector(metrics, metricsutil.SubjectNamespacePairsMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(subNsPairs)))
	}
},
	Entry("nil data", nil, nil, 0),
	Entry("empty data", cache.AccessData{}, nil, 0),
	Entry("1 subject", cache.AccessData{
		rbacv1.Subject{}: []corev1.Namespace{},
	}, nil, 0),
	Entry("2 subjects - 10 Namespaces", cache.AccessData{
		rbacv1.Subject{Name: "1"}: []corev1.Namespace{{}, {}, {}, {}, {}},
		rbacv1.Subject{Name: "2"}: []corev1.Namespace{{}, {}, {}, {}, {}},
	}, nil, 10),
)

var _ = DescribeTable("MetricsAccessCache/UnsuccessfulSynch", func(data cache.AccessData, err error) {
	// given
	metrics := cache.NewAccessCacheMetrics()

	// when
	metrics.CollectSynchMetrics(.0, data, err)

	// then
	// check that the synch operation has been executed
	{
		vec, err := metricsutil.GetVector(metrics, metricsutil.SyncMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
		Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue("failed")))
	}
	// check we have registered the correct amount of subjects
	{
		vec, err := metricsutil.GetVector(metrics, metricsutil.SubjectsMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(0)))
	}
	// check we have registered the correct amount of (subject,namespace) pairs
	{
		vec, err := metricsutil.GetVector(metrics, metricsutil.SubjectNamespacePairsMetricFullname)
		Expect(err).To(MatchError("metric family not found"))
		Expect(vec).To(BeEmpty())
	}
},
	Entry("unexpected data in cache", cache.AccessData{
		rbacv1.Subject{Name: "1"}: []corev1.Namespace{{}, {}, {}, {}, {}},
		rbacv1.Subject{Name: "2"}: []corev1.Namespace{{}, {}, {}, {}, {}},
	}, errors.New("error")),
	Entry("generic error", cache.AccessData{}, errors.New("error")),
	Entry("context deadline exceeded error", cache.AccessData{}, errors.New("context deadline exceeded")),
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
		nl := mocks.NewMockClientReader(ctrl)
		nl.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// when
		cache.NewSynchronizedAccessCache(nil, nl, cache.CacheSynchronizerOptions{
			ResyncPeriod: 100 * time.Millisecond,
			Metrics:      metrics,
		}).Start(ctx)

		time.Sleep(150 * time.Millisecond)

		// then
		vec, err := metricsutil.GetVector(metrics, metricsutil.TimeRequestsMetricFullname)
		Expect(err).NotTo(HaveOccurred())
		Expect(vec).To(HaveLen(1))
		Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
		Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue("queued")))
	})
})

var _ = DescribeTableSubtree("MetricsAccessCache/ResourceRequests",
	func(actualEvent cache.Event) {
		var metrics cache.AccessCacheMetrics

		BeforeEach(func() {
			metrics = cache.NewAccessCacheMetrics()
		})

		It("collects request metrics for queued request", func() {
			// when
			metrics.CollectRequestMetrics(actualEvent, true)

			// then
			vec, err := metricsutil.GetVector(metrics, metricsutil.ResourcesRequestsMetricFullname)
			Expect(err).NotTo(HaveOccurred())
			Expect(vec).To(HaveLen(1))
			Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
			Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue(cache.StatusQueuedLabel)))
		})

		It("collects request metrics for skipped request", func() {
			// when
			metrics.CollectRequestMetrics(actualEvent, false)

			// then
			vec, err := metricsutil.GetVector(metrics, metricsutil.ResourcesRequestsMetricFullname)
			Expect(err).NotTo(HaveOccurred())
			Expect(vec).To(HaveLen(1))
			Expect(vec[0].Value).To(Equal(model.SampleValue(1)))
			Expect(vec[0].Metric["status"]).To(Equal(model.LabelValue(cache.StatusSkippedLabel)))
		})
	},
	Entry("nil object event", cache.Event{Type: cache.ResourceAddedEventType}),
	Entry("namespace add event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceAddedEventType,
	}),
	Entry("namespace update event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceUpdatedEventType,
	}),
	Entry("namespace deleted event", cache.Event{
		Object: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "myns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		},
		Type: cache.ResourceDeletedEventType,
	}),
)
