package main_test

import (
	"context"
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	namespacelister "github.com/konflux-ci/namespace-lister"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Namespacelister", func() {

	var (
		ctx    = context.TODO()
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	)

	DescribeTable("when listing namespaces", func(
		nn corev1.NamespaceList,
		cr rbacv1.ClusterRoleList,
		crb rbacv1.ClusterRoleBindingList,
		r rbacv1.RoleList,
		rb rbacv1.RoleBindingList,
		expected corev1.NamespaceList,
	) {
		// given
		reader := fake.NewClientBuilder().WithLists(&nn, &cr, &crb, &r, &rb).Build()
		authorizer := namespacelister.NewAuthorizer(ctx, reader, logger)
		nsl := namespacelister.NewNamespaceLister(reader, authorizer, logger)

		// when
		ann, err := nsl.ListNamespaces(ctx, "user")

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(&expected).To(BeEquivalentTo(ann))
	},
		Entry("returns no namespaces if no namespaces exist",
			corev1.NamespaceList{},
			rbacv1.ClusterRoleList{},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.RoleList{},
			rbacv1.RoleBindingList{},
			corev1.NamespaceList{Items: []corev1.Namespace{}},
		),
		Entry("returns no namespaces if no clusterroles or roles exist",
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
			}},
			rbacv1.ClusterRoleList{},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.RoleList{},
			rbacv1.RoleBindingList{},
			corev1.NamespaceList{Items: []corev1.Namespace{}},
		),
		Entry("returns no namespace if a clusterrole exists but no clusterrolebinding",
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
			}},
			rbacv1.ClusterRoleList{Items: []rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ns-get"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{""},
							Verbs:         []string{"get"},
							Resources:     []string{"namespaces"},
							ResourceNames: []string{"myns-1"},
						},
					},
				},
			}},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.RoleList{},
			rbacv1.RoleBindingList{},
			corev1.NamespaceList{Items: []corev1.Namespace{}},
		),
		Entry("returns the namespace if both the clusterrole and a valid clusterrolebinding exist",
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
			}},
			rbacv1.ClusterRoleList{Items: []rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ns-get"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{""},
							Verbs:         []string{"get"},
							Resources:     []string{"namespaces"},
							ResourceNames: []string{"myns-1"},
						},
					},
				},
			}},
			rbacv1.ClusterRoleBindingList{Items: []rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ns-get:user"},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "User",
							Name:     "user",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "ClusterRole",
						APIGroup: rbacv1.GroupName,
						Name:     "ns-get",
					},
				},
			}},
			rbacv1.RoleList{},
			rbacv1.RoleBindingList{},
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1", ResourceVersion: "999"}},
			}},
		),
		Entry("returns no namespace if the role exists but no rolebinding",
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
			}},
			rbacv1.ClusterRoleList{},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.RoleList{Items: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ns-get", Namespace: "myns-1"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Verbs:     []string{"get"},
							Resources: []string{"namespaces"},
						},
					},
				},
			}},
			rbacv1.RoleBindingList{},
			corev1.NamespaceList{Items: []corev1.Namespace{}},
		),
		Entry("returns a namespace if both the role and the rolebinding exist",
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
			}},
			rbacv1.ClusterRoleList{},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.RoleList{Items: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ns-get",
						Namespace: "myns-1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Verbs:     []string{"get"},
							Resources: []string{"namespaces"},
						},
					},
				},
			}},
			rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ns-get:user", Namespace: "myns-1"},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "User",
							Name:     "user",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "Role",
						APIGroup: rbacv1.GroupName,
						Name:     "ns-get",
					},
				},
			}},
			corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myns-1", ResourceVersion: "999"}},
			}},
		),
	)
})
