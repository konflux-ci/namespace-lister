package transform_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/namespace-lister/internal/resourcecache/internal/transform"
)

var _ = Describe("Transform Functions", func() {
	Describe("TrimAnnotations", func() {
		It("strips annotations from an object", func() {
			// given
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-ns",
					Labels:      map[string]string{"team": "infra"},
					Annotations: map[string]string{"note": "value"},
				},
			}

			// when
			result, err := transform.TrimAnnotations()(ns)

			// then
			Expect(err).NotTo(HaveOccurred())
			out := result.(*corev1.Namespace)
			Expect(out.Annotations).To(BeNil())
			Expect(out.Labels).To(Equal(map[string]string{"team": "infra"}))
			Expect(out.Name).To(Equal("test-ns"))
		})

		It("is a no-op when there are no annotations", func() {
			// given
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}

			// when
			result, err := transform.TrimAnnotations()(ns)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result.(*corev1.Namespace).Annotations).To(BeNil())
		})
	})

	Describe("MergeTransformFunc", func() {
		It("chains multiple transforms in order", func() {
			// given
			addLabel := func(i interface{}) (interface{}, error) {
				ns := i.(*corev1.Namespace)
				if ns.Labels == nil {
					ns.Labels = map[string]string{}
				}
				ns.Labels["added"] = "true"
				return ns, nil
			}
			clearSpec := func(i interface{}) (interface{}, error) {
				ns := i.(*corev1.Namespace)
				ns.Spec = corev1.NamespaceSpec{}
				return ns, nil
			}
			merged := transform.MergeTransformFunc(addLabel, clearSpec)
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes}},
			}

			// when
			result, err := merged(ns)

			// then
			Expect(err).NotTo(HaveOccurred())
			out := result.(*corev1.Namespace)
			Expect(out.Labels).To(HaveKeyWithValue("added", "true"))
			Expect(out.Spec.Finalizers).To(BeEmpty())
		})

		It("short-circuits on error", func() {
			// given
			failing := func(i interface{}) (interface{}, error) {
				return nil, errors.New("boom")
			}
			shouldNotRun := func(i interface{}) (interface{}, error) {
				Fail("should not be called")
				return i, nil
			}
			merged := transform.MergeTransformFunc(failing, shouldNotRun)

			// when
			_, err := merged(&corev1.Namespace{})

			// then
			Expect(err).To(MatchError("boom"))
		})
	})

	Describe("TrimRole", func() {
		It("keeps a Role with namespace-get rules and strips annotations", func() {
			// given
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "ns-reader",
					Namespace:   "default",
					Annotations: map[string]string{"note": "value"},
				},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}},
			}

			// when
			result, err := transform.TrimRole()(role)

			// then
			Expect(err).NotTo(HaveOccurred())
			out := result.(*rbacv1.Role)
			Expect(out.Rules).To(HaveLen(1))
			Expect(out.Annotations).To(BeNil())
		})

		It("returns nil for a Role with no namespace-related rules", func() {
			// given
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "pod-reader"},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				}},
			}

			// when
			result, err := transform.TrimRole()(role)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("errors when given a non-Role object", func() {
			// when
			_, err := transform.TrimRole()(&corev1.Namespace{})

			// then
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected Role"))
		})
	})

	Describe("TrimClusterRole", func() {
		It("keeps a ClusterRole with namespace-get rules and strips annotations", func() {
			// given
			cr := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "ns-reader",
					Annotations: map[string]string{"note": "value"},
				},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}},
			}

			// when
			result, err := transform.TrimClusterRole()(cr)

			// then
			Expect(err).NotTo(HaveOccurred())
			out := result.(*rbacv1.ClusterRole)
			Expect(out.Rules).To(HaveLen(1))
			Expect(out.Annotations).To(BeNil())
		})

		It("returns nil for a ClusterRole with no namespace-related rules", func() {
			// given
			cr := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "pod-reader"},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				}},
			}

			// when
			result, err := transform.TrimClusterRole()(cr)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("errors when given a non-ClusterRole object", func() {
			// when
			_, err := transform.TrimClusterRole()(&corev1.Namespace{})

			// then
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a ClusterRole"))
		})
	})
})
