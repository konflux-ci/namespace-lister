package main_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	namespacelister "github.com/konflux-ci/namespace-lister"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("CRAuthRetriever", func() {
	It("retrieves clusterrole", func(ctx context.Context) {
		// given
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "ns-get"},
		}
		cli := fake.NewClientBuilder().WithObjects(cr).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(cli)

		// when
		acr, err := authRetriever.GetClusterRole(ctx, cr.Name)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(acr).To(Equal(acr))
	})

	It("retrieves role", func(ctx context.Context) {
		// given
		r := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "ns-get", Namespace: "myns"},
		}
		cli := fake.NewClientBuilder().WithObjects(r).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(cli)

		// when
		ar, err := authRetriever.GetRole(ctx, r.Namespace, r.Name)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(ar).To(Equal(ar))
	})

	It("retrieves rolebinding", func(ctx context.Context) {
		// given
		rbl := []client.Object{
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0-0", Namespace: "myns-0"}},
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0-1", Namespace: "myns-0"}},
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-1-0", Namespace: "myns-1"}},
		}
		cli := fake.NewClientBuilder().WithObjects(rbl...).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(cli)

		// when
		arbl, err := authRetriever.ListRoleBindings(ctx, "myns-0")

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(arbl).To(ConsistOf(rbl[0:2]))
	})

	It("retrieves clusterrolebinding", func(ctx context.Context) {
		// given
		crbl := []client.Object{
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-1"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-2"}},
		}
		cli := fake.NewClientBuilder().WithObjects(crbl...).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(cli)

		// when
		acrbl, err := authRetriever.ListClusterRoleBindings(ctx)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(acrbl).To(BeEmpty())
	})
})
