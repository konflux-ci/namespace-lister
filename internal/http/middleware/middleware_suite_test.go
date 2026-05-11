package middleware_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Middleware Suite")
}

func getName(f *dto.MetricFamily) string { return f.GetName() }

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

func labelMap(m *dto.Metric) map[string]string {
	labels := make(map[string]string)
	for _, lp := range m.GetLabel() {
		labels[lp.GetName()] = lp.GetValue()
	}
	return labels
}

func bucketBoundaries(buckets []*dto.Bucket) []float64 {
	bounds := make([]float64, len(buckets))
	for i, b := range buckets {
		bounds[i] = b.GetUpperBound()
	}
	return bounds
}
