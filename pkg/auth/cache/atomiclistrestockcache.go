package cache

import (
	"sync/atomic"
)

// AtomicListRestockCache is a key value cache that stores data behind an atomic.Pointer.
// It exposes APIs to list data by key and to update data.
type AtomicListRestockCache[K comparable, S ~[]E, E any] struct {
	data atomic.Pointer[map[K]S]
}

func newAtomicListRestockCache[K comparable, S ~[]E, E any]() *AtomicListRestockCache[K, S, E] {
	return &AtomicListRestockCache[K, S, E]{
		data: atomic.Pointer[map[K]S]{},
	}
}

// List returns the data stored for the given key
func (c *AtomicListRestockCache[K, S, E]) List(key K) S {
	m := c.data.Load()
	if m == nil {
		return nil
	}
	return (*m)[key]
}

// Restock fully replaces data stored in the cache
func (c *AtomicListRestockCache[K, S, E]) Restock(data *map[K]S) {
	c.data.Store(data)
}
