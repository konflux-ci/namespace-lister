package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// NamespaceLister represents a lister that returns the list of namespaces a user has direct access to
type NamespaceLister interface {
	ListNamespaces(ctx context.Context, username string, groups []string) (*corev1.NamespaceList, error)
}
