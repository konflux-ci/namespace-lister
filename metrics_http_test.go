package main

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP Metrics", func() {
	var reg *prometheus.Registry
	var m httpMetrics

	BeforeEach(func() {
		reg = prometheus.NewRegistry()
		m = newHTTPMetrics(reg)
	})

	Describe("registration", func() {
		It("registers all metrics with correct names", func() {
			m.requestTiming.WithLabelValues("200", "GET").Observe(0)
			m.requestCounter.WithLabelValues("200", "GET").Inc()
			m.responseSize.WithLabelValues("200", "GET").Observe(0)
			m.inFlightGauge.Set(0)

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
			Expect(func() { newHTTPMetrics(reg) }).To(Panic())
		})
	})

	Describe("request latency histogram", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			m.requestTiming.WithLabelValues("200", "GET").Observe(0.001)
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
			Expect(labels).To(HaveKeyWithValue("method", "GET"))
		})
	})

	Describe("request counter", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			m.requestCounter.WithLabelValues("404", "POST").Inc()
			mf = findFamily(reg, "namespace_lister_api_counter")
		})

		It("has correct help text and type", func() {
			Expect(mf.GetHelp()).To(Equal("Number of requests completed"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_COUNTER))
		})

		It("uses code and method labels", func() {
			labels := labelMap(mf.GetMetric()[0])
			Expect(labels).To(HaveKeyWithValue("code", "404"))
			Expect(labels).To(HaveKeyWithValue("method", "POST"))
		})
	})

	Describe("response size histogram", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			m.responseSize.WithLabelValues("200", "GET").Observe(100)
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
			Expect(labels).To(HaveKeyWithValue("method", "GET"))
		})
	})

	Describe("in-flight gauge", func() {
		var mf *dto.MetricFamily

		BeforeEach(func() {
			m.inFlightGauge.Set(0)
			mf = findFamily(reg, "namespace_lister_api_requests_in_flight")
		})

		It("has correct help text and type", func() {
			Expect(mf.GetHelp()).To(Equal("Number of requests currently processing"))
			Expect(mf.GetType()).To(Equal(dto.MetricType_GAUGE))
		})
	})
})

func metricFamilyNames(families []*dto.MetricFamily) []string {
	names := make([]string, len(families))
	for i, f := range families {
		names[i] = f.GetName()
	}
	return names
}

func findFamily(reg *prometheus.Registry, name string) *dto.MetricFamily {
	families, err := reg.Gather()
	Expect(err).NotTo(HaveOccurred())

	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	Fail("metric family not found: " + name)
	return nil
}

func bucketBoundaries(buckets []*dto.Bucket) []float64 {
	bounds := make([]float64, len(buckets))
	for i, b := range buckets {
		bounds[i] = b.GetUpperBound()
	}
	return bounds
}

func labelMap(m *dto.Metric) map[string]string {
	labels := make(map[string]string)
	for _, lp := range m.GetLabel() {
		labels[lp.GetName()] = lp.GetValue()
	}
	return labels
}