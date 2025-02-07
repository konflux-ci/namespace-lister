package mocks

import (
	"errors"
	"slices"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/kubernetes/pkg/registry/rbac/validation"
)

var (
	_ validation.RoleGetter               = &MockStaticRoles{}
	_ validation.ClusterRoleGetter        = &MockStaticRoles{}
	_ validation.RoleBindingLister        = &MockStaticRoles{}
	_ validation.ClusterRoleBindingLister = &MockStaticRoles{}
)

// MockStaticRoles serves Roles, ClusterRoles, RoleBindings, and ClusterRoleBindings
// from static lists, implementing the interfaces required by rbac's SubjectAccessEvaluator
type MockStaticRoles struct {
	Roles               []*rbacv1.Role
	RoleBindings        []*rbacv1.RoleBinding
	ClusterRoles        []*rbacv1.ClusterRole
	ClusterRoleBindings []*rbacv1.ClusterRoleBinding
}

func findFunc[E any, S ~[]E](s S, f func(e E) bool) (E, bool) {
	if i := slices.IndexFunc(s, f); i != -1 {
		return s[i], true
	}

	var e E
	return e, false
}

func filterFunc[E any, S ~[]E](s S, f func(e E) bool) S {
	ns := S{}
	for _, e := range s {
		if f(e) {
			ns = append(ns, e)
		}
	}
	return ns
}

func (r *MockStaticRoles) GetRole(namespace, name string) (*rbacv1.Role, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required when getting Roles")
	}

	matchRole := func(r *rbacv1.Role) bool { return r.Name == name && r.Namespace == namespace }
	if r, ok := findFunc(r.Roles, matchRole); ok {
		return r, nil
	}
	return nil, errors.New("Role not found")
}

func (r *MockStaticRoles) GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	matchClusterRole := func(cr *rbacv1.ClusterRole) bool { return cr.Name == name }
	if r, ok := findFunc(r.ClusterRoles, matchClusterRole); ok {
		return r, nil
	}
	return nil, errors.New("ClusterRole not found")
}

func (r *MockStaticRoles) ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required when getting RoleBindings")
	}

	matchRoleBindings := func(rb *rbacv1.RoleBinding) bool { return rb.Namespace == namespace }
	return filterFunc(r.RoleBindings, matchRoleBindings), nil
}

func (r *MockStaticRoles) ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	return r.ClusterRoleBindings, nil
}
