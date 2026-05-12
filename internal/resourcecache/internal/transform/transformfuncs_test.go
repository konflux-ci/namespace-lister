package transform_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/namespace-lister/internal/resourcecache/internal/transform"
	"github.com/konflux-ci/namespace-lister/internal/resourcecache/internal/transform/mocks"
)

var managedFields = []metav1.ManagedFieldsEntry{{
	Manager: "test-manager", Operation: metav1.ManagedFieldsOperationApply,
}}

var _ = Describe("Transform Functions", func() {
	DescribeTable("TrimAnnotations",
		func(ns *corev1.Namespace) {
			// when
			result, err := transform.TrimAnnotations()(ns.DeepCopy())

			// then
			By("ensuring annotations were stripped")
			Expect(err).NotTo(HaveOccurred())
			out := result.(*corev1.Namespace)
			Expect(out).NotTo(BeNil())
			Expect(out.Annotations).To(BeNil())

			By("ensuring other fields were not mutated")
			Expect(ns).To(
				WithTransform(func(ns *corev1.Namespace) *corev1.Namespace {
					ns.Annotations = nil
					return ns
				}, BeEquivalentTo(out)))
		},
		Entry("strips annotations from an object",
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-ns",
					Labels:      map[string]string{"team": "infra"},
					Annotations: map[string]string{"note": "value"},
				},
			}),
		Entry("is a no-op when there are no annotations",
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-ns",
					Labels: map[string]string{"team": "infra"},
				},
				Spec: corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes}},
			}),
	)

	Describe("MergeTransformFunc", func() {
		var ctrl *gomock.Controller

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			DeferCleanup(ctrl.Finish)
		})

		It("chains multiple transforms passing intermediate results", func() {
			// given
			nsIn := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			}
			nsOut := nsIn.DeepCopy()

			tf1 := mocks.NewMockTransformFunc(ctrl)
			tf2 := mocks.NewMockTransformFunc(ctrl)
			gomock.InOrder(
				tf1.EXPECT().TransformFunc(nsIn).Return(nsOut, nil).Times(1),
				tf2.EXPECT().TransformFunc(gomock.Any()).
					Do(func(a any) {
						if a.(*corev1.Namespace) != nsOut {
							Fail("intermediate result was not propagated as expected")
						}
					}).
					Return(nsOut, nil).
					Times(1),
			)

			// when
			merged := transform.MergeTransformFunc(tf1.TransformFunc, tf2.TransformFunc)
			result, err := merged(nsIn)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(nsOut))
		})

		It("short-circuits on error", func() {
			// given
			tf1 := mocks.NewMockTransformFunc(ctrl)
			tf2 := mocks.NewMockTransformFunc(ctrl)
			tf1.EXPECT().TransformFunc(gomock.Any()).Return(nil, errors.New("boom")).Times(1)
			tf2.EXPECT().TransformFunc(gomock.Any()).Times(0)

			// when
			merged := transform.MergeTransformFunc(tf1.TransformFunc, tf2.TransformFunc)
			result, err := merged(&corev1.Namespace{})

			// then
			Expect(err).To(MatchError("boom"))
			Expect(result).To(BeNil())
		})
	})

	Describe("TrimRole", func() {
		DescribeTable("strips annotations, managed fields, and non-namespace rules",
			func(role *rbacv1.Role, expectedRules []rbacv1.PolicyRule) {
				// when
				result, err := transform.TrimRole()(role.DeepCopy())

				// then
				By("ensuring fields expected to be stripped were stripped")
				Expect(err).NotTo(HaveOccurred())
				out := result.(*rbacv1.Role)
				Expect(out.Rules).To(BeEquivalentTo(expectedRules))
				Expect(out.Annotations).To(BeNil())
				Expect(out.ManagedFields).To(BeEmpty())

				By("ensuring other fields were not mutated")
				Expect(role).To(
					WithTransform(func(r *rbacv1.Role) *rbacv1.Role {
						r.Annotations = nil
						r.ManagedFields = nil
						r.Rules = out.Rules
						return r
					}, BeEquivalentTo(out)))
			},
			Entry("with annotations",
				&rbacv1.Role{
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
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("with managed fields",
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:          "ns-reader",
						Namespace:     "default",
						ManagedFields: managedFields,
					},
					Rules: []rbacv1.PolicyRule{{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					}},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("keeps namespace rule among mixed rules",
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-reader", Namespace: "default"},
					Rules: []rbacv1.PolicyRule{
						{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
						{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"get"}},
					},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("normalizes matching rule to namespace-get only",
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "broad-reader", Namespace: "default"},
					Rules: []rbacv1.PolicyRule{{
						APIGroups: []string{"", "apps"},
						Resources: []string{"namespaces", "pods"},
						Verbs:     []string{"get", "list", "watch"},
					}},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
		)

		DescribeTable("returns nil for Roles without namespace-get rules",
			func(rules []rbacv1.PolicyRule) {
				result, err := transform.TrimRole()(&rbacv1.Role{Rules: rules})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			},
			Entry("pods-only rule",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"},
				}}),
			Entry("wildcard resources do not match",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"*"}, Verbs: []string{"get"},
				}}),
			Entry("wildcard verbs do not match",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"*"},
				}}),
			Entry("wrong API group",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{"apps"}, Resources: []string{"namespaces"}, Verbs: []string{"get"},
				}}),
			Entry("empty rules", []rbacv1.PolicyRule(nil)),
		)

		It("errors when given a non-Role object", func() {
			v, err := transform.TrimRole()(&corev1.Namespace{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected Role"))
			Expect(v).To(BeNil())
		})
	})

	Describe("TrimClusterRole", func() {
		DescribeTable("strips annotations, managed fields, and non-namespace rules",
			func(cr *rbacv1.ClusterRole, expectedRules []rbacv1.PolicyRule) {
				// when
				result, err := transform.TrimClusterRole()(cr.DeepCopy())

				// then
				By("ensuring fields expected to be stripped were stripped")
				Expect(err).NotTo(HaveOccurred())
				out := result.(*rbacv1.ClusterRole)
				Expect(out.Rules).To(BeEquivalentTo(expectedRules))
				Expect(out.Annotations).To(BeNil())
				Expect(out.ManagedFields).To(BeEmpty())

				By("ensuring other fields were not mutated")
				Expect(cr).To(
					WithTransform(func(c *rbacv1.ClusterRole) *rbacv1.ClusterRole {
						c.Annotations = nil
						c.ManagedFields = nil
						c.Rules = out.Rules
						return c
					}, BeEquivalentTo(out)))
			},
			Entry("with annotations",
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "ns-reader",
						Annotations: map[string]string{"note": "value"},
					},
					Rules: []rbacv1.PolicyRule{{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					}},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("with managed fields",
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:          "ns-reader",
						ManagedFields: managedFields,
					},
					Rules: []rbacv1.PolicyRule{{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					}},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("keeps namespace rule among mixed rules",
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-reader"},
					Rules: []rbacv1.PolicyRule{
						{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
						{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"get"}},
					},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
			Entry("normalizes matching rule to namespace-get only",
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{Name: "broad-reader"},
					Rules: []rbacv1.PolicyRule{{
						APIGroups: []string{"", "apps"},
						Resources: []string{"namespaces", "pods"},
						Verbs:     []string{"get", "list", "watch"},
					}},
				},
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				}}),
		)

		DescribeTable("returns nil for ClusterRoles without namespace-get rules",
			func(rules []rbacv1.PolicyRule) {
				result, err := transform.TrimClusterRole()(&rbacv1.ClusterRole{Rules: rules})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			},
			Entry("pods-only rule",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"},
				}}),
			Entry("wildcard resources do not match",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"*"}, Verbs: []string{"get"},
				}}),
			Entry("wildcard verbs do not match",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"*"},
				}}),
			Entry("wrong API group",
				[]rbacv1.PolicyRule{{
					APIGroups: []string{"apps"}, Resources: []string{"namespaces"}, Verbs: []string{"get"},
				}}),
			Entry("empty rules", []rbacv1.PolicyRule(nil)),
		)

		It("errors when given a non-ClusterRole object", func() {
			v, err := transform.TrimClusterRole()(&corev1.Namespace{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a ClusterRole"))
			Expect(v).To(BeNil())
		})
	})

	Describe("TrimNamespace", func() {
		It("strips managed fields, spec, and status", func() {
			// given
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:          "test-ns",
					Labels:        map[string]string{"team": "infra"},
					ManagedFields: managedFields,
				},
				Spec:   corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes}},
				Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
			}

			// when
			result, err := transform.TrimNamespace()(ns.DeepCopy())

			// then
			By("ensuring fields expected to be stripped were stripped")
			Expect(err).NotTo(HaveOccurred())
			out := result.(*corev1.Namespace)
			Expect(out.ManagedFields).To(BeEmpty())
			Expect(out.Spec).To(BeEquivalentTo(corev1.NamespaceSpec{}))
			Expect(out.Status).To(BeEquivalentTo(corev1.NamespaceStatus{}))

			By("ensuring other fields were not mutated")
			Expect(ns).To(
				WithTransform(func(ns *corev1.Namespace) *corev1.Namespace {
					ns.ManagedFields = nil
					ns.Spec = corev1.NamespaceSpec{}
					ns.Status = corev1.NamespaceStatus{}
					return ns
				}, BeEquivalentTo(out)))
		})

		It("errors when given a non-Namespace object", func() {
			v, err := transform.TrimNamespace()(&rbacv1.Role{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a Namespace"))
			Expect(v).To(BeNil())
		})
	})

	Describe("TrimRoleBinding", func() {
		DescribeTable("strips annotations and managed fields",
			func(rb *rbacv1.RoleBinding) {
				// when
				result, err := transform.TrimRoleBinding()(rb.DeepCopy())

				// then
				By("ensuring fields expected to be stripped were stripped")
				Expect(err).NotTo(HaveOccurred())
				out := result.(*rbacv1.RoleBinding)
				Expect(out.Annotations).To(BeNil())
				Expect(out.ManagedFields).To(BeEmpty())

				By("ensuring other fields were not mutated")
				Expect(rb).To(
					WithTransform(func(r *rbacv1.RoleBinding) *rbacv1.RoleBinding {
						r.Annotations = nil
						r.ManagedFields = nil
						return r
					}, BeEquivalentTo(out)))
			},
			Entry("with annotations",
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-rb",
						Annotations: map[string]string{"note": "value"},
					},
				}),
			Entry("with managed fields",
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:          "test-rb",
						ManagedFields: managedFields,
					},
				}),
		)
	})

	Describe("TrimClusterRoleBinding", func() {
		DescribeTable("strips annotations and managed fields",
			func(crb *rbacv1.ClusterRoleBinding) {
				// when
				result, err := transform.TrimClusterRoleBinding()(crb.DeepCopy())

				// then
				By("ensuring fields expected to be stripped were stripped")
				Expect(err).NotTo(HaveOccurred())
				out := result.(*rbacv1.ClusterRoleBinding)
				Expect(out.Annotations).To(BeNil())
				Expect(out.ManagedFields).To(BeEmpty())

				By("ensuring other fields were not mutated")
				Expect(crb).To(
					WithTransform(func(c *rbacv1.ClusterRoleBinding) *rbacv1.ClusterRoleBinding {
						c.Annotations = nil
						c.ManagedFields = nil
						return c
					}, BeEquivalentTo(out)))
			},
			Entry("with annotations",
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-crb",
						Annotations: map[string]string{"note": "value"},
					},
				}),
			Entry("with managed fields",
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:          "test-crb",
						ManagedFields: managedFields,
					},
				}),
		)
	})
})
