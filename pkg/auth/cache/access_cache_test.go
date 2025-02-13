package cache_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
)

var _ = Describe("AuthCache", func() {
	enn := sets.New("myns")

	It("returns an empty result if it is empty", func() {
		// given
		emptyCache := cache.NewAccessCache()

		// when
		nn := emptyCache.List(rbacv1.Subject{})

		// then
		Expect(nn).To(BeEmpty())
	})

	It("matches subjects", func() {
		// given
		sub := rbacv1.Subject{Kind: "User", Name: "myuser"}
		c := cache.NewAccessCache()
		c.Restock(&map[rbacv1.Subject]sets.Set[string]{sub: enn})

		// when
		nn := c.List(sub)

		// then
		Expect(nn).To(BeEquivalentTo(enn.UnsortedList()))
	})
})
