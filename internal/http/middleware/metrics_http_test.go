package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konflux-ci/namespace-lister/internal/http/middleware"
)

var _ = Describe("AddMetricsMiddleware", func() {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	It("returns the original handler when registerer is nil", func() {
		h := middleware.AddMetricsMiddleware(nil, dummyHandler)
		Expect(fmt.Sprintf("%p", h)).To(Equal(fmt.Sprintf("%p", dummyHandler)))
	})

	It("registers all metrics with correct names", func() {
		reg := prometheus.NewRegistry()
		h := middleware.AddMetricsMiddleware(reg, dummyHandler)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rec, req)

		families, err := reg.Gather()
		Expect(err).NotTo(HaveOccurred())

		names := metricFamilyNames(families)
		Expect(names).To(ConsistOf(
			"namespace_lister_api_latency",
			"namespace_lister_api_counter",
			"namespace_lister_api_response_size",
			"namespace_lister_api_requests_in_flight",
		))
	})

	It("panics on duplicate registration", func() {
		reg := prometheus.NewRegistry()
		middleware.AddMetricsMiddleware(reg, dummyHandler)
		Expect(func() { middleware.AddMetricsMiddleware(reg, dummyHandler) }).To(Panic())
	})

	Describe("request latency histogram", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			reg := prometheus.NewRegistry()
			h := middleware.AddMetricsMiddleware(reg, dummyHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			h.ServeHTTP(rec, req)

			mf = findFamily(reg, "namespace_lister_api_latency")
		})

		It("has correct help text and type", func() {
			Expect(mf.GetHelp()).To(Equal("Latency of requests"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_HISTOGRAM))
		})

		It("has expected bucket boundaries", func() {
			boundaries := bucketBoundaries(mf.GetMetric()[0].GetHistogram().GetBucket())
			Expect(boundaries).To(Equal([]float64{
				1e-9, 2.5e-9, 5e-9,
				1e-8, 2.5e-8, 5e-8,
				1e-7, 2.5e-7, 5e-7,
				1e-6, 2.5e-6, 5e-6,
				1e-5, 2.5e-5, 5e-5,
				1e-4, 2.5e-4, 5e-4,
				0.001, 0.0025, 0.005,
				0.01, 0.025, 0.05,
				0.1, 0.25, 0.5,
				1.0, 2.0, 5.0,
				10.0, 20.0, 30.0, 60.0,
			}))
		})

		It("uses code and method labels", func() {
			labels := labelMap(mf.GetMetric()[0])
			Expect(labels).To(HaveKeyWithValue("code", "200"))
			Expect(labels).To(HaveKeyWithValue("method", "get"))
		})
	})

	Describe("request counter", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			reg := prometheus.NewRegistry()
			h := middleware.AddMetricsMiddleware(reg, dummyHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			h.ServeHTTP(rec, req)

			mf = findFamily(reg, "namespace_lister_api_counter")
		})

		It("has correct help text and type", func() {
			Expect(mf.GetHelp()).To(Equal("Number of requests completed"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_COUNTER))
		})

		It("uses code and method labels", func() {
			labels := labelMap(mf.GetMetric()[0])
			Expect(labels).To(HaveKeyWithValue("code", "200"))
			Expect(labels).To(HaveKeyWithValue("method", "post"))
		})
	})

	Describe("response size histogram", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			reg := prometheus.NewRegistry()
			h := middleware.AddMetricsMiddleware(reg, dummyHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			h.ServeHTTP(rec, req)

			mf = findFamily(reg, "namespace_lister_api_response_size")
		})

		It("has correct help text and type", func() {
			Expect(mf.GetHelp()).To(Equal("Size of responses"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_HISTOGRAM))
		})

		It("has expected bucket boundaries", func() {
			boundaries := bucketBoundaries(mf.GetMetric()[0].GetHistogram().GetBucket())
			Expect(boundaries).To(Equal([]float64{
				1.0, 2.0, 5.0,
				10.0, 20.0, 50.0,
				100.0, 200.0, 500.0,
				1000.0, 2000.0, 5000.0,
				10000.0, 20000.0, 50000.0,
				100000.0, 200000.0, 500000.0,
			}))
		})

		It("uses code and method labels", func() {
			labels := labelMap(mf.GetMetric()[0])
			Expect(labels).To(HaveKeyWithValue("code", "200"))
			Expect(labels).To(HaveKeyWithValue("method", "get"))
		})
	})

	Describe("in-flight gauge", func() {
		It("has correct help text and type", func() {
			reg := prometheus.NewRegistry()
			h := middleware.AddMetricsMiddleware(reg, dummyHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			h.ServeHTTP(rec, req)

			mf := findFamily(reg, "namespace_lister_api_requests_in_flight")
			Expect(mf.GetHelp()).To(Equal("Number of requests currently processing"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_GAUGE))
		})

		It("tracks in-flight requests", func() {
			reg := prometheus.NewRegistry()

			blocked := make(chan struct{})
			DeferCleanup(func() { close(blocked) })

			slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				<-blocked
				w.WriteHeader(http.StatusOK)
			})

			h := middleware.AddMetricsMiddleware(reg, slowHandler)

			started := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				close(started)
				h.ServeHTTP(rec, req)
			}()

			<-started
			Eventually(func() float64 {
				mf := findFamily(reg, "namespace_lister_api_requests_in_flight")
				return mf.GetMetric()[0].GetGauge().GetValue()
			}).Should(Equal(1.0))
		})
	})
})
