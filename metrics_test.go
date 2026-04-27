package main

import (
	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
