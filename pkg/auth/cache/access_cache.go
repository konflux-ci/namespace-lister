package cache

import (
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// stores data
type AccessCache struct {
	data atomic.Pointer[map[rbacv1.Subject][]corev1.Namespace]
}

func NewAccessCache() *AccessCache {
	return &AccessCache{
		data: atomic.Pointer[map[rbacv1.Subject][]corev1.Namespace]{},
	}
}

func (c *AccessCache) List(subject rbacv1.Subject) []corev1.Namespace {
	m := c.data.Load()
	if m == nil {
		return nil
	}
	return (*m)[subject]
}

func (c *AccessCache) Restock(data *map[rbacv1.Subject][]corev1.Namespace) {
	c.data.Store(data)
}

func (c *AccessCache) LenSubjects() int {
	return len(*c.data.Load())
}

func (c *AccessCache) LenNamespaces() int {
	n := 0
	for _, v := range *c.data.Load() {
		n += len(v)
	}
	return n
}
