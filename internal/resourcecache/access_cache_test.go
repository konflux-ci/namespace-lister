package resourcecache_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/konflux-ci/namespace-lister/internal/constants"
	"github.com/konflux-ci/namespace-lister/internal/resourcecache"
)

var _ = Describe("Access Cache", func() {
	Describe("BuildAndRegisterAccessCacheMetrics", func() {
		var reg *prometheus.Registry

		BeforeEach(func() {
			reg = prometheus.NewRegistry()
		})

		It("returns nil metrics when registry is nil", func() {
			// when
			metrics, err := resourcecache.BuildAndRegisterAccessCacheMetrics(nil)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(metrics).To(BeNil())
		})

		It("registers and returns metrics with a valid registry", func() {
			// when
			metrics, err := resourcecache.BuildAndRegisterAccessCacheMetrics(reg)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(metrics).NotTo(BeNil())
		})

		It("returns an error on duplicate registration", func() {
			// given
			_, err := resourcecache.BuildAndRegisterAccessCacheMetrics(reg)
			Expect(err).NotTo(HaveOccurred())

			// when
			_, err = resourcecache.BuildAndRegisterAccessCacheMetrics(reg)

			// then
			Expect(err).To(BeAssignableToTypeOf(prometheus.AlreadyRegisteredError{}))
		})
	})

	Describe("GetResyncPeriodFromEnvOrZero", func() {
		const envKey = constants.EnvCacheResyncPeriod

		It("returns zero when env is not set", func(ctx context.Context) {
			// given
			GinkgoT().Setenv(envKey, "dummy")
			Expect(os.Unsetenv(envKey)).To(Succeed()) //nolint:usetesting // need truly-unset env for this test

			// when
			d := resourcecache.GetResyncPeriodFromEnvOrZero(ctx)

			// then
			Expect(d).To(Equal(time.Duration(0)))
		})

		It("parses a valid duration string", func(ctx context.Context) {
			// given
			GinkgoT().Setenv(envKey, "5m")

			// when
			d := resourcecache.GetResyncPeriodFromEnvOrZero(ctx)

			// then
			Expect(d).To(Equal(5 * time.Minute))
		})

		It("returns zero for an invalid duration string", func(ctx context.Context) {
			// given
			GinkgoT().Setenv(envKey, "not-a-duration")

			// when
			d := resourcecache.GetResyncPeriodFromEnvOrZero(ctx)

			// then
			Expect(d).To(Equal(time.Duration(0)))
		})
	})
})
