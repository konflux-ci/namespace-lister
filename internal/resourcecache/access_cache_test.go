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

			families, err := reg.Gather()
			Expect(err).NotTo(HaveOccurred())
			Expect(families).NotTo(BeEmpty())
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

	Describe("GetValidResyncPeriodFromEnvOrZero", Serial, func() {
		It("returns zero when env is not set", func(ctx context.Context) {
			// given
			if v, ok := os.LookupEnv(constants.EnvCacheResyncPeriod); ok {
				Expect(os.Unsetenv(constants.EnvCacheResyncPeriod)).To(Succeed()) //nolint:usetesting
				defer os.Setenv(constants.EnvCacheResyncPeriod, v)                //nolint:usetesting
			}

			// when
			d := resourcecache.GetValidResyncPeriodFromEnvOrZero(ctx)

			// then
			Expect(d).To(Equal(time.Duration(0)))
		})

		DescribeTable("parses env value",
			func(ctx context.Context, envValue string, expected time.Duration) {
				// given
				GinkgoT().Setenv(constants.EnvCacheResyncPeriod, envValue)

				// when
				d := resourcecache.GetValidResyncPeriodFromEnvOrZero(ctx)

				// then
				Expect(d).To(Equal(expected))
			},
			Entry("valid duration 5m", "5m", 5*time.Minute),
			Entry("valid duration 30s", "30s", 30*time.Second),
			Entry("valid duration 1h", "1h", time.Hour),
			Entry("invalid duration returns zero", "not-a-duration", time.Duration(0)),
			Entry("negative duration returns zero", "-1h", time.Duration(0)),
		)
	})
})
