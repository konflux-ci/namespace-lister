package suite

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestNamespace(name, run string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"konflux.ci/type":           "user",
				"namespace-lister/scope":    "acceptance-tests",
				"namespace-lister/test-run": run,
			},
		},
	}
}

func newAccessRoleBinding(name, namespace string, subject rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "namespace-get",
			APIGroup: rbacv1.GroupName,
		},
		Subjects: []rbacv1.Subject{subject},
	}
}
