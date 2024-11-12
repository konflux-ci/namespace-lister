package main

import (
	"context"
	"log/slog"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

func NewAuthorizer(ctx context.Context, cli client.Reader, l *slog.Logger) *rbac.RBACAuthorizer {
	aur := &CRAuthRetriever{cli, ctx, l}
	ra := rbac.New(aur, aur, aur, aur)
	return ra
}

type CRAuthRetriever struct {
	cli client.Reader
	ctx context.Context
	l   *slog.Logger
}

func NewCRAuthRetriever(ctx context.Context, cli client.Reader, l *slog.Logger) *CRAuthRetriever {
	return &CRAuthRetriever{
		cli: cli,
		ctx: ctx,
		l:   l,
	}
}

func (r *CRAuthRetriever) GetRole(namespace, name string) (*rbacv1.Role, error) {
	ro := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.cli.Get(r.ctx, client.ObjectKeyFromObject(&ro), &ro); err != nil {
		return nil, err
	}
	r.l.Debug("getting role", "namespace", namespace, "name", name, "role", ro)
	return &ro, nil
}

func (r *CRAuthRetriever) ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	rbb := rbacv1.RoleBindingList{}
	if err := r.cli.List(r.ctx, &rbb, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	r.l.Debug("listing rolebindings", "namespace", namespace, "rolebindings", rbb)

	rbbp := make([]*rbacv1.RoleBinding, len(rbb.Items))
	for i, rb := range rbb.Items {
		rbbp[i] = rb.DeepCopy()
	}
	return rbbp, nil
}

func (r *CRAuthRetriever) GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	ro := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := r.cli.Get(r.ctx, client.ObjectKeyFromObject(&ro), &ro); err != nil {
		return nil, err
	}
	r.l.Debug("getting clusterrole", "name", name, "clusterrole", ro)
	return &ro, nil
}

func (r *CRAuthRetriever) ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	rbb := rbacv1.ClusterRoleBindingList{}
	if err := r.cli.List(r.ctx, &rbb); err != nil {
		return nil, err
	}
	r.l.Debug("listing clusterrolebindings", "clusterrolebindings", rbb)

	rbbp := make([]*rbacv1.ClusterRoleBinding, len(rbb.Items))
	for i, rb := range rbb.Items {
		rbbp[i] = rb.DeepCopy()
	}
	return rbbp, nil
}
