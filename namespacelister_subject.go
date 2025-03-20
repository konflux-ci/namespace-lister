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
	List(subjects ...rbacv1.Subject) []corev1.Namespace
}

type subjectNamespaceLister struct {
	subjectNamespacesLister SubjectNamespacesLister
}

// NewSubjectNamespaceLister builds a SubjectNamespacesLister
func NewSubjectNamespaceLister(subjectNamespacesLister SubjectNamespacesLister) NamespaceLister {
	return &subjectNamespaceLister{
		subjectNamespacesLister: subjectNamespacesLister,
	}
}

// ListNamespaces retrieves the namespaces the provided user can access from a cache calculated ahead of time
func (c *subjectNamespaceLister) ListNamespaces(ctx context.Context, username string, groups []string) (*corev1.NamespaceList, error) {
	subs := c.subjects(username, groups)
	nn := c.subjectNamespacesLister.List(subs...)

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

func (c *subjectNamespaceLister) subjects(username string, groups []string) []rbacv1.Subject {
	subs := make([]rbacv1.Subject, len(groups)+1)

	// add username subject
	subs[0] = c.parseUsername(username)

	// add groups subjects
	for i, g := range groups {
		subs[1+i] = rbacv1.Subject{
			Kind:     rbacv1.GroupKind,
			APIGroup: rbacv1.GroupName,
			Name:     g,
		}
	}

	return subs
}

func (c *subjectNamespaceLister) parseUsername(username string) rbacv1.Subject {
	if strings.HasPrefix(username, "system:serviceaccount:") {
		ss := strings.Split(username, ":")
		return rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      ss[3],
			Namespace: ss[2],
		}
	}

	return rbacv1.Subject{
		APIGroup: rbacv1.GroupName,
		Kind:     rbacv1.UserKind,
		Name:     username,
	}
}
