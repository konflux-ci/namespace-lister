package main

import (
	"context"
	"strings"

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

func NewNamespaceListerForSubject(subjectNamespacesLister SubjectNamespacesLister) NamespaceLister {
	return &subjectNamespaceLister{
		subjectNamespacesLister: subjectNamespacesLister,
	}
}

func (c *subjectNamespaceLister) ListNamespaces(ctx context.Context, username string) (*corev1.NamespaceList, error) {
	sub := c.parseUsername(username)
	nn := c.subjectNamespacesLister.List(sub)

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

func (c *subjectNamespaceLister) parseUsername(username string) rbacv1.Subject {
	if strings.HasPrefix(username, "system:serviceaccount:") {
		ss := strings.Split(username, ":")
		return rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      ss[3],
			Namespace: ss[2],
		}
	}

	return rbacv1.Subject{
		APIGroup: rbacv1.GroupName,
		Kind:     "User",
		Name:     username,
	}
}
