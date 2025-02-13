package cache

import (
	"sync/atomic"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// stores data
type AccessCache struct {
	data atomic.Pointer[map[rbacv1.Subject]sets.Set[string]]
}

func NewAccessCache() *AccessCache {
	return &AccessCache{
		data: atomic.Pointer[map[rbacv1.Subject]sets.Set[string]]{},
	}
}

func (c *AccessCache) List(subject rbacv1.Subject) []string {
	m := c.data.Load()
	if m == nil {
		return nil
	}
	return (*m)[subject].UnsortedList()
}

func (c *AccessCache) Restock(data *map[rbacv1.Subject]sets.Set[string]) {
	c.data.Store(data)
}
