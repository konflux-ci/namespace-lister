package main

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

// NewAuthorizer builds a new RBACAuthorizer
func NewAuthorizer(cli client.Reader) *rbac.RBACAuthorizer {
	aur := &CRAuthRetriever{cli}
	ra := rbac.New(aur, aur, aur, aur)
	return ra
}

// CRAuthRetriever implements RoleGetter, RoleBindingLister, ClusterRoleGetter, ClusterRoleBindingLister
// on top of a Controller-Runtime's Reader
type CRAuthRetriever struct {
	cli client.Reader
}

// NewCRAuthRetriever builds a new CRAuthRetriever
func NewCRAuthRetriever(cli client.Reader) *CRAuthRetriever {
	return &CRAuthRetriever{
		cli: cli,
	}
}

// GetRole retrieves a Role by namespace and name
func (r *CRAuthRetriever) GetRole(ctx context.Context, namespace, name string) (*rbacv1.Role, error) {
	ro := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.cli.Get(ctx, client.ObjectKeyFromObject(&ro), &ro); err != nil {
		return nil, err
	}
	return &ro, nil
}

// ListRoleBindings retrieves RoleBindings from a namespace
func (r *CRAuthRetriever) ListRoleBindings(ctx context.Context, namespace string) ([]*rbacv1.RoleBinding, error) {
	rbb := rbacv1.RoleBindingList{}
	if err := r.cli.List(ctx, &rbb, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	rbbp := make([]*rbacv1.RoleBinding, len(rbb.Items))
	for i, rb := range rbb.Items {
		rbbp[i] = rb.DeepCopy()
	}
	return rbbp, nil
}

// GetClusterRole retrieves a ClusterRole by name
func (r *CRAuthRetriever) GetClusterRole(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
	ro := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := r.cli.Get(ctx, client.ObjectKeyFromObject(&ro), &ro); err != nil {
		return nil, err
	}
	return &ro, nil
}

// ListClusterRoleBindings retrieves ClusterRoleBindings
func (r *CRAuthRetriever) ListClusterRoleBindings(context.Context) ([]*rbacv1.ClusterRoleBinding, error) {
	return make([]*rbacv1.ClusterRoleBinding, 0), nil
}
