package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ NamespaceLister = &subjectNamespaceLister{}

type SubjectNamespacesLister interface {
	List(subject rbacv1.Subject) []corev1.Namespace
}

type subjectNamespaceLister struct {
	subjectNamespacesLister SubjectNamespacesLister
}

func NewSubjectNamespacesLister(subjectNamespacesLister SubjectNamespacesLister) NamespaceLister {
	return &subjectNamespaceLister{
		subjectNamespacesLister: subjectNamespacesLister,
	}
}

func (c *subjectNamespaceLister) ListNamespaces(ctx context.Context, username string) (*corev1.NamespaceList, error) {
	nn := c.subjectNamespacesLister.List(rbacv1.Subject{
		APIGroup: rbacv1.GroupName,
		Kind:     "User",
		Name:     username,
	})

	// list all namespaces
	return &corev1.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			// even though `kubectl get namespaces -o yaml` is showing `kind: List`
			// the plain response from the APIServer is using `kind: NamespaceList`.
			// Use `kubectl get namespaces -v9` to inspect the APIServer plain response.
			Kind:       "NamespaceList",
			APIVersion: corev1.SchemeGroupVersion.Version,
		},
		Items: nn,
	}, nil
}
