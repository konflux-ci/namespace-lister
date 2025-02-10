package cache

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// AccessCache represents a cache that can list namespaces a subject has access to.
// Data in the cache can be updated via the Restock method.
type AccessCache interface {
	// List lists all the namespaces a subject has access to
	List(subject rbacv1.Subject) []corev1.Namespace
	// Restock updates the data stored in the cache
	Restock(data *map[rbacv1.Subject][]corev1.Namespace)
}

// NewAtomicListRestockAccessCache builds an AccessCache leveraging on the AtomicListRestockCache
func NewAtomicListRestockAccessCache() AccessCache {
	return newAtomicListRestockCache[rbacv1.Subject, []corev1.Namespace]()
}
