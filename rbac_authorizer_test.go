package main_test

import (
	"context"
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	namespacelister "github.com/konflux-workspaces/namespace-lister"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("CRAuthRetriever", func() {
	var (
		ctx    context.Context
		logger *slog.Logger
	)

	BeforeEach(func() {
		ctx = context.TODO()
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	})

	It("retrieves clusterrole", func() {
		// given
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "ns-get"},
		}
		cli := fake.NewClientBuilder().WithObjects(cr).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(ctx, cli, logger)

		// when
		acr, err := authRetriever.GetClusterRole(cr.Name)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(acr).To(Equal(acr))
	})

	It("retrieves role", func() {
		// given
		r := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "ns-get", Namespace: "myns"},
		}
		cli := fake.NewClientBuilder().WithObjects(r).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(ctx, cli, logger)

		// when
		ar, err := authRetriever.GetRole(r.Namespace, r.Name)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(ar).To(Equal(ar))
	})

	It("retrieves rolebinding", func() {
		// given
		rbl := []client.Object{
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0-0", Namespace: "myns-0"}},
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0-1", Namespace: "myns-0"}},
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-1-0", Namespace: "myns-1"}},
		}
		cli := fake.NewClientBuilder().WithObjects(rbl...).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(ctx, cli, logger)

		// when
		arbl, err := authRetriever.ListRoleBindings("myns-0")

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(arbl).To(ConsistOf(rbl[0:2]))
	})

	It("retrieves clusterrolebinding", func() {
		// given
		crbl := []client.Object{
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-0"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-1"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "ns-get-2"}},
		}
		cli := fake.NewClientBuilder().WithObjects(crbl...).Build()
		authRetriever := namespacelister.NewCRAuthRetriever(ctx, cli, logger)

		// when
		acrbl, err := authRetriever.ListClusterRoleBindings()

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(acrbl).To(ConsistOf(crbl))
	})
})
