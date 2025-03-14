package cache_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
)

var _ = Describe("AuthCache", func() {
	enn := []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "myns",
				Labels:      map[string]string{"key": "value"},
				Annotations: map[string]string{"key": "value"},
			},
		},
	}

	It("returns an empty result if it is empty", func() {
		// given
		emptyCache := cache.NewAtomicListRestockAccessCache()

		// when
		nn := emptyCache.List(rbacv1.Subject{})

		// then
		Expect(nn).To(BeEmpty())
	})

	It("matches a subject", func() {
		// given
		sub := rbacv1.Subject{Kind: "User", Name: "myuser"}
		c := cache.NewAtomicListRestockAccessCache()
		c.Restock(&cache.AccessData{sub: enn})

		// when
		nn := c.List(sub)

		// then
		Expect(nn).To(BeEquivalentTo(enn))
	})

	It("matches more subjects with overlapping namespaces", func() {
		// given
		sub1 := rbacv1.Subject{Kind: "User", Name: "myuser1"}
		sub2 := rbacv1.Subject{Kind: "User", Name: "myuser2"}
		c := cache.NewAtomicListRestockAccessCache()
		c.Restock(&cache.AccessData{
			sub1: enn,
			sub2: enn,
		})

		// when
		nn := c.List(sub1, sub2)

		// then
		Expect(nn).To(BeEquivalentTo(enn))
	})

	It("matches more subjects with overlapping namespaces and different subject kinds", func() {
		// given
		sub1 := rbacv1.Subject{Kind: rbacv1.UserKind, Name: "myuser"}
		sub2 := rbacv1.Subject{Kind: rbacv1.GroupKind, Name: "mygroup"}
		c := cache.NewAtomicListRestockAccessCache()
		c.Restock(&cache.AccessData{
			sub1: enn,
			sub2: enn,
		})

		// when
		nn := c.List(sub1, sub2)

		// then
		Expect(nn).To(BeEquivalentTo(enn))
	})

	It("matches more subjects with non-overlapping namespaces", func() {
		// given
		sub1 := rbacv1.Subject{Kind: rbacv1.UserKind, Name: "myuser1"}
		nn1 := []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "myuser1ns"}}}
		sub2 := rbacv1.Subject{Kind: rbacv1.UserKind, Name: "myuser2"}
		nn2 := []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "myuser2ns"}}}
		c := cache.NewAtomicListRestockAccessCache()
		c.Restock(&cache.AccessData{
			sub1: nn1,
			sub2: nn2,
		})

		// when
		nn := c.List(sub1, sub2)

		// then
		expectedNn := append(nn1, nn2...)
		Expect(nn).To(BeEquivalentTo(expectedNn))
	})

	It("matches more subjects with non-overlapping namespaces and different subject kinds", func() {
		// given
		sub1 := rbacv1.Subject{Kind: rbacv1.UserKind, Name: "myuser"}
		nn1 := []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "myuserns"}}}
		sub2 := rbacv1.Subject{Kind: rbacv1.GroupKind, Name: "mygroup"}
		nn2 := []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "mygroupns"}}}
		c := cache.NewAtomicListRestockAccessCache()
		c.Restock(&cache.AccessData{
			sub1: nn1,
			sub2: nn2,
		})

		// when
		nn := c.List(sub1, sub2)

		// then
		expectedNn := append(nn1, nn2...)
		Expect(nn).To(BeEquivalentTo(expectedNn))
	})
})
