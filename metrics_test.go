package main

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func metricFamilyNames(families []*dto.MetricFamily) []string {
	names := make([]string, len(families))
	for i, f := range families {
		names[i] = f.GetName()
	}
	return names
}

var _ = Describe("InitRegistry", func() {
	It("registers process collector on a fresh registry", func() {
		reg := prometheus.NewRegistry()
		Expect(func() { InitRegistry(reg) }).NotTo(Panic())

		families, err := reg.Gather()
		Expect(err).NotTo(HaveOccurred())

		names := metricFamilyNames(families)
		Expect(names).To(ContainElement(HavePrefix("namespace_lister_process_")))
	})

	It("panics on duplicate registration", func() {
		reg := prometheus.NewRegistry()
		InitRegistry(reg)
		Expect(func() { InitRegistry(reg) }).To(Panic())
	})
})
